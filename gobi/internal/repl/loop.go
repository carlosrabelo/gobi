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

	mux.hist = hist
	if hist != nil {
		hist.SetCap(ctx.Config.HistoryTo)
	}
	for {
		fmt.Fprint(ctx.Out(), ". ")
		line, err := readDotLine(ctx)
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
func readDotLine(ctx *context.Context) (string, error) {
	keys := term.NewReader(ctx.Stdin)
	ctx.Keys = keys
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
			// KeyEnter is CR; LF is also accepted if a reader yields it raw.
			return b.String(), nil
		case k == term.KeyCtrlC:
			// Cancel the current prompt line; do not exit (SIGINT is ignored above).
			b.Reset()
			fmt.Fprintln(ctx.Out())
			return "", nil
		case k >= term.KeyF1 && k <= term.KeyF12:
			n := k - term.KeyF1 + 1
			b.WriteString(ctx.FuncKeys[n])
		case k == term.KeyBackspace:
			s := b.String()
			if len(s) > 0 {
				b.Reset()
				b.WriteString(s[:len(s)-1])
			}
		case k >= 32 && k < 127:
			b.WriteByte(byte(k))
		}
	}
}
