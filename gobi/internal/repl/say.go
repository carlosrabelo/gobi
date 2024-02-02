package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func handleAtSay(ctx *context.Context, cmd Command) error {
	rowStr, colStr, sayStr, err := parseSayArgs(cmd.Args)
	if err != nil {
		return err
	}

	row, col, err := evaluateAtCoordinates(ctx, rowStr, colStr)
	if err != nil {
		return err
	}

	sayExpr, err := parseAtExpression("SAY expression", sayStr)
	if err != nil {
		return err
	}

	env := newReplEnvironment(ctx)
	obj, err := expr.Eval(sayExpr, env)
	if err != nil {
		return fmt.Errorf("*** Evaluation error in SAY expression: %w", err)
	}

	ctx.Screen.WriteAt(row, col, obj.String())
	return paintScreenText(ctx, row, col, obj.String())
}

func parseSayArgs(args string) (rowExpr, colExpr, sayExpr string, err error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", "", "", fmt.Errorf("*** @ SAY requires row, column, and expression")
	}

	rowExpr, colExpr, sayExpr, err = parseAtCoordinates(args, "SAY")
	if err != nil {
		return "", "", "", err
	}
	if strings.TrimSpace(sayExpr) == "" {
		return "", "", "", fmt.Errorf("*** @ SAY requires an expression")
	}

	return rowExpr, colExpr, sayExpr, nil
}
