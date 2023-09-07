package term

const (
	DefaultCols = 80
	DefaultRows = 24
)

// Screen is a stub buffer until the VT100 TUI milestone.
type Screen struct {
	cols int
	rows int
}

func NewScreen(cols, rows int) *Screen {
	if cols <= 0 {
		cols = DefaultCols
	}
	if rows <= 0 {
		rows = DefaultRows
	}
	return &Screen{cols: cols, rows: rows}
}

func (s *Screen) Cols() int { return s.cols }
func (s *Screen) Rows() int { return s.rows }
