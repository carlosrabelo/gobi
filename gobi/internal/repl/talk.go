package repl

import (
	"fmt"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// applySetTalk updates the TALK flag and prints the dBase confirmation line.
func applySetTalk(ctx *context.Context, parts []string) {
	ctx.Config.Talk = parseOnOff(parts)
	fmt.Fprintf(ctx.Stdout, "Talk: %s\r\n", onOffStr(ctx.Config.Talk))
}

// talkEnabled reports whether command status output should be shown.
func talkEnabled(ctx *context.Context) bool {
	return ctx != nil && ctx.Config != nil && ctx.Config.Talk
}

// talkPrint writes formatted status output when TALK is ON.
func talkPrint(ctx *context.Context, format string, args ...interface{}) {
	if talkEnabled(ctx) {
		fmt.Fprintf(ctx.Stdout, format, args...)
	}
}
