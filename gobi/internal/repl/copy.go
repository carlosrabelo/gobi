package repl

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func handleCopy(ctx *context.Context, cmd Command) error {
	if cmd.ToClause == "" {
		return fmt.Errorf("*** COPY requires TO <filename>")
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	outFields, fieldIdxs, err := parseCopyFields(area.Table, cmd.Args)
	if err != nil {
		return err
	}

	forExp, whileExp, err := parseForWhileClauses(cmd)
	if err != nil {
		return err
	}

	rseeker, ok := area.Table.Underlying().(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("*** Underlying database stream is not seekable")
	}

	filePath := resolveDBFFilePath(ctx, cmd.ToClause)
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("*** Could not create output file: %w", err)
	}
	defer f.Close()

	dstTbl, err := dbf.Create(f, outFields)
	if err != nil {
		return fmt.Errorf("*** Error creating output database: %w", err)
	}

	srcTbl := area.Table
	recCount := int(srcTbl.Header.RecordCount)
	startRec := 0
	if whileExp != nil {
		startRec = area.RecordNo
	}

	env := newReplEnvironment(ctx)
	copied := 0
	lastCopied := -1
	var lastRecord *dbf.Record

	for i := startRec; i < recCount; i++ {
		rec, err := srcTbl.ReadRecordAt(rseeker, i)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("*** Error reading record %d: %w", i, err)
		}

		area.RecordNo = i
		area.ActiveRecord = rec

		if whileExp != nil {
			res, err := expr.Eval(whileExp, env)
			if err != nil {
				return fmt.Errorf("*** Evaluation error in WHILE clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return fmt.Errorf("*** WHILE clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				break
			}
		}

		if forExp != nil {
			res, err := expr.Eval(forExp, env)
			if err != nil {
				return fmt.Errorf("*** Evaluation error in FOR clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return fmt.Errorf("*** FOR clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				continue
			}
		}

		if rec.Deleted {
			continue
		}

		values := make([]interface{}, len(outFields))
		for j, srcIdx := range fieldIdxs {
			decoded, err := rec.DecodeField(srcTbl, srcIdx)
			if err != nil {
				return fmt.Errorf("*** Error decoding field in record %d: %w", i+1, err)
			}
			values[j] = decoded
		}

		newRec, err := dbf.NewRecord(dstTbl, false, values)
		if err != nil {
			return fmt.Errorf("*** Error building copied record %d: %w", i+1, err)
		}

		if _, err := dstTbl.AppendRecord(f, newRec); err != nil {
			return fmt.Errorf("*** Error writing copied record %d: %w", i+1, err)
		}

		copied++
		lastCopied = i
		lastRecord = rec
	}

	if lastCopied >= 0 {
		area.RecordNo = lastCopied
		area.ActiveRecord = lastRecord
	}

	if err := dstTbl.Close(); err != nil {
		return fmt.Errorf("*** Error closing output database: %w", err)
	}

	talkPrint(ctx, "%05d RECORDS COPIED\r\n", copied)

	return nil
}

func parseCopyFields(tbl *dbf.Table, args string) ([]dbf.FieldDescriptor, []int, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		idxs := make([]int, len(tbl.Fields))
		for i := range tbl.Fields {
			idxs[i] = i
		}
		return tbl.Fields, idxs, nil
	}

	tokens := tokenize(args)
	start := 0
	if len(tokens) > 0 && strings.ToUpper(tokens[0]) == "FIELD" {
		start = 1
	}
	if start >= len(tokens) {
		return nil, nil, fmt.Errorf("*** COPY FIELD requires a field list")
	}

	parts := splitCommaOutsideParens(joinTokens(tokens[start:]))
	if len(parts) == 0 {
		return nil, nil, fmt.Errorf("*** COPY FIELD requires a field list")
	}

	outFields := make([]dbf.FieldDescriptor, 0, len(parts))
	idxs := make([]int, 0, len(parts))
	for _, part := range parts {
		name := strings.ToUpper(strings.TrimSpace(part))
		if name == "" {
			continue
		}
		fd, idx := tbl.FieldByName(name)
		if fd == nil {
			return nil, nil, fmt.Errorf("*** Unknown field: %s", name)
		}
		outFields = append(outFields, *fd)
		idxs = append(idxs, idx)
	}

	if len(outFields) == 0 {
		return nil, nil, fmt.Errorf("*** COPY FIELD requires a field list")
	}

	return outFields, idxs, nil
}
