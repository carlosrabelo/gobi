package repl

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/config"
	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func cmdSET(ctx *context.Context, cmd command.Command) error {
	args := strings.Fields(cmd.Args)
	if len(args) == 0 {
		return fail(ctx, "SET requires an option")
	}
	name := strings.ToUpper(args[0])
	to := strings.TrimSpace(cmd.Clauses["TO"])

	switch name {
	case "TALK", "INTENSITY", "BELL", "EXACT", "DELETED", "ECHO", "STEP", "CARRY",
		"CONFIRM", "ESCAPE", "CONSOLE", "DEBUG", "SAFETY", "PRINT", "FIXED", "HEADING",
		"ALTERNATE", "DELIMITERS":
		if name == "ALTERNATE" && (to != "" || (len(args) >= 2 && strings.EqualFold(args[1], "TO"))) {
			return setAlternateTO(ctx, to, args)
		}
		if name == "DELIMITERS" && (to != "" || (len(args) >= 2 && strings.EqualFold(args[1], "TO"))) {
			return setDelimitersTO(ctx, to, args)
		}
		if len(args) < 2 {
			return fail(ctx, "SET %s requires ON or OFF", name)
		}
		on, err := parseOnOff(args[1])
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		return setFlag(ctx, name, on)

	case "DEFAULT":
		path := to
		if path == "" && len(args) >= 3 && strings.EqualFold(args[1], "TO") {
			path = strings.Join(args[2:], " ")
		}
		if path == "" && to == "" && !(len(args) >= 2 && strings.EqualFold(args[1], "TO")) {
			return fail(ctx, "SET DEFAULT requires TO")
		}
		ctx.Config.Default = path
		return nil

	case "INDEX":
		list := to
		if list == "" && len(args) >= 2 && strings.EqualFold(args[1], "TO") {
			list = strings.Join(args[2:], " ")
		}
		return cmdSETINDEX(ctx, list)

	case "PATH":
		path := to
		if path == "" && len(args) >= 2 && strings.EqualFold(args[1], "TO") {
			path = strings.Join(args[2:], " ")
		}
		ctx.Config.Path = path
		return nil

	case "MARGIN":
		return setIntTO(ctx, &ctx.Config.Margin, to, args, 0, 1000, false)

	case "DECIMALS":
		return setIntTO(ctx, &ctx.Config.Decimals, to, args, 0, 15, true)

	case "DEVICE":
		val := strings.ToUpper(to)
		if val == "" && len(args) >= 2 {
			val = strings.ToUpper(args[1])
		}
		if val != "SCREEN" && val != "PRINT" {
			return fail(ctx, "SET DEVICE requires SCREEN or PRINT")
		}
		ctx.Config.Device = val
		return nil

	case "SCREEN":
		val := strings.ToUpper(to)
		if val == "" && len(args) >= 2 {
			val = strings.ToUpper(args[1])
		}
		switch val {
		case "AUTO":
			ctx.Config.Screen = config.ScreenAuto
		case "DEFAULT":
			ctx.Config.Screen = config.ScreenDefault
			ensureScreen(ctx).Resize(80, 24)
		default:
			return fail(ctx, "SET SCREEN requires AUTO or DEFAULT")
		}
		return nil

	case "FORMAT":
		name := to
		if name == "" && len(args) >= 2 && strings.EqualFold(args[1], "TO") {
			name = strings.Join(args[2:], " ")
		}
		ctx.Config.Format = name
		ctx.Active().Format = name != ""
		return nil

	case "COLOR":
		if len(args) >= 2 && (strings.EqualFold(args[1], "ON") || strings.EqualFold(args[1], "OFF")) {
			return fail(ctx, "Unrecognized command: SET COLOR %s", strings.ToUpper(args[1]))
		}
		text := to
		if text == "" && len(args) >= 2 && strings.EqualFold(args[1], "TO") {
			text = strings.Join(args[2:], " ")
		}
		ctx.Config.Color = text
		return nil

	case "DATE":
		val := strings.ToUpper(to)
		if val == "" && len(args) >= 2 {
			val = strings.ToUpper(args[1])
		}
		switch val {
		case "AMERICAN", "ANSI", "BRITISH", "ITALIAN", "FRENCH", "GERMAN":
			ctx.Config.DateOrder = val
			return nil
		default:
			return fail(ctx, "SET DATE requires AMERICAN, ANSI, BRITISH, ITALIAN, FRENCH, or GERMAN")
		}

	case "CENTURY":
		if len(args) < 2 {
			return fail(ctx, "SET CENTURY requires ON or OFF")
		}
		on, err := parseOnOff(args[1])
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		ctx.Config.Century = on
		return nil

	default:
		return fail(ctx, "Unrecognized command: SET %s", name)
	}
}

func (m *Mux) cmdSET(ctx *context.Context, cmd command.Command) error {
	args := strings.Fields(cmd.Args)
	if len(args) >= 1 && strings.EqualFold(args[0], "PROCEDURE") {
		to := strings.TrimSpace(cmd.Clauses["TO"])
		if to == "" && len(args) >= 2 && strings.EqualFold(args[1], "TO") {
			to = strings.Join(args[2:], " ")
		}
		return cmdSETPROCEDURE(ctx, m, to)
	}
	return cmdSET(ctx, cmd)
}

func setFlag(ctx *context.Context, name string, on bool) error {
	c := ctx.Config
	switch name {
	case "TALK":
		c.Talk = on
	case "INTENSITY":
		c.Intensity = on
	case "BELL":
		c.Bell = on
	case "EXACT":
		c.Exact = on
	case "DELETED":
		c.Deleted = on
	case "ECHO":
		c.Echo = on
	case "STEP":
		c.Step = on
	case "CARRY":
		c.Carry = on
	case "CONFIRM":
		c.Confirm = on
	case "DELIMITERS":
		c.Delimiters = on
	case "ESCAPE":
		c.Escape = on
	case "CONSOLE":
		c.Console = on
	case "ALTERNATE":
		c.Alternate = on
	case "DEBUG":
		c.Debug = on
	case "SAFETY":
		c.Safety = on
	case "PRINT":
		c.Print = on
	case "FIXED":
		c.Fixed = on
	case "HEADING":
		c.Heading = on
	}
	return nil
}

func setAlternateTO(ctx *context.Context, to string, args []string) error {
	name := to
	if name == "" && len(args) >= 3 {
		name = strings.Join(args[2:], " ")
	}
	if name == "" {
		_ = ctx.CloseAlternate()
		ctx.Config.AlternateTo = ""
		return nil
	}
	if filepath.Ext(name) == "" {
		name += ".txt"
	}
	path, err := resolveAny(ctx, name)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	f, err := os.Create(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	_ = ctx.CloseAlternate()
	ctx.Alternate = f
	ctx.Config.AlternateTo = path
	return nil
}

func setDelimitersTO(ctx *context.Context, to string, args []string) error {
	text := to
	if text == "" && len(args) >= 3 {
		text = strings.Join(args[2:], " ")
	}
	text = strings.Trim(text, `"'`)
	if len(text) > 2 {
		return fail(ctx, "SET DELIMITERS TO accepts at most two characters")
	}
	if text == "" {
		text = ":"
	}
	ctx.Config.DelimChars = text
	return nil
}

func setIntTO(ctx *context.Context, dest *int, to string, args []string, min, max int, inclusiveMax bool) error {
	s := to
	if s == "" && len(args) >= 3 && strings.EqualFold(args[1], "TO") {
		s = args[2]
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fail(ctx, "invalid number")
	}
	if n < min || (!inclusiveMax && n > max) || (inclusiveMax && n > max) {
		return fail(ctx, "value out of range")
	}
	*dest = n
	return nil
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
