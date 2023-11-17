package repl

import (
	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func expandLineMacros(ctx *context.Context, line string) (string, error) {
	return expr.ExpandMacros(line, func(name string) (expr.Value, bool) {
		if ctx.Symbols == nil {
			return expr.Value{}, false
		}
		return ctx.Symbols.Get(name)
	})
}
