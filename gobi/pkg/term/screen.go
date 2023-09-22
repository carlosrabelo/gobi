package term

import (
	"fmt"
	"io"
	"os"
)

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

// IsTerminal is a stub until the raw-mode keyboard adapter lands.
func IsTerminal(file *os.File) bool { return false }

var ErrNotTerminal = fmt.Errorf("term: not a terminal")

type RawMode struct{}

func EnterRawMode(in *os.File) (*RawMode, error) { return nil, ErrNotTerminal }
func (m *RawMode) Close() error                  { return nil }

type KeyKind int

const (
	KeyByte KeyKind = iota
	KeyEnter
	KeyBackspace
	KeyEscape
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyShiftTab
)

type Key struct {
	Kind KeyKind
	Byte byte
}

type Keyboard struct{}

func NewKeyboard(in io.Reader) *Keyboard { return &Keyboard{} }
func (k *Keyboard) ReadKey() (Key, error) {
	return Key{}, io.EOF
}

func EraseLine(w io.Writer) error { return nil }
