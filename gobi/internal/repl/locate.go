package repl

import (
	"fmt"
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func handleLocate(ctx *context.Context, cmd Command) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	if cmd.ForClause == "" {
		return fmt.Errorf("*** LOCATE requires FOR <condition>")
	}

	scope, rest, err := parseScopeClause(cmd.Args)
	if err != nil {
		return err
	}
	if rest != "" {
		return fmt.Errorf("*** Unsupported LOCATE scope: %s", rest)
	}

	startRec := 0
	endRec := 0 // whole file
	if scope.next > 0 {
		startRec = area.RecordNo
		endRec = area.RecordNo + scope.next
	}

	area.LocateFor = cmd.ForClause
	area.LocateWhile = cmd.WhileClause
	area.LocateActive = true
	area.LocateEnd = endRec

	return runLocateSearch(ctx, area, startRec)
}

func handleContinue(ctx *context.Context, cmd Command) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	if strings.TrimSpace(cmd.Args) != "" {
		return fmt.Errorf("*** CONTINUE does not accept arguments")
	}

	if !area.LocateActive || area.LocateFor == "" {
		return fmt.Errorf("*** No active LOCATE command")
	}

	startRec := area.RecordNo + 1
	return runLocateSearch(ctx, area, startRec)
}

func runLocateSearch(ctx *context.Context, area *context.WorkArea, startRec int) error {
	forExp, whileExp, err := parseLocateClauses(area.LocateFor, area.LocateWhile)
	if err != nil {
		return err
	}

	rseeker, ok := area.Table.Underlying().(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("*** Underlying database stream is not seekable")
	}

	recCount := int(area.Table.Header.RecordCount)
	endRec := recCount
	if area.LocateEnd > 0 && area.LocateEnd < recCount {
		endRec = area.LocateEnd
	}
	env := newReplEnvironment(ctx)

	for i := startRec; i < endRec; i++ {
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

		area.Found = true
		talkPrint(ctx, "RECORD: %05d\r\n", i+1)
		return nil
	}

	area.RecordNo = recCount
	area.ActiveRecord = nil
	area.Found = false
	talkPrint(ctx, "End of Locate scope\r\n")
	return nil
}

func parseLocateClauses(forClause, whileClause string) (expr.Expression, expr.Expression, error) {
	var forExp expr.Expression
	if forClause != "" {
		lexer := expr.NewLexer(forClause)
		parser := expr.NewParser(lexer)
		forExp = parser.ParseExpression()
		if len(parser.Errors()) > 0 {
			return nil, nil, fmt.Errorf("*** Syntax error in FOR clause: %s", strings.Join(parser.Errors(), "; "))
		}
	}

	var whileExp expr.Expression
	if whileClause != "" {
		lexer := expr.NewLexer(whileClause)
		parser := expr.NewParser(lexer)
		whileExp = parser.ParseExpression()
		if len(parser.Errors()) > 0 {
			return nil, nil, fmt.Errorf("*** Syntax error in WHILE clause: %s", strings.Join(parser.Errors(), "; "))
		}
	}

	return forExp, whileExp, nil
}
