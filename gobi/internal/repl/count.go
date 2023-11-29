package repl

import (
	"fmt"
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func handleCount(ctx *context.Context, cmd Command) error {
	scope, rest, err := parseScopeClause(cmd.Args)
	if err != nil {
		return err
	}
	if rest != "" {
		return fmt.Errorf("*** Unsupported COUNT scope: %s", rest)
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	forExp, whileExp, err := parseForWhileClauses(cmd)
	if err != nil {
		return err
	}

	rseeker, ok := area.Table.Underlying().(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("*** Underlying database stream is not seekable")
	}

	savedRecordNo := area.RecordNo
	savedRecord := area.ActiveRecord

	recCount := int(area.Table.Header.RecordCount)
	startRec, limit := scanRange(area, scope, forExp != nil, whileExp != nil, false)
	env := newReplEnvironment(ctx)
	matched := 0
	scanned := 0

	for i := startRec; i < recCount; i++ {
		if limit > 0 && scanned >= limit {
			break
		}
		scanned++
		rec, err := area.Table.ReadRecordAt(rseeker, i)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("*** Error reading record %d: %w", i+1, err)
		}

		area.RecordNo = i
		area.ActiveRecord = rec

		if deletedHidden(ctx, rec) {
			continue
		}

		if whileExp != nil {
			res, err := expr.Eval(whileExp, env)
			if err != nil {
				area.RecordNo = savedRecordNo
				area.ActiveRecord = savedRecord
				return fmt.Errorf("*** Evaluation error in WHILE clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				area.RecordNo = savedRecordNo
				area.ActiveRecord = savedRecord
				return fmt.Errorf("*** WHILE clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				break
			}
		}

		if forExp != nil {
			res, err := expr.Eval(forExp, env)
			if err != nil {
				area.RecordNo = savedRecordNo
				area.ActiveRecord = savedRecord
				return fmt.Errorf("*** Evaluation error in FOR clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				area.RecordNo = savedRecordNo
				area.ActiveRecord = savedRecord
				return fmt.Errorf("*** FOR clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				continue
			}
		}

		matched++
	}

	area.RecordNo = savedRecordNo
	area.ActiveRecord = savedRecord

	if cmd.ToClause != "" {
		varName := strings.TrimSpace(cmd.ToClause)
		if err := ctx.Variables.Set(varName, float64(matched)); err != nil {
			return err
		}
	}

	talkPrint(ctx, "COUNT = %05d\r\n", matched)

	return nil
}
