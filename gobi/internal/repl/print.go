package repl

import (
	"fmt"
	"strconv"
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
			writeScreenLine(ctx, "", true)
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
		out = append(out, formatQuestionValue(ctx, tree, v))
	}
	writeScreenLine(ctx, strings.Join(out, " "), newline)
	return nil
}

func formatDisplayValue(ctx *context.Context, v expr.Value) string {
	if v.Kind == expr.KindNumber && ctx.Config.Fixed {
		return strconv.FormatFloat(v.Num, 'f', ctx.Config.Decimals, 64)
	}
	return v.String()
}

// formatQuestionValue formats ? / ?? output. A bare numeric field uses the
// field's Decimals (III Plus); SET FIXED still wins when on.
func formatQuestionValue(ctx *context.Context, tree expr.Expression, v expr.Value) string {
	if v.Kind == expr.KindNumber && ctx.Config.Fixed {
		return strconv.FormatFloat(v.Num, 'f', ctx.Config.Decimals, 64)
	}
	if v.Kind == expr.KindNumber {
		if id, ok := tree.(*expr.IdentNode); ok {
			a := ctx.Active()
			if a.HasTable() {
				fi := fieldIndex(a, id.Name)
				if fi >= 0 && a.Table.Header.Fields[fi].Type == 'N' {
					d := int(a.Table.Header.Fields[fi].Decimals)
					return strconv.FormatFloat(v.Num, 'f', d, 64)
				}
			}
		}
	}
	return v.String()
}

// writeScreenLine writes non-fullscreen text. SET PRINT ON sends it to the print
// sink once (with MARGIN) instead of doubling on stdout (023.14).
func writeScreenLine(ctx *context.Context, s string, newline bool) {
	w := ctx.Out()
	if ctx.Config.Print {
		w = printSink(ctx)
	}
	margin := ""
	if ctx.Config.Print && ctx.Config.Margin > 0 {
		margin = strings.Repeat(" ", ctx.Config.Margin)
	}
	if newline {
		fmt.Fprintln(w, margin+s)
	} else {
		fmt.Fprint(w, margin+s)
	}
}
