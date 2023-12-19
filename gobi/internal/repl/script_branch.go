package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/script"
)

func evalLogicalExpression(ctx *context.Context, expression, label string) (bool, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return false, fmt.Errorf("*** %s requires a logical expression", label)
	}

	lexer := expr.NewLexer(expression)
	parser := expr.NewParser(lexer)
	exp := parser.ParseExpression()
	if len(parser.Errors()) > 0 {
		return false, fmt.Errorf("*** Syntax error in %s expression: %s", label, strings.Join(parser.Errors(), "; "))
	}
	if exp == nil {
		return false, fmt.Errorf("*** %s requires a logical expression", label)
	}

	res, err := expr.Eval(exp, newReplEnvironment(ctx))
	if err != nil {
		return false, fmt.Errorf("*** Evaluation error in %s expression: %w", label, err)
	}

	boolRes, ok := res.(*expr.BooleanObject)
	if !ok {
		return false, fmt.Errorf("*** %s expression must evaluate to .T. or .F.", label)
	}

	return boolRes.Value, nil
}

func finishAdvance(ctrl *script.Controller) bool {
	if ctrl.Advance() {
		return false
	}
	return ctrl.Depth() <= 1
}

func executeScriptLine(ctx *context.Context, ctrl *script.Controller, line script.Line) (stop bool, err error) {
	prog := ctrl.Program()
	if prog == nil {
		return false, fmt.Errorf("*** script: no active program")
	}

	switch line.Command.Verb {
	case "RETURN":
		return true, nil
	case "IF":
		block, ok := prog.IfBlockAt(ctrl.Index())
		if !ok {
			return false, fmt.Errorf("*** Error in %s, line %d: unmatched IF block", prog.Path, line.Number)
		}

		truthy, err := evalLogicalExpression(ctx, line.Command.Args, "IF")
		if err != nil {
			return false, fmt.Errorf("*** Error in %s, line %d: %w", prog.Path, line.Number, err)
		}

		if truthy {
			if finishAdvance(ctrl) {
				return true, nil
			}
		} else if block.ElseIndex >= 0 {
			if err := ctrl.SetIndex(block.ElseIndex + 1); err != nil {
				return false, err
			}
		} else if err := ctrl.SetIndex(block.EndIndex + 1); err != nil {
			return false, err
		}
		return false, nil
	case "ELSE":
		skip, ok := prog.SkipAfterElse(ctrl.Index())
		if !ok {
			return false, fmt.Errorf("*** Error in %s, line %d: ELSE without matching IF", prog.Path, line.Number)
		}
		if err := ctrl.SetIndex(skip); err != nil {
			return false, err
		}
		return false, nil
	case "ENDIF":
		if finishAdvance(ctrl) {
			return true, nil
		}
		return false, nil
	}

	if err := scriptDispatch(ctx, line.Command); err != nil {
		return false, fmt.Errorf("*** Error in %s, line %d: %w", prog.Path, line.Number, err)
	}

	if finishAdvance(ctrl) {
		return true, nil
	}

	return false, nil
}
