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

	// ColorDefault means host default fg/bg (no SET COLOR pair applied).
	ColorDefault = -1
)

// StatusSep is the segment separator painted on the status bar.
// Prefer the double-line glyph; hosts that only render ASCII still show a vertical bar cell.
var StatusSep = BoxVDouble

// Cell is one screen cell. Ch is a Unicode code point (one column).
// FG/BG are 0–15 ANSI indexes, or ColorDefault (-1).
type Cell struct {
	Ch      rune
	Reverse bool
	FG, BG  int
}

// Screen is a double-buffered character grid.
type Screen struct {
	Cols, Rows int
	pending    []Cell
	presented  []Cell
	// NextRow/NextCol are 0-based cursor for ROW()/COL() after writes.
	NextRow, NextCol int

	// ColorActive applies Standard/Enhanced pairs on Put when true (SET COLOR ON + Caps).
	ColorActive bool
	StandardFG  int
	StandardBG  int
	EnhancedFG  int
	EnhancedBG  int
}

// New returns an 80x24 screen filled with spaces.
func New() *Screen {
	return NewSize(DefaultCols, DefaultRows)
}

// NewSize returns a screen of the given size, clamped to floors.
func NewSize(cols, rows int) *Screen {
	s := &Screen{
		StandardFG: ColorDefault,
		StandardBG: ColorDefault,
		EnhancedFG: ColorDefault,
		EnhancedBG: ColorDefault,
	}
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
	blank := Cell{Ch: ' ', FG: ColorDefault, BG: ColorDefault}
	for i := range s.pending {
		s.pending[i] = blank
		s.presented[i] = blank
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

func (s *Screen) attrsFor(reverse bool) (fg, bg int) {
	if !s.ColorActive {
		return ColorDefault, ColorDefault
	}
	if reverse {
		return s.EnhancedFG, s.EnhancedBG
	}
	return s.StandardFG, s.StandardBG
}

// Put writes ch at (row, col). Outside the grid is a no-op.
func (s *Screen) Put(row, col int, ch rune, reverse bool) {
	if row < 0 || col < 0 || row >= s.Rows || col >= s.Cols {
		return
	}
	if ch == 0 {
		ch = ' '
	}
	fg, bg := s.attrsFor(reverse)
	s.pending[s.at(row, col)] = Cell{Ch: ch, Reverse: reverse, FG: fg, BG: bg}
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
	blank := Cell{Ch: ' ', FG: ColorDefault, BG: ColorDefault}
	if s.ColorActive {
		blank.FG, blank.BG = s.StandardFG, s.StandardBG
	}
	for i := range s.pending {
		s.pending[i] = blank
	}
	s.NextRow, s.NextCol = 0, 0
}

// RefreshColorAttrs rewrites FG/BG on every pending cell from the current scheme.
func (s *Screen) RefreshColorAttrs() {
	for i := range s.pending {
		fg, bg := s.attrsFor(s.pending[i].Reverse)
		s.pending[i].FG = fg
		s.pending[i].BG = bg
	}
}

// Present writes ANSI for cells that differ from the last present.
// Consecutive dirty cells on a row with the same Reverse/FG/BG are one run.
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
				p := s.pending[j]
				if p == s.presented[j] || p.Reverse != cell.Reverse || p.FG != cell.FG || p.BG != cell.BG {
					break
				}
				end++
			}
			fmt.Fprintf(&b, "\x1b[%d;%dH", r+1, c+1)
			writeCellAttrs(&b, cell)
			for k := c; k < end; k++ {
				j := s.at(r, k)
				b.WriteRune(s.pending[j].Ch)
				s.presented[j] = s.pending[j]
			}
			b.WriteString("\x1b[0m")
			c = end
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func writeCellAttrs(b *strings.Builder, cell Cell) {
	if cell.Reverse {
		b.WriteString("\x1b[7m")
	}
	if cell.FG == ColorDefault && cell.BG == ColorDefault {
		return
	}
	if cell.FG >= 0 && cell.FG <= 15 {
		code := 30 + cell.FG
		if cell.FG >= 8 {
			code = 90 + (cell.FG - 8)
		}
		fmt.Fprintf(b, "\x1b[%dm", code)
	}
	if cell.BG >= 0 && cell.BG <= 15 {
		code := 40 + cell.BG
		if cell.BG >= 8 {
			code = 100 + (cell.BG - 8)
		}
		fmt.Fprintf(b, "\x1b[%dm", code)
	}
}

// PresentClear clears the terminal then presents every cell.
func (s *Screen) PresentClear(w io.Writer) error {
	for i := range s.presented {
		s.presented[i] = Cell{Ch: 0, FG: ColorDefault, BG: ColorDefault} // force dirty
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
	blank := Cell{Ch: ' ', FG: ColorDefault, BG: ColorDefault}
	for i := range s.presented {
		s.presented[i] = blank
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
