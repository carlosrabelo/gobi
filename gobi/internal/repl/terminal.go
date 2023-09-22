package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// terminalReader reads input from a terminal with raw mode support for
// arrow key navigation. Falls back to line reading for non-terminal inputs.
type terminalReader struct {
	in     *os.File
	out    io.Writer
	prompt string
}

func newTerminalReader(in *os.File, out io.Writer, prompt string) *terminalReader {
	return &terminalReader{
		in:     in,
		out:    out,
		prompt: prompt,
	}
}

// readLine reads a full line of input, supporting arrow key history navigation.
// The history parameter is used for up/down navigation callbacks.
func (r *terminalReader) readLine(history *History) (string, error) {
	if !term.IsTerminal(r.in) {
		return r.readLineFallback()
	}

	raw, err := term.EnterRawMode(r.in)
	if err != nil {
		return r.readLineFallback()
	}
	defer raw.Close()

	kbd := term.NewKeyboard(r.in)
	var buf []byte
	cur := 0
	histPos := len(history.items)
	fmt.Fprint(r.out, r.prompt)

	for {
		key, err := kbd.ReadKey()
		if err != nil {
			return "", err
		}

		switch key.Kind {
		case term.KeyEnter:
			fmt.Fprint(r.out, "\r\n")
			return string(buf), nil

		case term.KeyByte:
			if key.Byte == 4 {
				return "", io.EOF
			}
			if key.Byte >= 32 {
				buf = append(buf[:cur], append([]byte{key.Byte}, buf[cur:]...)...)
				cur++
				r.redrawLine(buf, cur)
			}

		case term.KeyBackspace:
			if cur > 0 {
				buf = append(buf[:cur-1], buf[cur:]...)
				cur--
				r.redrawLine(buf, cur)
			}

		case term.KeyLeft:
			if cur > 0 {
				cur--
				r.redrawLine(buf, cur)
			}

		case term.KeyRight:
			if cur < len(buf) {
				cur++
				r.redrawLine(buf, cur)
			}

		case term.KeyUp:
			if histPos > 0 {
				histPos--
				buf = []byte(history.items[histPos])
				cur = len(buf)
				r.redrawLine(buf, cur)
			}

		case term.KeyDown:
			if histPos < len(history.items) {
				histPos++
				if histPos < len(history.items) {
					buf = []byte(history.items[histPos])
				} else {
					buf = nil
				}
				cur = len(buf)
				r.redrawLine(buf, cur)
			}
		}
	}
}

// redrawLine repaints the prompt line with the given buffer, erasing any
// leftover characters from a previously displayed longer line, and places
// the terminal cursor at position cur within the buffer.
func (r *terminalReader) redrawLine(buf []byte, cur int) {
	fmt.Fprint(r.out, "\r", r.prompt, string(buf))
	term.EraseLine(r.out)
	if back := len(buf) - cur; back > 0 {
		fmt.Fprintf(r.out, "\033[%dD", back)
	}
}

func (r *terminalReader) readLineFallback() (string, error) {
	fmt.Fprint(r.out, r.prompt)
	reader := bufio.NewReader(r.in)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
