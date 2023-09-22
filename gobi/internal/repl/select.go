package repl

import (
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func cmdSELECT(ctx *context.Context, cmd command.Command) error {
	name := strings.TrimSpace(cmd.Args)
	if name == "" {
		return fail(ctx, "SELECT requires a work area")
	}
	if err := ctx.Select(name); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return nil
}
