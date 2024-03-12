package repl

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

// handleChange implements CHANGE [<scope>] FIELD <list> [FOR <expr>]: a
// line-mode field editor. For each record in scope it shows every listed
// field value and prompts CHANGE? for the substring to replace and TO? for
// the replacement, exactly like the dBase II line editor. An empty CHANGE?
// answer leaves the field untouched.
func handleChange(ctx *context.Context, cmd Command) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	scope, rest, err := parseScopeClause(cmd.Args)
	if err != nil {
		return err
	}

	fieldIdxs, err := parseChangeFields(area.Table, rest)
	if err != nil {
		return err
	}

	forExp, whileExp, err := parseForWhileClauses(cmd)
	if err != nil {
		return err
	}

	wseeker, ok := area.Table.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	tbl := area.Table
	recCount := int(tbl.Header.RecordCount)
	startRec, limit := scanRange(area, scope, forExp != nil, whileExp != nil, true)
	if !scope.explicit && forExp == nil && whileExp == nil && startRec >= recCount {
		return fmt.Errorf("*** Record number out of range")
	}

	env := newReplEnvironment(ctx)
	reader := ctx.StdinReader()
	scanned := 0

	for i := startRec; i < recCount; i++ {
		if limit > 0 && scanned >= limit {
			break
		}
		scanned++

		rec, err := tbl.ReadRecordAt(wseeker, i)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("*** Error reading record %d: %w", i, err)
		}

		area.RecordNo = i
		area.ActiveRecord = rec

		if whileExp != nil {
			keep, err := evalLogicalClause(whileExp, env, "WHILE")
			if err != nil {
				return err
			}
			if !keep {
				break
			}
		}

		if forExp != nil {
			match, err := evalLogicalClause(forExp, env, "FOR")
			if err != nil {
				return err
			}
			if !match {
				continue
			}
		}

		fmt.Fprintf(ctx.Stdout, "RECORD: %05d\r\n", i+1)

		updated, edited, stop, err := changeRecordFields(ctx, reader, tbl, rec, fieldIdxs)
		if err != nil {
			return err
		}

		if edited {
			if err := tbl.WriteRecordAt(wseeker, i, updated); err != nil {
				return fmt.Errorf("*** Error writing record %d: %w", i+1, err)
			}
			if err := syncOpenIndexesAfterReplace(ctx, area, i, rec); err != nil {
				return err
			}
			area.ActiveRecord = updated
		}

		if stop {
			break
		}
	}

	return nil
}

// parseChangeFields parses the mandatory FIELD <list> clause and resolves
// every comma-separated name against the table structure.
func parseChangeFields(tbl *dbf.Table, args string) ([]int, error) {
	word, rest := splitLeadingWord(args)
	switch strings.ToUpper(word) {
	case "FIELD", "FIELDS":
	default:
		return nil, fmt.Errorf("*** CHANGE requires a FIELD list")
	}

	var idxs []int
	for _, name := range strings.Split(rest, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		fd, idx := tbl.FieldByName(strings.ToUpper(name))
		if fd == nil {
			return nil, fmt.Errorf("*** Unknown field: %s", name)
		}
		idxs = append(idxs, idx)
	}
	if len(idxs) == 0 {
		return nil, fmt.Errorf("*** CHANGE requires a FIELD list")
	}
	return idxs, nil
}

// changeRecordFields runs the CHANGE?/TO? dialogue over the listed fields of
// one record. It returns the (possibly rebuilt) record, whether any field was
// edited, and whether input ended (EOF) so the caller should stop the scan.
func changeRecordFields(ctx *context.Context, reader interface {
	ReadString(delim byte) (string, error)
}, tbl *dbf.Table, rec *dbf.Record, fieldIdxs []int) (*dbf.Record, bool, bool, error) {
	edited := false

	for _, fieldIdx := range fieldIdxs {
		fd := tbl.Fields[fieldIdx]

		current, err := changeFieldText(tbl, rec, fieldIdx)
		if err != nil {
			return nil, edited, false, err
		}
		fmt.Fprintf(ctx.Stdout, "%s:  %s\r\n", fd.Name, current)

		fmt.Fprint(ctx.Stdout, "CHANGE? ")
		from, err := reader.ReadString('\n')
		if err != nil {
			return rec, edited, true, nil
		}
		from = strings.TrimRight(from, "\r\n")
		if from == "" {
			continue
		}

		fmt.Fprint(ctx.Stdout, "TO? ")
		to, err := reader.ReadString('\n')
		if err != nil {
			return rec, edited, true, nil
		}
		to = strings.TrimRight(to, "\r\n")

		if !strings.Contains(current, from) {
			fmt.Fprintf(ctx.Stdout, "*** %s not found\r\n", from)
			continue
		}

		newText := strings.Replace(current, from, to, 1)
		val, err := parseFieldInput(fd, newText)
		if err != nil {
			return nil, edited, false, err
		}

		rec, err = replaceRecordField(tbl, rec, fieldIdx, val)
		if err != nil {
			return nil, edited, false, fmt.Errorf("*** Error updating field %s: %w", fd.Name, err)
		}
		edited = true
	}

	return rec, edited, false, nil
}

// changeFieldText renders a field value as the editable text shown by CHANGE.
func changeFieldText(tbl *dbf.Table, rec *dbf.Record, fieldIdx int) (string, error) {
	val, err := rec.DecodeField(tbl, fieldIdx)
	if err != nil {
		return "", fmt.Errorf("*** Error decoding field %s: %w", tbl.Fields[fieldIdx].Name, err)
	}
	switch v := val.(type) {
	case string:
		return v, nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		if v {
			return "T", nil
		}
		return "F", nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// evalLogicalClause evaluates a FOR or WHILE expression to a boolean.
func evalLogicalClause(exp expr.Expression, env expr.Environment, clause string) (bool, error) {
	res, err := expr.Eval(exp, env)
	if err != nil {
		return false, fmt.Errorf("*** Evaluation error in %s clause: %w", clause, err)
	}
	boolRes, ok := res.(*expr.BooleanObject)
	if !ok {
		return false, fmt.Errorf("*** %s clause must evaluate to a logical value", clause)
	}
	return boolRes.Value, nil
}
