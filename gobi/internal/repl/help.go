package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func (m *Mux) cmdHELP(ctx *context.Context, cmd command.Command) error {
	args := strings.TrimSpace(cmd.Args)
	width := helpWidth(ctx)

	if args == "" {
		if term.IsTerminalWriter(ctx.Stdout) {
			return m.helpBrowser(ctx, "")
		}
		fmt.Fprint(ctx.Out(), help.FormatCatalog(width))
		return nil
	}

	topic := strings.ToUpper(strings.TrimSpace(args))
	t, ok := help.Lookup(topic)
	if !ok {
		fields := strings.Fields(topic)
		if len(fields) == 0 {
			return fail(ctx, "no help for %s", topic)
		}
		t, ok = help.Lookup(fields[0])
		if !ok {
			return fail(ctx, "no help for %s", topic)
		}
	}
	if term.IsTerminalWriter(ctx.Stdout) {
		return m.helpBrowser(ctx, t.Name)
	}
	fmt.Fprint(ctx.Out(), help.FormatTopic(t, width))
	return nil
}

func helpWidth(ctx *context.Context) int {
	if ctx.Screen != nil && ctx.Screen.Cols >= 40 {
		return ctx.Screen.Cols
	}
	return 80
}
