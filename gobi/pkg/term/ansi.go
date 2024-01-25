// Package term provides ANSI/VT100 escape sequences for terminal output.
package term

import (
	"fmt"
	"io"
)

const esc = "\033"

// Color is a standard ANSI color index (0-7).
type Color int

const (
	ColorBlack   Color = 0
	ColorRed     Color = 1
	ColorGreen   Color = 2
	ColorYellow  Color = 3
	ColorBlue    Color = 4
	ColorMagenta Color = 5
	ColorCyan    Color = 6
	ColorWhite   Color = 7
)

// Style describes ANSI SGR text attributes.
type Style struct {
	Foreground *Color
	Background *Color
	Bold       bool
	Reverse    bool
}

// Sequence returns the SGR escape sequence for s.
func (s Style) Sequence() string {
	parts := make([]int, 0, 4)
	if s.Bold {
		parts = append(parts, 1)
	}
	if s.Reverse {
		parts = append(parts, 7)
	}
	if s.Foreground != nil {
		parts = append(parts, 30+int(*s.Foreground))
	}
	if s.Background != nil {
		parts = append(parts, 40+int(*s.Background))
	}
	if len(parts) == 0 {
		return esc + "[0m"
	}
	return fmt.Sprintf("%s[%sm", esc, joinInts(parts))
}

// MoveTo writes a cursor-position sequence using dBase-style 0-based coordinates.
func MoveTo(w io.Writer, row, col int) error {
	return writef(w, "%s[%d;%dH", esc, row+1, col+1)
}

// Home writes a cursor-home sequence to the upper-left cell.
func Home(w io.Writer) error {
	_, err := io.WriteString(w, esc+"[H")
	return err
}

// SaveCursor stores the current cursor position (DECSC).
func SaveCursor(w io.Writer) error {
	_, err := io.WriteString(w, esc+"7")
	return err
}

// RestoreCursor returns the cursor to the position stored by SaveCursor (DECRC).
func RestoreCursor(w io.Writer) error {
	_, err := io.WriteString(w, esc+"8")
	return err
}

// ClearScreen erases the display and moves the cursor home.
func ClearScreen(w io.Writer) error {
	_, err := io.WriteString(w, esc+"[2J"+esc+"[H")
	return err
}

// EraseDisplay erases the entire display without moving the cursor.
func EraseDisplay(w io.Writer) error {
	_, err := io.WriteString(w, esc+"[2J")
	return err
}

// EraseLine erases from the cursor to the end of the current line.
func EraseLine(w io.Writer) error {
	_, err := io.WriteString(w, esc+"[K")
	return err
}

// HideCursor hides the terminal cursor.
func HideCursor(w io.Writer) error {
	_, err := io.WriteString(w, esc+"[?25l")
	return err
}

// ShowCursor shows the terminal cursor.
func ShowCursor(w io.Writer) error {
	_, err := io.WriteString(w, esc+"[?25h")
	return err
}

// SetStyle writes the SGR sequence for style.
func SetStyle(w io.Writer, style Style) error {
	_, err := io.WriteString(w, style.Sequence())
	return err
}

// Reset writes the default attribute reset sequence.
func Reset(w io.Writer) error {
	_, err := io.WriteString(w, esc+"[0m")
	return err
}

// SetForeground writes a foreground color SGR sequence.
func SetForeground(w io.Writer, color Color) error {
	return writef(w, "%s[%dm", esc, 30+int(color))
}

// SetBackground writes a background color SGR sequence.
func SetBackground(w io.Writer, color Color) error {
	return writef(w, "%s[%dm", esc, 40+int(color))
}

// SetBold enables bold/intensity highlighting.
func SetBold(w io.Writer) error {
	_, err := io.WriteString(w, esc+"[1m")
	return err
}

// SetReverse enables reverse-video (inverse intensity) mode.
func SetReverse(w io.Writer) error {
	_, err := io.WriteString(w, esc+"[7m")
	return err
}

// RingBell writes the terminal bell (BEL) character.
func RingBell(w io.Writer) error {
	_, err := io.WriteString(w, "\a")
	return err
}

func writef(w io.Writer, format string, args ...interface{}) error {
	_, err := fmt.Fprintf(w, format, args...)
	return err
}

func joinInts(values []int) string {
	if len(values) == 0 {
		return ""
	}
	out := fmt.Sprintf("%d", values[0])
	for _, v := range values[1:] {
		out += fmt.Sprintf(";%d", v)
	}
	return out
}
