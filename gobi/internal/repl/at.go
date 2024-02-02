package repl

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func handleAt(ctx *context.Context, cmd Command) error {
	args := strings.TrimSpace(cmd.Args)
	if args == "" {
		return fmt.Errorf("*** @ requires SAY or GET clause")
	}

	if _, _, ok := splitAtKeyword(args, "GET"); ok {
		return handleAtGet(ctx, cmd)
	}
	if _, _, ok := splitAtKeyword(args, "SAY"); ok {
		return handleAtSay(ctx, cmd)
	}
	return fmt.Errorf("*** @ requires SAY or GET clause")
}

func evaluateAtCoordinates(ctx *context.Context, rowStr, colStr string) (int, int, error) {
	env := newReplEnvironment(ctx)

	rowExpr, err := parseAtExpression("row", rowStr)
	if err != nil {
		return 0, 0, err
	}
	colExpr, err := parseAtExpression("column", colStr)
	if err != nil {
		return 0, 0, err
	}

	row, err := evalNumericExpression(env, rowExpr)
	if err != nil {
		return 0, 0, fmt.Errorf("*** Evaluation error in @ row: %w", err)
	}
	col, err := evalNumericExpression(env, colExpr)
	if err != nil {
		return 0, 0, fmt.Errorf("*** Evaluation error in @ column: %w", err)
	}
	if row < 0 || col < 0 {
		return 0, 0, fmt.Errorf("*** @ coordinates must be non-negative")
	}

	return int(row), int(col), nil
}

func parseAtCoordinates(args, keyword string) (rowExpr, colExpr, rest string, err error) {
	coordPart, rest, ok := splitAtKeyword(args, keyword)
	if !ok {
		return "", "", "", fmt.Errorf("*** @ requires %s clause", keyword)
	}

	parts := splitCommaOutsideParens(coordPart)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", fmt.Errorf("*** @ %s requires row, column coordinates", keyword)
	}

	return parts[0], parts[1], strings.TrimSpace(rest), nil
}

func parseAtExpression(part, exprStr string) (expr.Expression, error) {
	lexer := expr.NewLexer(strings.TrimSpace(exprStr))
	parser := expr.NewParser(lexer)
	exp := parser.ParseExpression()
	if len(parser.Errors()) > 0 {
		return nil, fmt.Errorf("*** Syntax error in @ %s: %s", part, strings.Join(parser.Errors(), "; "))
	}
	if exp == nil {
		return nil, fmt.Errorf("*** @ %s requires an expression", part)
	}
	return exp, nil
}

func splitAtKeyword(s, keyword string) (before, after string, ok bool) {
	if keyword == "" {
		return "", "", false
	}

	kwLen := len(keyword)
	for i := 0; i <= len(s)-kwLen; i++ {
		if !strings.EqualFold(s[i:i+kwLen], keyword) {
			continue
		}

		beforeOK := i == 0 || unicode.IsSpace(rune(s[i-1]))
		afterEnd := i + kwLen
		afterOK := afterEnd >= len(s) || unicode.IsSpace(rune(s[afterEnd]))
		if !beforeOK || !afterOK {
			continue
		}

		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[afterEnd:]), true
	}

	return "", "", false
}
