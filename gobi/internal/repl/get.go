package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/internal/symbols"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func handleAtGet(ctx *context.Context, cmd Command) error {
	rowStr, colStr, target, picture, err := parseGetArgs(cmd.Args)
	if err != nil {
		return err
	}

	if err := symbols.ValidateName(target); err != nil {
		return fmt.Errorf("*** Invalid GET target %s: %v", target, err)
	}

	row, col, err := evaluateAtCoordinates(ctx, rowStr, colStr)
	if err != nil {
		return err
	}

	name := strings.ToUpper(strings.TrimSpace(target))
	display := formatGetDisplay(resolveGetDisplayValue(ctx, name), picture)
	ctx.Screen.RegisterGet(row, col, name, picture)
	ctx.Screen.WriteAt(row, col, display)
	return paintScreenText(ctx, row, col, display)
}

func parseGetArgs(args string) (rowExpr, colExpr, target, picture string, err error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", "", "", "", fmt.Errorf("*** @ GET requires row, column, and variable")
	}

	rowExpr, colExpr, getPart, err := parseAtCoordinates(args, "GET")
	if err != nil {
		return "", "", "", "", err
	}
	if getPart == "" {
		return "", "", "", "", fmt.Errorf("*** @ GET requires a variable or field name")
	}

	targetPart, picturePart, hasPicture := splitAtKeyword(getPart, "PICTURE")
	if hasPicture {
		target = strings.TrimSpace(targetPart)
		picture, err = parsePictureTemplate(picturePart)
		if err != nil {
			return "", "", "", "", err
		}
	} else {
		target = strings.TrimSpace(getPart)
	}
	if target == "" {
		return "", "", "", "", fmt.Errorf("*** @ GET requires a variable or field name")
	}

	return rowExpr, colExpr, target, picture, nil
}

func parsePictureTemplate(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("*** @ GET PICTURE requires a template")
	}

	tokens := tokenize(s)
	if len(tokens) == 0 {
		return "", fmt.Errorf("*** @ GET PICTURE requires a template")
	}

	tok := tokens[0]
	if len(tok) >= 2 {
		switch tok[0] {
		case '"':
			if tok[len(tok)-1] == '"' {
				return tok[1 : len(tok)-1], nil
			}
		case '\'':
			if tok[len(tok)-1] == '\'' {
				return tok[1 : len(tok)-1], nil
			}
		}
	}

	return "", fmt.Errorf("*** @ GET PICTURE requires a quoted template")
}

func resolveGetDisplayValue(ctx *context.Context, name string) string {
	env := newReplEnvironment(ctx)
	obj, err := expr.Eval(&expr.Identifier{Name: name}, env)
	if err != nil {
		return ""
	}
	return obj.String()
}

func formatGetDisplay(value, picture string) string {
	if picture == "" {
		return value
	}

	width := len(picture)
	if len(value) >= width {
		return value[:width]
	}
	return value + strings.Repeat(" ", width-len(value))
}
