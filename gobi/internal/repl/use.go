package repl

import (
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func cmdUSE(ctx *context.Context, cmd command.Command) error {
	name, alias, indexes, err := parseUseIndexes(cmd.Args)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if err := ctx.Use(name, alias); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if len(indexes) > 0 {
		if err := cmdSETINDEX(ctx, strings.Join(indexes, ",")); err != nil {
			_ = ctx.CloseArea(ctx.ActiveArea)
			return err
		}
	}
	return nil
}

var (
	errUseAliasMissing = errString("ALIAS requires a name")
	errUseExtra        = errString("invalid USE arguments")
)

type errString string

func (e errString) Error() string { return string(e) }

func cmdCLOSE(ctx *context.Context, cmd command.Command) error {
	arg := strings.ToUpper(strings.TrimSpace(cmd.Args))
	switch arg {
	case "DATABASES":
		if err := ctx.CloseDatabases(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		return nil
	case "INDEX":
		ctx.CloseIndexes()
		ctx.Active().IndexOrder = nil
		return nil
	case "FORMAT":
		ctx.CloseFormat()
		return nil
	case "ALTERNATE":
		if err := ctx.CloseAlternate(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		return nil
	case "ALL":
		if err := ctx.CloseAll(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		return nil
	case "":
		return fail(ctx, "CLOSE requires an argument")
	default:
		return fail(ctx, "invalid CLOSE argument: %s", arg)
	}
}
