package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func cmdSTORE(ctx *context.Context, cmd command.Command) error {
	src := strings.TrimSpace(cmd.Args)
	to := strings.TrimSpace(cmd.Clauses["TO"])
	if to == "" {
		return fail(ctx, "STORE requires TO")
	}
	// TO may be "NAME" only; take first token as the variable name.
	name := strings.Fields(to)[0]
	return storeExpr(ctx, src, name)
}

func tryAssignment(ctx *context.Context, cmd command.Command) (handled bool, err error) {
	args := strings.TrimSpace(cmd.Args)
	if !strings.HasPrefix(args, "=") {
		return false, nil
	}
	name := cmd.Verb
	if name == "" {
		return false, nil
	}
	src := strings.TrimSpace(args[1:])
	return true, storeExpr(ctx, src, name)
}

func storeExpr(ctx *context.Context, src, name string) error {
	src = strings.TrimSpace(src)
	if src == "" {
		return fail(ctx, "missing expression")
	}
	tree, errs := expr.Parse(src)
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

func cmdRELEASE(ctx *context.Context, cmd command.Command) error {
	args := strings.TrimSpace(cmd.Args)
	if args == "" {
		return fail(ctx, "RELEASE requires a name or ALL")
	}
	up := strings.ToUpper(args)
	fields := strings.Fields(up)
	if fields[0] == "MODULE" {
		name := ""
		if len(fields) > 1 {
			name = strings.Join(fields[1:], " ")
		}
		return releaseModule(ctx, name)
	}
	if fields[0] == "ALL" {
		if len(fields) == 1 {
			ctx.Symbols.ReleaseAll()
			return nil
		}
		if len(fields) >= 3 && fields[1] == "LIKE" {
			ctx.Symbols.ReleaseLike(fields[2], true)
			return nil
		}
		if len(fields) >= 3 && fields[1] == "EXCEPT" {
			ctx.Symbols.ReleaseLike(fields[2], false)
			return nil
		}
		return fail(ctx, "invalid RELEASE ALL")
	}
	for _, name := range strings.Split(args, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			ctx.Symbols.Release(name)
		}
	}
	return nil
}

func cmdDisplayMemory(ctx *context.Context, _ command.Command) error {
	return printMemory(ctx)
}

func printMemory(ctx *context.Context) error {
	for _, name := range ctx.Symbols.Names() {
		v, _ := ctx.Symbols.Get(name)
		fmt.Fprintf(ctx.Out(), "%s  %s  %s\n", name, typeLetter(v), v.String())
	}
	return nil
}

func typeLetter(v expr.Value) string {
	switch v.Kind {
	case expr.KindNumber:
		return "N"
	case expr.KindLogical:
		return "L"
	case expr.KindDate:
		return "D"
	case expr.KindMemo:
		return "M"
	default:
		return "C"
	}
}

func isMemoryList(cmd command.Command) bool {
	return strings.EqualFold(strings.TrimSpace(cmd.Args), "MEMORY")
}
