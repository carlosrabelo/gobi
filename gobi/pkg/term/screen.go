package term

import (
	"io"
)

const (
	// DefaultCols is the default full-screen width in characters.
	DefaultCols = 80
	// DefaultRows is the default full-screen height in characters.
	DefaultRows = 24
)

// GetField describes a registered @ GET input location for a later READ.
type GetField struct {
	Row     int
	Col     int
	Name    string
	Picture string
}

// Screen is an in-memory character grid for full-screen TUI output.
type Screen struct {
	cols        int
	rows        int
	cells       []byte
	commandLine bool
	getFields   []GetField
}

// NewScreen returns a cleared screen with the given dimensions.
func NewScreen(cols, rows int) *Screen {
	if cols <= 0 {
		cols = DefaultCols
	}
	if rows <= 0 {
		rows = DefaultRows
	}
	s := &Screen{
		cols:  cols,
		rows:  rows,
		cells: make([]byte, cols*rows),
	}
	s.Clear()
	return s
}

// Resize changes the screen dimensions, clearing the buffer and any
// registered GET fields when the size actually changes.
func (s *Screen) Resize(cols, rows int) {
	if s == nil {
		return
	}
	if cols <= 0 {
		cols = DefaultCols
	}
	if rows <= 0 {
		rows = DefaultRows
	}
	if cols == s.cols && rows == s.rows {
		return
	}
	s.cols = cols
	s.rows = rows
	s.cells = make([]byte, cols*rows)
	s.Clear()
}

// Cols returns the screen width in characters.
func (s *Screen) Cols() int {
	if s == nil {
		return 0
	}
	return s.cols
}

// Rows returns the screen height in characters.
func (s *Screen) Rows() int {
	if s == nil {
		return 0
	}
	return s.rows
}

// Clear fills every cell with spaces and enables command-line prompt mode.
func (s *Screen) Clear() {
	if s == nil {
		return
	}
	for i := range s.cells {
		s.cells[i] = ' '
	}
	s.commandLine = true
	s.getFields = s.getFields[:0]
}

// ClearGets releases all registered @ GET fields without touching the
// screen contents (CLEAR GETS).
func (s *Screen) ClearGets() {
	if s == nil {
		return
	}
	s.getFields = s.getFields[:0]
}

// RegisterGet appends an interactive input field at the given coordinates.
func (s *Screen) RegisterGet(row, col int, name, picture string) {
	if s == nil {
		return
	}
	s.getFields = append(s.getFields, GetField{
		Row:     row,
		Col:     col,
		Name:    name,
		Picture: picture,
	})
}

// GetFields returns a copy of registered @ GET fields in registration order.
func (s *Screen) GetFields() []GetField {
	if s == nil || len(s.getFields) == 0 {
		return nil
	}
	out := make([]GetField, len(s.getFields))
	copy(out, s.getFields)
	return out
}

// UseCommandLine reports whether interactive input should use the last screen row.
func (s *Screen) UseCommandLine() bool {
	if s == nil {
		return false
	}
	return s.commandLine
}

// CommandLineRow returns the 0-based row index reserved for the REPL prompt.
func (s *Screen) CommandLineRow() int {
	if s == nil {
		return DefaultRows - 1
	}
	return s.rows - 1
}

// MoveToCommandLine positions the terminal cursor at the start of the command row.
func (s *Screen) MoveToCommandLine(w io.Writer) error {
	if s == nil {
		return nil
	}
	return MoveTo(w, s.CommandLineRow(), 0)
}

// At returns the character at the 0-based row and column.
func (s *Screen) At(row, col int) byte {
	if s == nil || !s.inBounds(row, col) {
		return ' '
	}
	return s.cells[s.index(row, col)]
}

// Set stores ch at the 0-based row and column.
func (s *Screen) Set(row, col int, ch byte) {
	if s == nil || !s.inBounds(row, col) {
		return
	}
	s.cells[s.index(row, col)] = ch
}

// WriteAt writes text starting at the 0-based row and column.
func (s *Screen) WriteAt(row, col int, text string) {
	if s == nil {
		return
	}
	for i := 0; i < len(text); i++ {
		if col+i >= s.cols {
			break
		}
		s.Set(row, col+i, text[i])
	}
}

// Highlight marks a screen region for reverse-video rendering when intensity is on.
type Highlight struct {
	Row    int
	Col    int
	Length int
}

// PresentOptions controls optional reverse-video highlighting during Present.
type PresentOptions struct {
	Intensity  bool
	Highlights []Highlight
}

// Present renders the screen through w using a frame writer.
func (s *Screen) Present(w io.Writer) error {
	return s.PresentWith(w, PresentOptions{})
}

// PresentWith renders the screen and applies reverse-video to highlighted regions
// when Intensity is true.
func (s *Screen) PresentWith(w io.Writer, opts PresentOptions) error {
	if s == nil {
		return nil
	}
	fw := NewFrameWriter(w)
	fw.Begin()
	s.writeTo(fw.Back(), opts)
	return fw.Present()
}

func (s *Screen) writeTo(w io.Writer, opts PresentOptions) {
	for row := 0; row < s.rows; row++ {
		s.writeRow(w, row, opts)
		_, _ = io.WriteString(w, "\r\n")
	}
}

func (s *Screen) writeRow(w io.Writer, row int, opts PresentOptions) {
	highlighted := false
	for col := 0; col < s.cols; col++ {
		shouldHighlight := opts.Intensity && s.isHighlighted(row, col, opts.Highlights)
		if shouldHighlight != highlighted {
			if shouldHighlight {
				_, _ = io.WriteString(w, Style{Reverse: true}.Sequence())
			} else {
				_, _ = io.WriteString(w, esc+"[0m")
			}
			highlighted = shouldHighlight
		}
		_, _ = w.Write([]byte{s.cells[s.index(row, col)]})
	}
	if highlighted {
		_, _ = io.WriteString(w, esc+"[0m")
	}
}

func (s *Screen) isHighlighted(row, col int, highlights []Highlight) bool {
	for _, h := range highlights {
		if row != h.Row || col < h.Col || col >= h.Col+h.Length {
			continue
		}
		return true
	}
	return false
}

func (s *Screen) index(row, col int) int {
	return row*s.cols + col
}

func (s *Screen) inBounds(row, col int) bool {
	return row >= 0 && row < s.rows && col >= 0 && col < s.cols
}
