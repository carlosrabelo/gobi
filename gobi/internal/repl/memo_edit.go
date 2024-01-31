package repl

import (
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// editMemoBuffer is the III Plus–style memo editor (Ctrl-W save, Esc abort).
func editMemoBuffer(ctx *context.Context, title, text string) (string, bool, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	var lines []string
	if text == "" {
		lines = []string{""}
	} else {
		lines = strings.Split(text, "\n")
	}
	scr := ensureScreen(ctx)
	prevMode := ctx.StatusMode
	ctx.StatusMode = StatusModeModify
	defer func() { ctx.StatusMode = prevMode }()

	if ctx.Keys == nil {
		ctx.Keys = term.NewReader(ctx.Stdin)
	}
	keys := ctx.Keys
	cur := 0
	col := 0
	first := true

	redraw := func() {
		blank := strings.Repeat(" ", scr.Cols)
		for r := 0; r < scr.Rows; r++ {
			scr.WriteString(r, 0, blank, false)
		}
		head := "Memo: " + title
		if len(head) > scr.Cols {
			head = head[:scr.Cols]
		}
		scr.WriteString(0, 0, head, false)
		work := statusWorkRows(scr.Rows)
		maxBody := work - 1
		if maxBody < 1 {
			maxBody = 1
		}
		start := 0
		if cur >= maxBody {
			start = cur - maxBody + 1
		}
		for i := start; i < len(lines) && i-start < maxBody; i++ {
			show := lines[i]
			if len(show) > scr.Cols {
				show = show[:scr.Cols]
			}
			scr.WriteString(1+(i-start), 0, padRight(show, scr.Cols), i == cur && ctx.Config.Intensity)
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
			if err == io.EOF {
				return strings.Join(lines, "\n"), false, nil
			}
			return "", false, err
		}
		switch k {
		case term.KeyCtrlW:
			return strings.Join(lines, "\n"), true, nil
		case term.KeyCtrlQ, term.KeyEsc:
			return text, false, nil
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
				lines[cur] = lines[cur][:col-1] + lines[cur][col:]
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
				lines[cur] = lines[cur][:col] + string(byte(k)) + lines[cur][col:]
				col++
			}
		}
	}
}

func memoFieldText(a *context.Area, v dbf.Value) string {
	if v.Empty || strings.TrimSpace(v.Str) == "" {
		return ""
	}
	if a.Table == nil || a.Table.Memo == nil {
		return ""
	}
	block, err := dbf.ParseBlockDigits(v.Str)
	if err != nil || block == 0 {
		// Might already be plain text from an in-progress form.
		return v.Str
	}
	text, err := a.Table.Memo.ReadText(block)
	if err != nil {
		return ""
	}
	return text
}

func memoBrowseLabel(a *context.Area, v dbf.Value) string {
	if v.Empty || strings.TrimSpace(v.Str) == "" {
		return ""
	}
	if memoFieldText(a, v) == "" {
		return ""
	}
	return "memo"
}

func writeMemoField(a *context.Area, rec *dbf.Record, fi int, text string) error {
	if a.Table == nil || a.Table.Memo == nil {
		return errString("no memo file")
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if strings.TrimSpace(text) == "" {
		rec.Values[fi] = dbf.Value{Empty: true}
		return nil
	}
	block, err := a.Table.Memo.AllocWrite(text)
	if err != nil {
		return err
	}
	rec.Values[fi] = dbf.Value{Str: dbf.FormatBlockDigits(block)}
	return nil
}
