package repl

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
)

func init() {
	help.Register("SUSPEND", "SUSPEND — return to the prompt keeping the call stack.")
	help.Register("RESUME", "RESUME — continue after SUSPEND.")
	help.Register("TYPE", "TYPE name — print a text file.")
}

func setHistoryFlags(ctx *context.Context, m *Mux, name string, args []string, to string) error {
	switch name {
	case "HISTORY":
		if to != "" || (len(args) >= 2 && strings.EqualFold(args[1], "TO")) {
			n, err := strconv.Atoi(strings.TrimSpace(to))
			if to == "" && len(args) >= 3 {
				n, err = strconv.Atoi(args[2])
			}
			if err != nil || n < 0 || n > 500 {
				return fail(ctx, "SET HISTORY TO out of range")
			}
			ctx.Config.HistoryTo = n
			return nil
		}
		if len(args) < 2 {
			return fail(ctx, "SET HISTORY requires ON or OFF")
		}
		on, err := parseOnOff(args[1])
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		ctx.Config.History = on
		return nil
	case "DOHISTORY":
		if len(args) < 2 {
			return fail(ctx, "SET DOHISTORY requires ON or OFF")
		}
		on, err := parseOnOff(args[1])
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		ctx.Config.DoHistory = on
		return nil
	case "TYPEAHEAD":
		n := 0
		var err error
		if to != "" {
			n, err = strconv.Atoi(strings.TrimSpace(to))
		} else if len(args) >= 3 {
			n, err = strconv.Atoi(args[2])
		}
		if err != nil || n < 0 || n > 32000 {
			return fail(ctx, "SET TYPEAHEAD out of range")
		}
		ctx.Config.Typeahead = n
		return nil
	case "STATUS", "SCOREBOARD", "MENUS":
		if len(args) < 2 {
			return fail(ctx, "SET %s requires ON or OFF", name)
		}
		on, err := parseOnOff(args[1])
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		switch name {
		case "STATUS":
			ctx.Config.Status = on
		case "SCOREBOARD":
			ctx.Config.Scoreboard = on
		case "MENUS":
			ctx.Config.Menus = on
		}
		return nil
	case "MESSAGE":
		text := to
		if text == "" && len(args) >= 2 && strings.EqualFold(args[1], "TO") {
			text = strings.Join(args[2:], " ")
		}
		if len(text) > 79 {
			text = text[:79]
		}
		ctx.Message = text
		return nil
	case "MEMOWIDTH":
		n := 0
		var err error
		if to != "" {
			n, err = strconv.Atoi(strings.TrimSpace(to))
		} else if len(args) >= 3 {
			n, err = strconv.Atoi(args[2])
		}
		if err != nil || n < 8 || n > 254 {
			return fail(ctx, "SET MEMOWIDTH out of range")
		}
		ctx.Config.MemoWidth = n
		return nil
	}
	return fail(ctx, "internal chrome SET")
}

func cmdDisplayHistory(ctx *context.Context, hist *History, cmd command.Command) error {
	if hist == nil {
		return nil
	}
	lines := hist.Lines()
	args := strings.ToUpper(cmd.Args)
	if strings.Contains(args, "LAST") {
		fields := strings.Fields(cmd.Args)
		for i, f := range fields {
			if strings.EqualFold(f, "LAST") && i+1 < len(fields) {
				n, _ := strconv.Atoi(fields[i+1])
				if n > 0 && n < len(lines) {
					lines = lines[len(lines)-n:]
				}
			}
		}
	}
	w := ctx.Stdout
	if strings.Contains(args, "TO PRINT") || strings.EqualFold(strings.TrimSpace(cmd.Clauses["TO"]), "PRINT") {
		w = printSink(ctx)
	}
	for _, ln := range lines {
		fmt.Fprintln(w, ln)
	}
	return nil
}

func cmdSUSPEND(ctx *context.Context, m *Mux, cmd command.Command) error {
	if m.prog == nil || len(m.prog.Stack) == 0 {
		return fail(ctx, "SUSPEND outside a program")
	}
	ctx.Suspended = true
	return errSuspend
}

func cmdRESUME(ctx *context.Context, m *Mux, cmd command.Command) error {
	if !ctx.Suspended || m.prog == nil || len(m.prog.Stack) == 0 {
		return fail(ctx, "RESUME with no suspended file")
	}
	ctx.Suspended = false
	return runProgram(ctx, m)
}

func cmdTYPE(ctx *context.Context, cmd command.Command) error {
	args := strings.Fields(cmd.Args)
	if len(args) == 0 {
		return fail(ctx, "TYPE requires a name")
	}
	path, err := ctx.Resolve(args[0])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if strings.ContainsRune(string(data), 0) {
		fmt.Fprint(ctx.Stdout, string(data))
		return fail(ctx, "binary file")
	}
	fmt.Fprint(ctx.Stdout, string(data))
	to := strings.TrimSpace(cmd.Clauses["TO"])
	if strings.EqualFold(to, "PRINT") {
		fmt.Fprint(printSink(ctx), string(data))
	}
	return nil
}

// errSuspend stops runProgram without failing the DO.
var errSuspend = errString("suspend")
