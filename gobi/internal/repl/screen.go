package repl

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// Minimum usable screen geometry when adapting to very small terminals.
const (
	minScreenCols = 40
	minScreenRows = 10
)

// syncScreenSize adapts the logical screen to the detected terminal size
// when SET SCREEN AUTO is active. SET SCREEN DEFAULT pins the classic
// dBase II 80x24 geometry regardless of the real terminal.
func syncScreenSize(ctx *context.Context) {
	if ctx == nil || ctx.Screen == nil {
		return
	}
	if ctx.Config != nil && !ctx.Config.ScreenAuto {
		ctx.Screen.Resize(term.DefaultCols, term.DefaultRows)
		return
	}
	cols, rows, ok := detectTerminalSize(ctx)
	if !ok {
		return
	}
	ctx.Screen.Resize(cols, rows)
}

// detectTerminalSize queries the real terminal dimensions from the
// context's stdout or stdin, clamped to the minimum usable geometry.
func detectTerminalSize(ctx *context.Context) (cols, rows int, ok bool) {
	for _, stream := range []interface{}{ctx.Stdout, ctx.Stdin} {
		f, isFile := stream.(*os.File)
		if !isFile {
			continue
		}
		c, r, err := term.Size(f)
		if err != nil {
			continue
		}
		cols, rows = clampScreenSize(c, r)
		return cols, rows, true
	}
	return 0, 0, false
}

func clampScreenSize(cols, rows int) (int, int) {
	if cols < minScreenCols {
		cols = minScreenCols
	}
	if rows < minScreenRows {
		rows = minScreenRows
	}
	return cols, rows
}

// applySetScreen switches between adaptive (AUTO) and classic 80x24
// (DEFAULT) screen geometry. SET SCREEN is a Gobi extension, not a
// dBase II command.
func applySetScreen(ctx *context.Context, parts []string) error {
	if len(parts) != 2 {
		return fmt.Errorf("*** SET SCREEN requires AUTO or DEFAULT")
	}
	switch parts[1] {
	case "AUTO":
		ctx.Config.ScreenAuto = true
	case "DEFAULT":
		ctx.Config.ScreenAuto = false
	default:
		return fmt.Errorf("*** SET SCREEN requires AUTO or DEFAULT")
	}
	syncScreenSize(ctx)
	talkPrint(ctx, "SCREEN %s (%dx%d)\r\n", parts[1], ctx.Screen.Cols(), ctx.Screen.Rows())
	return nil
}

// interactiveTerminal reports whether the session is attached to a real
// terminal where cursor positioning sequences make sense.
func interactiveTerminal(ctx *context.Context) bool {
	f, ok := ctx.Stdin.(*os.File)
	return ok && term.IsTerminal(f)
}

// paintScreenText writes text at absolute screen coordinates without
// clearing the rest of the terminal, then restores the cursor so the dot
// prompt continues where it was. Text is clipped to the screen bounds.
func paintScreenText(ctx *context.Context, row, col int, text string) error {
	if ctx.Screen == nil || row >= ctx.Screen.Rows() || col >= ctx.Screen.Cols() {
		return nil
	}
	if max := ctx.Screen.Cols() - col; len(text) > max {
		text = text[:max]
	}

	w := ctx.Stdout
	if err := term.SaveCursor(w); err != nil {
		return err
	}
	if err := term.MoveTo(w, row, col); err != nil {
		return err
	}
	if _, err := io.WriteString(w, text); err != nil {
		return err
	}
	return term.RestoreCursor(w)
}

// returnToConsole places the cursor on the bottom screen row after a
// full-screen session, so the dot prompt resumes there and subsequent
// output scrolls the painted screen upward naturally.
func returnToConsole(ctx *context.Context) error {
	if ctx.Screen == nil || !interactiveTerminal(ctx) {
		return nil
	}
	if err := term.MoveTo(ctx.Stdout, ctx.Screen.CommandLineRow(), 0); err != nil {
		return err
	}
	return term.EraseLine(ctx.Stdout)
}

func presentClearScreen(ctx *context.Context) error {
	if ctx.Screen == nil {
		return nil
	}
	ctx.Screen.Clear()
	return term.ClearScreen(ctx.Stdout)
}

// handleClear implements the dBase II CLEAR command: it closes every open
// database (with indexes) and releases all memory variables. CLEAR GETS
// lands in a later commit. Screen clearing belongs to ERASE in dBase II.
func handleClear(ctx *context.Context, cmd Command) error {
	arg := strings.ToUpper(strings.TrimSpace(cmd.Args))
	switch arg {
	case "":
		for name, area := range ctx.WorkAreas {
			if err := closeWorkAreaDatabase(area, string(name)); err != nil {
				return err
			}
			closeOpenIndexes(area)
		}
		ctx.Variables.Clear()
		ctx.Screen.ClearGets()
		return nil
	case "GETS":
		ctx.Screen.ClearGets()
		return nil
	default:
		return fmt.Errorf("*** Unrecognized CLEAR option: %s", arg)
	}
}

func handleSet(ctx *context.Context, cmd Command) error {
	parts := strings.Fields(strings.ToUpper(strings.TrimSpace(cmd.Args)))
	if len(parts) == 0 {
		return fmt.Errorf("*** SET requires an option")
	}
	switch parts[0] {
	case "TALK":
		applySetTalk(ctx, parts)
		return nil
	case "INTENSITY":
		applySetIntensity(ctx, parts)
		return nil
	case "BELL":
		applySetBell(ctx, parts)
		return nil
	case "DEFAULT":
		args := cmd.Args
		if cmd.ToClause != "" {
			args += " TO " + cmd.ToClause
		}
		return applySetDefault(ctx, args)
	case "INDEX":
		return applySetIndex(ctx, cmd)
	case "SCREEN":
		return applySetScreen(ctx, parts)
	default:
		return fmt.Errorf("*** Unrecognized SET option: %s", parts[0])
	}
}

func parseOnOff(parts []string) bool {
	if len(parts) < 2 {
		return true
	}
	return parts[1] != "OFF"
}

func onOffStr(v bool) string {
	if v {
		return "ON"
	}
	return "OFF"
}
