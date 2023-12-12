package repl

import (
	"os/exec"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func cmdRUN(ctx *context.Context, cmd command.Command) error {
	text := strings.TrimSpace(cmd.Args)
	if text == "" {
		return fail(ctx, "RUN requires a command")
	}
	c := exec.Command("sh", "-c", text)
	c.Stdout = ctx.Out()
	c.Stderr = ctx.Stderr
	c.Stdin = ctx.Stdin
	err := c.Run()
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return nil
}

func cmdBang(ctx *context.Context, cmd command.Command) error {
	return cmdRUN(ctx, cmd)
}
