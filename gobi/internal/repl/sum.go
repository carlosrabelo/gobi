package repl

import (
	"fmt"
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

const maxAggregateExpressions = 5

func handleSum(ctx *context.Context, cmd Command) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	scope, rest, err := parseScopeClause(cmd.Args)
	if err != nil {
		return err
	}

	expressions, err := parseAggregateExpressions(rest, "SUM")
	if err != nil {
		return err
	}

	memvars, err := parseAggregateMemvars(cmd.ToClause, len(expressions), "SUM")
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
	startRec, limit := scanRange(area, scope, forExp != nil, whileExp != nil, false)
	env := newReplEnvironment(ctx)
	totals := make([]float64, len(expressions))
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

		for j, sumExp := range expressions {
			val, err := evalNumericExpression(env, sumExp)
			if err != nil {
				restoreRecordPointer(area, savedRecordNo, savedRecord)
				return fmt.Errorf("*** Error evaluating SUM expression: %w", err)
			}
			totals[j] += val
		}
	}

	restoreRecordPointer(area, savedRecordNo, savedRecord)

	for i, name := range memvars {
		if err := ctx.Variables.Set(name, totals[i]); err != nil {
			return err
		}
	}

	for _, total := range totals {
		talkPrint(ctx, "%12.2f\r\n", total)
	}

	return nil
}

func parseAggregateExpressions(args, verb string) ([]expr.Expression, error) {
	fieldArgs, err := stripAggregateScope(args, verb)
	if err != nil {
		return nil, err
	}
	if fieldArgs == "" {
		return nil, fmt.Errorf("*** %s requires at least one numeric expression", verb)
	}

	parts := splitCommaOutsideParens(fieldArgs)
	expressions := make([]expr.Expression, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		lexer := expr.NewLexer(part)
		parser := expr.NewParser(lexer)
		sumExp := parser.ParseExpression()
		if len(parser.Errors()) > 0 {
			return nil, fmt.Errorf("*** Syntax error in %s expression: %s", verb, strings.Join(parser.Errors(), "; "))
		}
		expressions = append(expressions, sumExp)
	}

	if len(expressions) == 0 {
		return nil, fmt.Errorf("*** %s requires at least one numeric expression", verb)
	}
	if len(expressions) > maxAggregateExpressions {
		return nil, fmt.Errorf("*** MORE THAN 5 FIELDS TO %s", strings.ToUpper(verb))
	}

	return expressions, nil
}

func parseAggregateMemvars(toClause string, exprCount int, verb string) ([]string, error) {
	toClause = strings.TrimSpace(toClause)
	if toClause == "" {
		return nil, nil
	}

	parts := splitCommaOutsideParens(toClause)
	memvars := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.ToUpper(strings.TrimSpace(part))
		if name == "" {
			continue
		}
		memvars = append(memvars, name)
	}

	if len(memvars) != exprCount {
		return nil, fmt.Errorf("*** %s TO requires one memory variable per expression", verb)
	}

	return memvars, nil
}

func stripAggregateScope(args, verb string) (string, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", nil
	}

	tokens := strings.Fields(args)
	if len(tokens) == 1 && strings.ToUpper(tokens[0]) == "ALL" {
		return "", fmt.Errorf("*** %s requires at least one numeric expression", verb)
	}
	if len(tokens) > 1 && strings.ToUpper(tokens[len(tokens)-1]) == "ALL" {
		return strings.TrimSpace(strings.Join(tokens[:len(tokens)-1], " ")), nil
	}

	upper := strings.ToUpper(args)
	if strings.HasPrefix(upper, "NEXT ") || upper == "REST" || strings.HasPrefix(upper, "REST ") {
		return "", fmt.Errorf("*** Unsupported %s scope: %s", verb, args)
	}

	return args, nil
}

func restoreRecordPointer(area *context.WorkArea, recordNo int, record *dbf.Record) {
	area.RecordNo = recordNo
	area.ActiveRecord = record
}

func evalNumericExpression(env expr.Environment, sumExp expr.Expression) (float64, error) {
	res, err := expr.Eval(sumExp, env)
	if err != nil {
		return 0, err
	}
	num, ok := res.(*expr.NumberObject)
	if !ok {
		return 0, fmt.Errorf("non-numeric expression")
	}
	return num.Value, nil
}
