package repl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/ndx"
)

type sortEntry struct {
	key ndx.Key
	rec *dbf.Record
}

func handleSort(ctx *context.Context, cmd Command) error {
	expression, err := parseSortOnExpression(cmd.Args)
	if err != nil {
		return err
	}
	if cmd.ToClause == "" {
		return fmt.Errorf("*** SORT requires TO <filename>")
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	forExp, whileExp, err := parseForWhileClauses(cmd)
	if err != nil {
		return err
	}

	entries, sortHeader, err := collectSortEntries(ctx, area, expression, forExp, whileExp)
	if err != nil {
		return err
	}

	sort.SliceStable(entries, func(i, j int) bool {
		cmp, cmpErr := ndx.CompareKeys(sortHeader, entries[i].key, entries[j].key)
		if cmpErr != nil {
			return false
		}
		return cmp < 0
	})

	filePath := resolveDBFFilePath(ctx, cmd.ToClause)
	destroy, err := confirmDestroyExistingFile(ctx, ctx.StdinReader(), filePath)
	if err != nil {
		return err
	}
	if !destroy {
		return nil
	}

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("*** Could not create output file: %w", err)
	}
	defer f.Close()

	dstTbl, err := dbf.Create(f, area.Table.Fields)
	if err != nil {
		return fmt.Errorf("*** Error creating output database: %w", err)
	}

	for _, entry := range entries {
		values := make([]interface{}, len(area.Table.Fields))
		for j := range area.Table.Fields {
			decoded, err := entry.rec.DecodeField(area.Table, j)
			if err != nil {
				return fmt.Errorf("*** Error decoding field in sorted record: %w", err)
			}
			values[j] = decoded
		}

		rec, err := dbf.NewRecord(dstTbl, false, values)
		if err != nil {
			return fmt.Errorf("*** Error building sorted record: %w", err)
		}
		if _, err := dstTbl.AppendRecord(f, rec); err != nil {
			return fmt.Errorf("*** Error writing sorted record: %w", err)
		}
	}

	if err := dstTbl.Close(); err != nil {
		return fmt.Errorf("*** Error closing output database: %w", err)
	}

	talkPrint(ctx, "SORT ON %s TO %s\r\n", expression, filepath.Base(filePath))
	talkPrint(ctx, "%05d RECORDS SORTED\r\n", len(entries))

	return nil
}

func collectSortEntries(ctx *context.Context, area *context.WorkArea, expression string, forExp, whileExp expr.Expression) ([]sortEntry, *ndx.Header, error) {
	keyExp, err := parseReplExpression(expression)
	if err != nil {
		return nil, nil, err
	}

	rseeker, ok := area.Table.Underlying().(io.ReadSeeker)
	if !ok {
		return nil, nil, fmt.Errorf("*** Underlying database stream is not seekable")
	}

	savedRecordNo := area.RecordNo
	savedRecord := area.ActiveRecord
	defer func() {
		area.RecordNo = savedRecordNo
		area.ActiveRecord = savedRecord
	}()

	env := newReplEnvironment(ctx)
	state := keyScanState{keyType: ndx.KeyTypeCharacter}
	var entries []sortEntry

	srcTbl := area.Table
	recCount := int(srcTbl.Header.RecordCount)
	startRec := 0
	if whileExp != nil {
		startRec = area.RecordNo
	}

	for i := startRec; i < recCount; i++ {
		rec, err := srcTbl.ReadRecordAt(rseeker, i)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, fmt.Errorf("*** Error reading record %d: %w", i+1, err)
		}

		area.RecordNo = i
		area.ActiveRecord = rec

		if whileExp != nil {
			res, err := expr.Eval(whileExp, env)
			if err != nil {
				return nil, nil, fmt.Errorf("*** Evaluation error in WHILE clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return nil, nil, fmt.Errorf("*** WHILE clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				break
			}
		}

		if forExp != nil {
			res, err := expr.Eval(forExp, env)
			if err != nil {
				return nil, nil, fmt.Errorf("*** Evaluation error in FOR clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return nil, nil, fmt.Errorf("*** FOR clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				continue
			}
		}

		if rec.Deleted {
			continue
		}

		value, err := expr.Eval(keyExp, env)
		if err != nil {
			return nil, nil, fmt.Errorf("*** Evaluation error in sort expression: %w", err)
		}

		text, numeric, err := indexValueText(value)
		if err != nil {
			return nil, nil, err
		}
		if err := updateKeyScanState(&state, text, numeric); err != nil {
			return nil, nil, err
		}

		entries = append(entries, sortEntry{
			key: ndx.Key(text),
			rec: rec,
		})
	}

	if state.keyLength == 0 {
		state.keyLength = inferKeyLengthFromExpression(area.Table, expression)
	}

	header := ndx.NewHeaderForExpression(expression, state.keyType, state.keyLength)
	return entries, header, nil
}

func parseSortOnExpression(args string) (string, error) {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return "", fmt.Errorf("*** SORT requires ON <expression>")
	}
	upper := strings.ToUpper(trimmed)
	if !strings.HasPrefix(upper, "ON ") {
		return "", fmt.Errorf("*** SORT requires ON <expression>")
	}
	expression := strings.TrimSpace(trimmed[2:])
	if expression == "" {
		return "", fmt.Errorf("*** SORT requires ON <expression>")
	}
	return expression, nil
}
