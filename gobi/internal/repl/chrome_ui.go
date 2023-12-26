package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func paintChrome(ctx *context.Context) {
	if ctx.AssistOpen {
		return
	}
	scr := ctx.Screen
	if scr == nil || scr.Rows < 2 || scr.Cols < 8 {
		return
	}
	a := ctx.Active()
	if ctx.Config.Status {
		row := scr.Rows - 1
		clearScreenRow(ctx, row)
		alias := "(none)"
		if a.HasTable() && a.Alias != "" {
			alias = a.Alias
		}
		rec := 0
		if a.Landed {
			rec = a.Cursor
		}
		line := fmt.Sprintf("%s Rec:%d", alias, rec)
		if ctx.InsertMode {
			line += " Ins"
		}
		if len(line) > scr.Cols {
			line = line[:scr.Cols]
		}
		scr.WriteString(row, 0, line, false)
		if ctx.Message != "" && row > 0 {
			msgRow := row - 1
			clearScreenRow(ctx, msgRow)
			msg := ctx.Message
			if len(msg) > scr.Cols {
				msg = msg[:scr.Cols]
			}
			scr.WriteString(msgRow, 0, msg, false)
		}
		return
	}
	if ctx.Config.Scoreboard {
		var parts []string
		if a.Landed && a.Record.Deleted {
			parts = append(parts, "Del")
		}
		if ctx.InsertMode {
			parts = append(parts, "Ins")
		}
		text := strings.Join(parts, " ")
		if text == "" {
			return
		}
		col := scr.Cols - len(text)
		if col < 0 {
			col = 0
		}
		scr.WriteString(0, col, text, false)
	}
}

func clearScreenRow(ctx *context.Context, row int) {
	scr := ctx.Screen
	if scr == nil {
		return
	}
	scr.WriteString(row, 0, strings.Repeat(" ", scr.Cols), false)
}

func flushScreen(ctx *context.Context) error {
	paintChrome(ctx)
	return ensureScreen(ctx).Present(ctx.Out())
}

func flushScreenClear(ctx *context.Context) error {
	paintChrome(ctx)
	return ensureScreen(ctx).PresentClear(ctx.Out())
}

func menuReminder(ctx *context.Context) string {
	if !ctx.Config.Menus {
		return ""
	}
	return "^W save  Esc abort  Tab next"
}
