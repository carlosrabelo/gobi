package repl

import (
	"fmt"
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func cmdACCEPT(ctx *context.Context, cmd command.Command) error {
	ctx.KeyRead = true
	prompt, name, err := parsePromptTO(cmd)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if prompt != "" {
		fmt.Fprint(ctx.Out(), prompt)
	}
	line, err := readInteractiveLine(ctx)
	if err != nil && err != io.EOF {
		return fail(ctx, "%s", err.Error())
	}
	if err := ctx.Symbols.Set(name, expr.StringVal(line)); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return nil
}

func cmdINPUT(ctx *context.Context, cmd command.Command) error {
	ctx.KeyRead = true
	prompt, name, err := parsePromptTO(cmd)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if prompt != "" {
		fmt.Fprint(ctx.Out(), prompt)
	}
	line, err := readInteractiveLine(ctx)
	if err != nil && err != io.EOF {
		return fail(ctx, "%s", err.Error())
	}
	tree, errs := expr.Parse(line)
	if len(errs) > 0 || tree == nil {
		msg := "invalid expression"
		if len(errs) > 0 {
			msg = errs[0]
		}
		return fail(ctx, "%s", msg)
	}
	v, err := expr.Eval(tree, newEnv(ctx))
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if err := ctx.Symbols.Set(name, v); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return nil
}

func cmdWAIT(ctx *context.Context, cmd command.Command) error {
	ctx.KeyRead = true
	prompt := "WAITING"
	to := strings.TrimSpace(cmd.Clauses["TO"])
	args := strings.TrimSpace(cmd.Args)
	if args != "" {
		prompt = strings.Trim(args, `"'`)
	}
	fmt.Fprint(ctx.Out(), prompt)
	k, err := readInteractiveKey(ctx)
	if err != nil && err != io.EOF {
		return fail(ctx, "%s", err.Error())
	}
	if to != "" {
		ch := ""
		if k == term.KeyEnter {
			ch = "\r"
		} else if k > 0 && k < 127 {
			ch = string(byte(k))
		}
		if err := ctx.Symbols.Set(strings.Fields(to)[0], expr.StringVal(ch)); err != nil {
			return fail(ctx, "%s", err.Error())
		}
	}
	return nil
}

func cmdREMARK(ctx *context.Context, cmd command.Command) error {
	fmt.Fprintln(ctx.Out(), cmd.Args)
	return nil
}

func parsePromptTO(cmd command.Command) (prompt, name string, err error) {
	to := strings.TrimSpace(cmd.Clauses["TO"])
	if to == "" {
		return "", "", errString("requires TO")
	}
	name = strings.Fields(to)[0]
	prompt = strings.TrimSpace(cmd.Args)
	prompt = strings.Trim(prompt, `"'`)
	return prompt, name, nil
}
