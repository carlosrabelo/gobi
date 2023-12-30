package repl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

const maxLine = 254

// Run writes the dot prompt, reads lines, and dispatches verbs until QUIT or EOF.
func Run(ctx *context.Context, mux *Mux) error {
	return RunWithHistory(ctx, mux, nil)
}

// RunWithHistory is Run with optional command history.
func RunWithHistory(ctx *context.Context, mux *Mux, hist *History) error {
	// Cooked terminals turn Ctrl-C into SIGINT; dBase stays in the session.
	// Ignore for the interactive loop only (QUIT / EOF still leave).
	signal.Ignore(os.Interrupt)
	defer signal.Reset(os.Interrupt)

	// Spec 018: raw keyboard when stdin is a terminal (prompt + full-screen).
	restore, err := term.EnterRaw(ctx.Stdin)
	if err != nil {
		return err
	}
	defer restore()

	mux.hist = hist
	if hist != nil {
		hist.SetCap(ctx.Config.HistoryTo)
	}
	for {
		fmt.Fprint(ctx.Out(), ". ")
		line, err := readDotLine(ctx, hist)
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				if hist != nil {
					_ = hist.Save()
				}
				return nil
			}
		} else if err != nil {
			return err
		}
		if line == "" {
			continue
		}
		if len(line) > maxLine {
			fmt.Fprintln(ctx.Stderr, "*** Line too long")
			continue
		}
		if hist != nil && ctx.Config.History {
			hist.Add(line)
		}
		line, err = expandLineMacros(ctx, line)
		if err != nil {
			fmt.Fprintln(ctx.Stderr, "*** "+err.Error())
			continue
		}
		cmd := command.Split(line)
		err = mux.Dispatch(ctx, cmd)
		if errors.Is(err, ErrQuit) {
			if hist != nil {
				_ = hist.Save()
			}
			return nil
		}
		if err != nil {
			continue
		}
	}
}

// readDotLine builds one dot-prompt line, expanding SET FUNCTION keys (032.6).
// When stdin is a terminal, it echoes and walks history with up/down (018.2).
func readDotLine(ctx *context.Context, hist *History) (string, error) {
	// Reuse the session reader so unread pushback (e.g. after CR) is not lost.
	if ctx.Keys == nil {
		ctx.Keys = term.NewReader(ctx.Stdin)
	}
	keys := ctx.Keys
	echo := term.IsTerminal(ctx.Stdin)
	var b strings.Builder
	for {
		k, err := keys.ReadKey()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return b.String(), io.EOF
			}
			return "", err
		}
		switch {
		case k == term.KeyEnter || k == '\n':
			if echo {
				fmt.Fprintln(ctx.Out())
			}
			return b.String(), nil
		case k == term.KeyCtrlC:
			// Cancel the current prompt line; do not exit (SIGINT is ignored above).
			b.Reset()
			fmt.Fprintln(ctx.Out())
			return "", nil
		case k == term.KeyUp:
			if hist == nil {
				continue
			}
			if s, ok := hist.Prev(); ok {
				b.Reset()
				b.WriteString(s)
				if echo {
					redrawDotLine(ctx.Out(), s)
				}
			}
		case k == term.KeyDown:
			if hist == nil {
				continue
			}
			if s, ok := hist.Next(); ok {
				b.Reset()
				b.WriteString(s)
				if echo {
					redrawDotLine(ctx.Out(), s)
				}
			} else {
				b.Reset()
				if echo {
					redrawDotLine(ctx.Out(), "")
				}
			}
		case k >= term.KeyF1 && k <= term.KeyF12:
			n := k - term.KeyF1 + 1
			frag := ctx.FuncKeys[n]
			b.WriteString(frag)
			if echo && frag != "" {
				fmt.Fprint(ctx.Out(), frag)
			}
		case k == term.KeyBackspace:
			s := b.String()
			if len(s) > 0 {
				b.Reset()
				b.WriteString(s[:len(s)-1])
				if echo {
					fmt.Fprint(ctx.Out(), "\b \b")
				}
			}
		case k >= 32 && k < 127:
			b.WriteByte(byte(k))
			if echo {
				fmt.Fprint(ctx.Out(), string(byte(k)))
			}
		}
	}
}

// redrawDotLine rewrites the prompt line after a history walk.
func redrawDotLine(out io.Writer, content string) {
	fmt.Fprintf(out, "\r. %s\x1b[K", content)
}
