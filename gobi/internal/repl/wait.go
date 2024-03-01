package repl

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/internal/symbols"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// handleWait implements WAIT [TO <memvar>], suspending execution until a
// single key is pressed. With TO, the pressed printable character is stored
// in a memory variable as a one-character string.
func handleWait(ctx *context.Context, cmd Command) error {
	target := strings.ToUpper(strings.TrimSpace(cmd.ToClause))
	if target != "" {
		if err := symbols.ValidateName(target); err != nil {
			return fmt.Errorf("*** Invalid WAIT target %s: %v", target, err)
		}
	}
	if strings.TrimSpace(cmd.Args) != "" {
		return fmt.Errorf("*** WAIT accepts only an optional TO <variable>")
	}

	fmt.Fprint(ctx.Stdout, "WAITING ")

	key, err := readWaitKey(ctx)
	if err != nil {
		if err == io.EOF {
			fmt.Fprint(ctx.Stdout, "\r\n")
			return nil
		}
		return fmt.Errorf("*** Error reading WAIT input: %w", err)
	}
	fmt.Fprint(ctx.Stdout, "\r\n")

	if target == "" {
		return nil
	}

	value := ""
	if key >= 32 && key <= 126 {
		value = string(key)
	}
	if err := ctx.Variables.Set(target, value); err != nil {
		return fmt.Errorf("*** Error storing %s: %v", target, err)
	}
	return nil
}

// readWaitKey reads a single keypress, using raw terminal mode when
// available so no Enter is required, or one byte from the shared stdin
// reader otherwise.
func readWaitKey(ctx *context.Context) (byte, error) {
	if interactiveTerminal(ctx) {
		tty := ctx.Stdin.(*os.File)
		raw, err := term.EnterRawMode(tty)
		if err == nil {
			defer raw.Close()
			return readReplKey(term.NewKeyboard(tty), true, tty)
		}
	}
	return readReplKey(nil, false, ctx.StdinReader())
}
