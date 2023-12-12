package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func cmdQuestion(ctx *context.Context, cmd command.Command) error {
	return printExprs(ctx, cmd.Args, true)
}

func cmdQuestionQuestion(ctx *context.Context, cmd command.Command) error {
	return printExprs(ctx, cmd.Args, false)
}

func printExprs(ctx *context.Context, args string, newline bool) error {
	parts := splitCommaExprs(strings.TrimSpace(args))
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		if newline {
			fmt.Fprintln(ctx.Out())
		}
		return nil
	}
	env := newEnv(ctx)
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		tree, errs := expr.Parse(p)
		if len(errs) > 0 || tree == nil {
			msg := "invalid expression"
			if len(errs) > 0 {
				msg = errs[0]
			}
			return fail(ctx, "%s", msg)
		}
		v, err := expr.Eval(tree, env)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		out = append(out, v.String())
	}
	text := strings.Join(out, " ")
	if newline {
		fmt.Fprintln(ctx.Out(), text)
	} else {
		fmt.Fprint(ctx.Out(), text)
	}
	return nil
}
