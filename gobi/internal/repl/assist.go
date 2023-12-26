package repl

import (
	"fmt"
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// assistCategories is the top menu bar order (037).
var assistCategories = []string{
	"Set Up", "Create", "Update", "Position", "Retrieve",
	"Organize", "Modify", "Tools", "Exit",
}

func (m *Mux) cmdASSIST(ctx *context.Context, cmd command.Command) error {
	if strings.TrimSpace(cmd.Args) != "" {
		return fail(ctx, "ASSIST takes no arguments")
	}
	ctx.AssistOpen = true
	defer func() { ctx.AssistOpen = false }()

	scr := ensureScreen(ctx)
	keys := term.NewReader(ctx.Stdin)
	ctx.Keys = keys
	cat := 0
	barScroll := 0

	for {
		drawAssistShell(ctx, scr, cat, barScroll, "")
		if err := flushScreenClear(ctx); err != nil {
			return err
		}
		k, err := keys.ReadKey()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fail(ctx, "%s", err.Error())
		}
		switch k {
		case term.KeyEsc, term.KeyCtrlQ:
			return nil
		case term.KeyLeft:
			if cat > 0 {
				cat--
			}
			barScroll = assistBarScroll(scr.Cols, cat, barScroll)
		case term.KeyRight:
			if cat < len(assistCategories)-1 {
				cat++
			}
			barScroll = assistBarScroll(scr.Cols, cat, barScroll)
		default:
			// 037.1: shell only; other keys ignored until later tasks.
		}
	}
}

func assistBarScroll(cols, cat, scroll int) int {
	// Ensure category cat is visible: recompute scroll so labels fit.
	_ = cols
	if cat < scroll {
		return cat
	}
	return scroll
}

func drawAssistShell(ctx *context.Context, scr *term.Screen, cat, barScroll int, message string) {
	scr.Clear()
	rev := ctx.Config.Intensity
	drawAssistBar(scr, cat, barScroll, rev)
	msgRow := scr.Rows - 2
	statRow := scr.Rows - 1
	if message == "" {
		message = "Esc=leave  Left/Right=menu"
	}
	if len(message) > scr.Cols {
		message = message[:scr.Cols]
	}
	scr.WriteString(msgRow, 0, message, false)
	scr.WriteString(statRow, 0, assistStatusLine(ctx, scr.Cols), false)
}

func drawAssistBar(scr *term.Screen, cat, scroll int, reverseActive bool) {
	col := 0
	for i := scroll; i < len(assistCategories); i++ {
		label := assistCategories[i]
		need := len(label)
		if i > scroll {
			need += 2 // gap
		}
		if col+need > scr.Cols {
			break
		}
		if i > scroll {
			scr.WriteString(0, col, "  ", false)
			col += 2
		}
		scr.WriteString(0, col, label, reverseActive && i == cat)
		col += len(label)
	}
}

func assistStatusLine(ctx *context.Context, cols int) string {
	a := ctx.Active()
	alias := "(none)"
	if a.HasTable() && a.Alias != "" {
		alias = a.Alias
	}
	rec := 0
	if a.Landed {
		rec = a.Cursor
	}
	line := fmt.Sprintf("%s Rec:%d", alias, rec)
	if len(line) > cols {
		return line[:cols]
	}
	return line
}
