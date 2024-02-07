package repl

import (
	"fmt"
	"io"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// applySetBell updates the BELL flag and prints the dBase confirmation line.
func applySetBell(ctx *context.Context, parts []string) {
	ctx.Config.Bell = parseOnOff(parts)
	fmt.Fprintf(ctx.Stdout, "Bell: %s\r\n", onOffStr(ctx.Config.Bell))
}

// bellEnabled reports whether the terminal bell should sound on validation errors.
func bellEnabled(ctx *context.Context) bool {
	return ctx != nil && ctx.Config != nil && ctx.Config.Bell
}

// bellAlert rings the terminal bell when BELL is ON.
func bellAlert(ctx *context.Context) {
	if !bellEnabled(ctx) {
		return
	}
	bellAlertTo(ctx.Stdout)
}

func bellAlertTo(w io.Writer) {
	if w == nil {
		return
	}
	_ = term.RingBell(w)
}

// reportValidationError prints an input validation message and rings the bell when enabled.
func reportValidationError(ctx *context.Context, err error) {
	if err == nil {
		return
	}
	bellAlert(ctx)
	fmt.Fprintln(ctx.Stderr, err.Error())
}
