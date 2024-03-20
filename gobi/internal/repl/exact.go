package repl

import (
	"fmt"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// applySetExact implements SET EXACT ON/OFF. With EXACT OFF (the dBase II
// default) string comparisons stop at the end of the right operand, so
// NAME = 'Sm' matches 'Smith'; with EXACT ON the whole strings must match.
func applySetExact(ctx *context.Context, parts []string) {
	ctx.Config.Exact = parseOnOff(parts)
	fmt.Fprintf(ctx.Stdout, "Exact: %s\r\n", onOffStr(ctx.Config.Exact))
}
