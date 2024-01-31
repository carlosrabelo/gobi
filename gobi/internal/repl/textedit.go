package repl

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// editTextFile is the shared line editor for MODIFY COMMAND and designers (040).
// Ctrl-W saves; Esc / Ctrl-Q discards. create==true starts with initial if the file is missing.
func editTextFile(ctx *context.Context, path, title string, initial []string) error {
	var lines []string
	if fileExists(path) {
		data, err := os.ReadFile(path)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		text := string(data)
		text = strings.ReplaceAll(text, "\r\n", "\n")
		text = strings.TrimSuffix(text, "\n")
		if text != "" {
			lines = strings.Split(text, "\n")
		}
	}
	if len(lines) == 0 {
		if len(initial) > 0 {
			lines = append([]string{}, initial...)
		} else {
			lines = []string{""}
		}
	}

	scr := ensureScreen(ctx)
	ctx.StatusMode = StatusModeModify
	defer func() { _ = leaveFullScreen(ctx) }()
	keys := term.NewReader(ctx.Stdin)
	ctx.Keys = keys
	cur := 0
	col := len(lines[0])
	if col > 0 {
		col = 0
	}

	first := true
	redraw := func() {
		for r := 0; r < scr.Rows; r++ {
			scr.WriteString(r, 0, strings.Repeat(" ", scr.Cols), false)
		}
		head := title + ": " + filepath.Base(path)
		if len(head) > scr.Cols {
			head = head[:scr.Cols]
		}
		scr.WriteString(0, 0, head, false)
		work := statusWorkRows(scr.Rows)
		maxBody := work - 1
		if maxBody < 1 {
			maxBody = 1
		}
		for i, ln := range lines {
			if i >= maxBody {
				break
			}
			show := ln
			if len(show) > scr.Cols {
				show = show[:scr.Cols]
			}
			scr.WriteString(i+1, 0, show, i == cur && ctx.Config.Intensity)
		}
		prev := ctx.Message
		ctx.Message = "Ctrl-W save  Esc abort"
		paintStatusBand(ctx, StatusModeModify, true)
		ctx.Message = prev
		if first {
			_ = flushScreenClear(ctx)
			first = false
		} else {
			_ = flushScreen(ctx)
		}
	}

	for {
		redraw()
		k, err := keys.ReadKey()
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		switch k {
		case term.KeyCtrlW:
			body := strings.Join(lines, "\n")
			if !strings.HasSuffix(body, "\n") {
				body += "\n"
			}
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				return fail(ctx, "%s", err.Error())
			}
			return nil
		case term.KeyCtrlQ, term.KeyEsc:
			return nil
		case term.KeyUp:
			if cur > 0 {
				cur--
				if col > len(lines[cur]) {
					col = len(lines[cur])
				}
			}
		case term.KeyDown:
			if cur < len(lines)-1 {
				cur++
				if col > len(lines[cur]) {
					col = len(lines[cur])
				}
			}
		case term.KeyEnter:
			left := lines[cur][:col]
			right := lines[cur][col:]
			lines[cur] = left
			lines = append(lines[:cur+1], append([]string{right}, lines[cur+1:]...)...)
			cur++
			col = 0
		case term.KeyBackspace:
			if col > 0 {
				ln := lines[cur]
				lines[cur] = ln[:col-1] + ln[col:]
				col--
			} else if cur > 0 {
				prev := lines[cur-1]
				col = len(prev)
				lines[cur-1] = prev + lines[cur]
				lines = append(lines[:cur], lines[cur+1:]...)
				cur--
			}
		default:
			if term.IsInputByte(k) {
				ch := byte(k)
				ln := lines[cur]
				lines[cur] = ln[:col] + string(ch) + ln[col:]
				col++
			}
		}
	}
}
