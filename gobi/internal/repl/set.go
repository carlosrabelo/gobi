package repl

import (
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func cmdSET(ctx *context.Context, cmd command.Command) error {
	args := strings.Fields(cmd.Args)
	if len(args) == 0 {
		return fail(ctx, "SET requires an option")
	}
	name := strings.ToUpper(args[0])
	switch name {
	case "TALK", "INTENSITY", "BELL":
		if len(args) < 2 {
			return fail(ctx, "SET %s requires ON or OFF", name)
		}
		on, err := parseOnOff(args[1])
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		switch name {
		case "TALK":
			ctx.Config.Talk = on
		case "INTENSITY":
			ctx.Config.Intensity = on
		case "BELL":
			ctx.Config.Bell = on
		}
		return nil
	case "DEFAULT":
		path := strings.TrimSpace(cmd.Clauses["TO"])
		if path == "" {
			if len(args) >= 3 && strings.EqualFold(args[1], "TO") {
				path = strings.Join(args[2:], " ")
			} else {
				return fail(ctx, "SET DEFAULT requires TO")
			}
		}
		ctx.Config.Default = path
		return nil
	case "INDEX":
		list := strings.TrimSpace(cmd.Clauses["TO"])
		if list == "" && len(args) >= 2 && strings.EqualFold(args[1], "TO") {
			list = strings.Join(args[2:], " ")
		}
		return cmdSETINDEX(ctx, list)
	default:
		return fail(ctx, "Unrecognized command: SET %s", name)
	}
}

func parseOnOff(s string) (bool, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "ON":
		return true, nil
	case "OFF":
		return false, nil
	default:
		return false, errString("expected ON or OFF")
	}
}
