package repl

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func init() {
	help.Register("MODIFY", "MODIFY STRUCTURE|COMMAND — change table structure or edit a .prg.")
}

func cmdMODIFY(ctx *context.Context, cmd command.Command) error {
	if err := refuseCreateModify(ctx, cmd, "MODIFY"); err != nil {
		return err
	}
	args := strings.Fields(cmd.Args)
	if len(args) == 0 {
		return fail(ctx, "MODIFY STRUCTURE or MODIFY COMMAND required")
	}
	if strings.EqualFold(args[0], "STRUCTURE") {
		return cmdModifyStructure(ctx, cmd)
	}
	if strings.EqualFold(args[0], "COMMAND") {
		name := ""
		if len(args) >= 2 {
			name = args[1]
		}
		if name == "" {
			return fail(ctx, "MODIFY COMMAND requires a name")
		}
		return modifyCommand(ctx, name)
	}
	return fail(ctx, "MODIFY STRUCTURE or MODIFY COMMAND required")
}

func modifyCommand(ctx *context.Context, name string) error {
	if filepath.Ext(name) == "" {
		name += ".prg"
	}
	path, err := ctx.Resolve(name)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
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
	scr := ensureScreen(ctx)
	keys := term.NewReader(ctx.Stdin)
	cur := 0
	if len(lines) == 0 {
		lines = []string{""}
	}
	col := len(lines[0])

	redraw := func() {
		scr.Clear()
		scr.WriteString(0, 0, "MODIFY COMMAND: "+filepath.Base(path), false)
		for i, ln := range lines {
			if i+1 >= scr.Rows {
				break
			}
			scr.WriteString(i+1, 0, ln, i == cur)
		}
		_ = scr.Present(ctx.Stdout)
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
			if k >= 32 && k < 127 {
				ch := byte(k)
				ln := lines[cur]
				lines[cur] = ln[:col] + string(ch) + ln[col:]
				col++
			}
		}
	}
}
