package hostterm

import (
	"fmt"
	"io"
)

// Color indexes for SetColor. -1 means default/reset that channel.
const (
	ColorDefault = -1
)

// MoveCursor writes a cursor-address sequence when Caps.CursorAddress is true.
// Row and col are 0-based.
func MoveCursor(out io.Writer, caps Caps, row, col int) error {
	if !caps.CursorAddress {
		return nil
	}
	_, err := fmt.Fprintf(out, "\x1b[%d;%dH", row+1, col+1)
	return err
}

// Clear clears the display when Caps.Clear is true.
func Clear(out io.Writer, caps Caps) error {
	if !caps.Clear {
		return nil
	}
	_, err := io.WriteString(out, "\x1b[2J\x1b[H")
	return err
}

// SetReverse enables or disables reverse video when Caps.Reverse is true.
func SetReverse(out io.Writer, caps Caps, on bool) error {
	if !caps.Reverse {
		return nil
	}
	seq := "\x1b[0m"
	if on {
		seq = "\x1b[7m"
	}
	_, err := io.WriteString(out, seq)
	return err
}

// SetColor sets ANSI 16-color foreground/background when Caps.Color is true.
// Values 0–15 are standard; ColorDefault resets that channel.
func SetColor(out io.Writer, caps Caps, fg, bg int) error {
	if !caps.Color {
		return nil
	}
	if fg == ColorDefault && bg == ColorDefault {
		_, err := io.WriteString(out, "\x1b[39;49m")
		return err
	}
	if fg >= 0 && fg <= 15 {
		code := 30 + fg
		if fg >= 8 {
			code = 90 + (fg - 8)
		}
		if _, err := fmt.Fprintf(out, "\x1b[%dm", code); err != nil {
			return err
		}
	}
	if bg >= 0 && bg <= 15 {
		code := 40 + bg
		if bg >= 8 {
			code = 100 + (bg - 8)
		}
		if _, err := fmt.Fprintf(out, "\x1b[%dm", code); err != nil {
			return err
		}
	}
	return nil
}
