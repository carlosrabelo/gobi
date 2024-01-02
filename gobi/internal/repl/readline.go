package repl

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// readInteractiveLine reads one line via the session key reader.
// Enter (CR or LF) ends the line. When stdin is a terminal, bytes are echoed
// (required after EnterRaw; cooked echo is off).
func readInteractiveLine(ctx *context.Context) (string, error) {
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
		case k == term.KeyEnter:
			if echo {
				fmt.Fprintln(ctx.Out())
			}
			return b.String(), nil
		case k == term.KeyBackspace:
			s := b.String()
			if len(s) > 0 {
				b.Reset()
				b.WriteString(s[:len(s)-1])
				if echo {
					fmt.Fprint(ctx.Out(), "\b \b")
				}
			}
		case k == term.KeyCtrlC:
			b.Reset()
			if echo {
				fmt.Fprintln(ctx.Out())
			}
			return "", nil
		case k >= 32 && k < 127:
			b.WriteByte(byte(k))
			if echo {
				fmt.Fprint(ctx.Out(), string(byte(k)))
			}
		}
	}
}

// readInteractiveKey returns one logical key (for WAIT).
func readInteractiveKey(ctx *context.Context) (int, error) {
	if ctx.Keys == nil {
		ctx.Keys = term.NewReader(ctx.Stdin)
	}
	return ctx.Keys.ReadKey()
}
