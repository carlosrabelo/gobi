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
)

// Cell is one screen cell.
type Cell struct {
	Ch      byte
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
func (s *Screen) Resize(cols, rows int) {
	if cols < MinCols {
		cols = MinCols
	}
	if rows < MinRows {
		rows = MinRows
	}
	n := cols * rows
	s.Cols = cols
	s.Rows = rows
	s.pending = make([]Cell, n)
	s.presented = make([]Cell, n)
	for i := range s.pending {
		s.pending[i] = Cell{Ch: ' '}
		s.presented[i] = Cell{Ch: ' '}
	}
	s.NextRow, s.NextCol = 0, 0
}

func (s *Screen) at(r, c int) int {
	return r*s.Cols + c
}

// Put writes ch at (row, col). Outside the grid is a no-op.
func (s *Screen) Put(row, col int, ch byte, reverse bool) {
	if row < 0 || col < 0 || row >= s.Rows || col >= s.Cols {
		return
	}
	s.pending[s.at(row, col)] = Cell{Ch: ch, Reverse: reverse}
}

// WriteString writes text starting at (row, col). Advances NextRow/NextCol.
func (s *Screen) WriteString(row, col int, text string, reverse bool) {
	r, c := row, col
	for i := 0; i < len(text); i++ {
		s.Put(r, c, text[i], reverse)
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
func (s *Screen) Present(w io.Writer) error {
	var b strings.Builder
	first := true
	for r := 0; r < s.Rows; r++ {
		for c := 0; c < s.Cols; c++ {
			i := s.at(r, c)
			if s.pending[i] == s.presented[i] {
				continue
			}
			if first {
				// no full clear — only deltas
				first = false
			}
			cell := s.pending[i]
			fmt.Fprintf(&b, "\x1b[%d;%dH", r+1, c+1)
			if cell.Reverse {
				b.WriteString("\x1b[7m")
			}
			b.WriteByte(cell.Ch)
			if cell.Reverse {
				b.WriteString("\x1b[0m")
			}
			s.presented[i] = cell
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
	return s.Present(w)
}

// CellAt returns the pending cell (for tests).
func (s *Screen) CellAt(row, col int) Cell {
	if row < 0 || col < 0 || row >= s.Rows || col >= s.Cols {
		return Cell{}
	}
	return s.pending[s.at(row, col)]
}
