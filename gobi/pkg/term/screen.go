package term

import (
	"fmt"
	"io"
	"strings"
)

const (
	DefaultCols = 80
	DefaultRows = 24
	MinCols     = 40
	MinRows     = 10

	// BoxVDouble is CP437/Unicode double vertical (║) used on the III Plus status bar.
	BoxVDouble = '\u2551'
	// BoxVSingle is the single vertical (│) fallback when a host cannot show BoxVDouble.
	BoxVSingle = '\u2502'
)

// StatusSep is the segment separator painted on the status bar.
// Prefer the double-line glyph; hosts that only render ASCII still show a vertical bar cell.
var StatusSep = BoxVDouble

// Cell is one screen cell. Ch is a Unicode code point (one column).
type Cell struct {
	Ch      rune
	Reverse bool
}

// Screen is a double-buffered character grid.
type Screen struct {
	Cols, Rows int
	pending    []Cell
	presented  []Cell
	// NextRow/NextCol are 0-based cursor for ROW()/COL() after writes.
	NextRow, NextCol int
}

// New returns an 80x24 screen filled with spaces.
func New() *Screen {
	return NewSize(DefaultCols, DefaultRows)
}

// NewSize returns a screen of the given size, clamped to floors.
func NewSize(cols, rows int) *Screen {
	s := &Screen{}
	s.Resize(cols, rows)
	return s
}

// Resize changes the grid size. Values below the floors are clamped.
// The overlapping top-left region of pending/presented is preserved; new
// cells are spaces (spec 021).
func (s *Screen) Resize(cols, rows int) {
	if cols < MinCols {
		cols = MinCols
	}
	if rows < MinRows {
		rows = MinRows
	}
	if cols == s.Cols && rows == s.Rows && s.pending != nil {
		return
	}
	oldCols, oldRows := s.Cols, s.Rows
	oldPending, oldPresented := s.pending, s.presented
	n := cols * rows
	s.Cols = cols
	s.Rows = rows
	s.pending = make([]Cell, n)
	s.presented = make([]Cell, n)
	for i := range s.pending {
		s.pending[i] = Cell{Ch: ' '}
		s.presented[i] = Cell{Ch: ' '}
	}
	copyRows := oldRows
	if rows < copyRows {
		copyRows = rows
	}
	copyCols := oldCols
	if cols < copyCols {
		copyCols = cols
	}
	for r := 0; r < copyRows; r++ {
		for c := 0; c < copyCols; c++ {
			oi := r*oldCols + c
			ni := r*cols + c
			if oi < len(oldPending) {
				s.pending[ni] = oldPending[oi]
			}
			if oi < len(oldPresented) {
				s.presented[ni] = oldPresented[oi]
			}
		}
	}
	if s.NextRow >= s.Rows {
		s.NextRow = s.Rows - 1
	}
	if s.NextCol >= s.Cols {
		s.NextCol = s.Cols - 1
	}
}

func (s *Screen) at(r, c int) int {
	return r*s.Cols + c
}

// Put writes ch at (row, col). Outside the grid is a no-op.
func (s *Screen) Put(row, col int, ch rune, reverse bool) {
	if row < 0 || col < 0 || row >= s.Rows || col >= s.Cols {
		return
	}
	if ch == 0 {
		ch = ' '
	}
	s.pending[s.at(row, col)] = Cell{Ch: ch, Reverse: reverse}
}

// WriteString writes text starting at (row, col), one rune per cell.
// Advances NextRow/NextCol.
func (s *Screen) WriteString(row, col int, text string, reverse bool) {
	r, c := row, col
	for _, ch := range text {
		s.Put(r, c, ch, reverse)
		c++
		if c >= s.Cols {
			c = s.Cols - 1
		}
	}
	s.NextRow = r
	s.NextCol = c
	if s.NextCol >= s.Cols {
		s.NextCol = s.Cols - 1
	}
}

// Clear fills pending with spaces and non-reverse.
func (s *Screen) Clear() {
	for i := range s.pending {
		s.pending[i] = Cell{Ch: ' '}
	}
	s.NextRow, s.NextCol = 0, 0
}

// Present writes ANSI for cells that differ from the last present.
// Consecutive dirty cells on a row with the same Reverse are emitted as one
// cursor move + run (keeps ASSIST/status leave snappy on wide terminals).
func (s *Screen) Present(w io.Writer) error {
	var b strings.Builder
	for r := 0; r < s.Rows; r++ {
		c := 0
		for c < s.Cols {
			i := s.at(r, c)
			if s.pending[i] == s.presented[i] {
				c++
				continue
			}
			cell := s.pending[i]
			end := c + 1
			for end < s.Cols {
				j := s.at(r, end)
				if s.pending[j] == s.presented[j] || s.pending[j].Reverse != cell.Reverse {
					break
				}
				end++
			}
			fmt.Fprintf(&b, "\x1b[%d;%dH", r+1, c+1)
			if cell.Reverse {
				b.WriteString("\x1b[7m")
			}
			for k := c; k < end; k++ {
				j := s.at(r, k)
				b.WriteRune(s.pending[j].Ch)
				s.presented[j] = s.pending[j]
			}
			if cell.Reverse {
				b.WriteString("\x1b[0m")
			}
			c = end
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// PresentClear clears the terminal then presents every cell.
func (s *Screen) PresentClear(w io.Writer) error {
	for i := range s.presented {
		s.presented[i] = Cell{Ch: 0} // force all different
	}
	if _, err := io.WriteString(w, "\x1b[2J\x1b[H"); err != nil {
		return err
	}
	if err := s.Present(w); err != nil {
		return err
	}
	s.NextRow, s.NextCol = 0, 0
	_, err := io.WriteString(w, "\x1b[H")
	return err
}

// ClearTerminal blanks the buffer and wipes the physical terminal to home,
// without redrawing the grid. Used when leaving full-screen commands so the
// dot prompt is not painted over leftover Assist/EDIT chrome.
func (s *Screen) ClearTerminal(w io.Writer) error {
	s.Clear()
	for i := range s.presented {
		s.presented[i] = Cell{Ch: ' '}
	}
	_, err := io.WriteString(w, "\x1b[2J\x1b[H")
	return err
}

// CellAt returns the pending cell (for tests).
func (s *Screen) CellAt(row, col int) Cell {
	if row < 0 || col < 0 || row >= s.Rows || col >= s.Cols {
		return Cell{}
	}
	return s.pending[s.at(row, col)]
}
