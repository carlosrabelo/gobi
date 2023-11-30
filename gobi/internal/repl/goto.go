package repl

import (
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func cmdGOTO(ctx *context.Context, cmd command.Command) error {
	return runGo(ctx, goArg(cmd))
}

func cmdGO(ctx *context.Context, cmd command.Command) error {
	return runGo(ctx, goArg(cmd))
}

// goArg merges Args with a TO clause (Split treats "GO TO n" as verb GO + clause TO).
func goArg(cmd command.Command) string {
	if to, ok := cmd.Clauses["TO"]; ok && strings.TrimSpace(to) != "" {
		return strings.TrimSpace(to)
	}
	return strings.TrimSpace(cmd.Args)
}

func runGo(ctx *context.Context, args string) error {
	args = strings.TrimSpace(args)
	if args == "" {
		return fail(ctx, "GO requires an argument")
	}
	fields := strings.Fields(args)
	if len(fields) >= 1 && strings.EqualFold(fields[0], "TO") {
		fields = fields[1:]
	}
	if len(fields) != 1 {
		return fail(ctx, "invalid GO arguments")
	}
	word := fields[0]
	switch strings.ToUpper(word) {
	case "TOP":
		if err := ctx.GoTop(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		return hookAfterMove(ctx)
	case "BOTTOM":
		if err := ctx.GoBottom(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		return hookAfterMove(ctx)
	}
	n, err := strconv.Atoi(word)
	if err != nil {
		return fail(ctx, "invalid record number: %s", word)
	}
	if err := ctx.Goto(n); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return hookAfterMove(ctx)
}

func cmdSKIP(ctx *context.Context, cmd command.Command) error {
	args := strings.TrimSpace(cmd.Args)
	n := 1
	if args != "" {
		var err error
		n, err = strconv.Atoi(args)
		if err != nil {
			return fail(ctx, "invalid SKIP count: %s", args)
		}
	}
	if err := ctx.Skip(n); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return hookAfterMove(ctx)
}
