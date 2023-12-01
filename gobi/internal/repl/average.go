package repl

import (
	"fmt"
	"io"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func handleAverage(ctx *context.Context, cmd Command) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	expressions, err := parseAggregateExpressions(cmd.Args, "AVERAGE")
	if err != nil {
		return err
	}

	memvars, err := parseAggregateMemvars(cmd.ToClause, len(expressions), "AVERAGE")
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

	savedRecordNo := area.RecordNo
	savedRecord := area.ActiveRecord

	recCount := int(area.Table.Header.RecordCount)
	env := newReplEnvironment(ctx)
	totals := make([]float64, len(expressions))
	counts := make([]int, len(expressions))

	for i := 0; i < recCount; i++ {
		rec, err := area.Table.ReadRecordAt(rseeker, i)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("*** Error reading record %d: %w", i+1, err)
		}
		if rec.Deleted {
			continue
		}

		area.RecordNo = i
		area.ActiveRecord = rec

		if whileExp != nil {
			res, err := expr.Eval(whileExp, env)
			if err != nil {
				restoreRecordPointer(area, savedRecordNo, savedRecord)
				return fmt.Errorf("*** Evaluation error in WHILE clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				restoreRecordPointer(area, savedRecordNo, savedRecord)
				return fmt.Errorf("*** WHILE clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				break
			}
		}

		if forExp != nil {
			res, err := expr.Eval(forExp, env)
			if err != nil {
				restoreRecordPointer(area, savedRecordNo, savedRecord)
				return fmt.Errorf("*** Evaluation error in FOR clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				restoreRecordPointer(area, savedRecordNo, savedRecord)
				return fmt.Errorf("*** FOR clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				continue
			}
		}

		for j, avgExp := range expressions {
			val, err := evalNumericExpression(env, avgExp)
			if err != nil {
				restoreRecordPointer(area, savedRecordNo, savedRecord)
				return fmt.Errorf("*** Error evaluating AVERAGE expression: %w", err)
			}
			totals[j] += val
			counts[j]++
		}
	}

	restoreRecordPointer(area, savedRecordNo, savedRecord)

	averages := make([]float64, len(expressions))
	for i := range averages {
		if counts[i] > 0 {
			averages[i] = totals[i] / float64(counts[i])
		}
	}

	for i, name := range memvars {
		if err := ctx.Variables.Set(name, averages[i]); err != nil {
			return err
		}
	}

	for _, average := range averages {
		talkPrint(ctx, "%12.2f\r\n", average)
	}

	return nil
}
