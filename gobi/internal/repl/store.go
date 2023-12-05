package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/internal/symbols"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func handleStore(ctx *context.Context, cmd Command) error {
	if cmd.ToClause == "" {
		return fmt.Errorf("*** STORE requires TO clause")
	}
	if strings.TrimSpace(cmd.Args) == "" {
		return fmt.Errorf("*** STORE requires an expression")
	}

	memvars, err := parseStoreMemvars(cmd.ToClause)
	if err != nil {
		return err
	}

	exp, err := parseStoreExpression(cmd.Args)
	if err != nil {
		return err
	}

	env := newReplEnvironment(ctx)
	obj, err := expr.Eval(exp, env)
	if err != nil {
		return fmt.Errorf("*** Evaluation error in STORE expression: %w", err)
	}

	value, err := objectToStoredValue(obj)
	if err != nil {
		return err
	}

	for _, name := range memvars {
		if err := symbols.ValidateName(name); err != nil {
			return fmt.Errorf("*** Invalid memory variable name %s: %v", name, err)
		}
		if err := ctx.Variables.Set(name, value); err != nil {
			return err
		}
	}

	talkPrint(ctx, "%s\r\n", obj.String())

	return nil
}

func parseStoreExpression(args string) (expr.Expression, error) {
	lexer := expr.NewLexer(strings.TrimSpace(args))
	parser := expr.NewParser(lexer)
	exp := parser.ParseExpression()
	if len(parser.Errors()) > 0 {
		return nil, fmt.Errorf("*** Syntax error in STORE expression: %s", strings.Join(parser.Errors(), "; "))
	}
	if exp == nil {
		return nil, fmt.Errorf("*** STORE requires an expression")
	}
	return exp, nil
}

func parseStoreMemvars(toClause string) ([]string, error) {
	toClause = strings.TrimSpace(toClause)
	if toClause == "" {
		return nil, fmt.Errorf("*** STORE requires TO clause")
	}

	parts := splitCommaOutsideParens(toClause)
	memvars := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		memvars = append(memvars, name)
	}

	if len(memvars) == 0 {
		return nil, fmt.Errorf("*** STORE requires at least one memory variable")
	}

	return memvars, nil
}

func objectToStoredValue(obj expr.Object) (interface{}, error) {
	switch o := obj.(type) {
	case *expr.NumberObject:
		return o.Value, nil
	case *expr.StringObject:
		return o.Value, nil
	case *expr.BooleanObject:
		return o.Value, nil
	default:
		return nil, fmt.Errorf("*** STORE expression returned unsupported type %T", obj)
	}
}
