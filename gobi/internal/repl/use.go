package repl

import (
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func cmdUSE(ctx *context.Context, cmd command.Command) error {
	name, alias, err := parseUseArgs(cmd.Args)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if err := ctx.Use(name, alias); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return nil
}

func parseUseArgs(args string) (name, alias string, err error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", "", nil
	}
	tokens := strings.Fields(args)
	name = tokens[0]
	for i := 1; i < len(tokens); i++ {
		if strings.EqualFold(tokens[i], "ALIAS") {
			if i+1 >= len(tokens) {
				return "", "", errUseAliasMissing
			}
			alias = tokens[i+1]
			i++
			continue
		}
		if strings.EqualFold(tokens[i], "INDEX") {
			return "", "", errUseIndex
		}
		return "", "", errUseExtra
	}
	return name, alias, nil
}

var (
	errUseAliasMissing = errString("ALIAS requires a name")
	errUseIndex        = errString("INDEX is not supported yet")
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
