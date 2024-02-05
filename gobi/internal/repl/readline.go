package repl

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// lastUTF8Rune returns the last rune and its byte size in s (1 if invalid).
func lastUTF8Rune(s string) (r rune, size int) {
	r, size = utf8.DecodeLastRuneInString(s)
	if r == utf8.RuneError && size == 1 {
		return r, 1
	}
	if size <= 0 {
		return 0, 1
	}
	return r, size
}

// readInteractiveLine reads one line via the session key reader.
// Enter (CR or LF) ends the line. When stdin is a terminal, bytes are echoed
// (required after EnterRaw; cooked echo is off).
func readInteractiveLine(ctx *context.Context) (string, error) {
	if ctx.Keys == nil {
		ctx.Keys = term.NewReader(ctx.Stdin)
	}
	drainTypeaheadIntoKeys(ctx)
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
				_, size := lastUTF8Rune(s)
				b.Reset()
				b.WriteString(s[:len(s)-size])
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
		case term.IsInputByte(k):
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
	drainTypeaheadIntoKeys(ctx)
	return ctx.Keys.ReadKey()
}
