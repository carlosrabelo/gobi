package repl

import (
	"fmt"
	"io"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// applySetIntensity updates the INTENSITY flag and prints the dBase confirmation line.
func applySetIntensity(ctx *context.Context, parts []string) {
	ctx.Config.Intensity = parseOnOff(parts)
	fmt.Fprintf(ctx.Stdout, "Intensity: %s\r\n", onOffStr(ctx.Config.Intensity))
}

// intensityEnabled reports whether reverse-video highlighting should be used.
func intensityEnabled(ctx *context.Context) bool {
	return ctx != nil && ctx.Config != nil && ctx.Config.Intensity
}

// presentScreen renders the screen buffer with optional reverse-video highlights.
func presentScreen(ctx *context.Context, w io.Writer, highlights []term.Highlight) error {
	return ctx.Screen.PresentWith(w, term.PresentOptions{
		Intensity:  intensityEnabled(ctx),
		Highlights: highlights,
	})
}
