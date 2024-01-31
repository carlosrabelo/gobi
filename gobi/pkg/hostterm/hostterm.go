// Package hostterm probes the host terminal (POSIX and Windows) for size,
// capabilities, and raw mode. Spec 003.
package hostterm

import (
	"io"
	"os"

	"golang.org/x/term"
)

const (
	DefaultCols = 80
	DefaultRows = 24
	MinCols     = 40
	MinRows     = 10
)

// Caps describes what the host can paint.
type Caps struct {
	CursorAddress bool
	Clear         bool
	Reverse       bool
	Color         bool
	TrueColor     bool
}

// IsTerminal reports whether r is an *os.File attached to a terminal.
func IsTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// IsTerminalFile reports whether f is a terminal.
func IsTerminalFile(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// Size returns the host columns and rows for fd. On failure or non-terminal,
// returns DefaultCols by DefaultRows. Success is not clamped to 80x24.
func Size(fd int) (cols, rows int) {
	w, h, err := term.GetSize(fd)
	if err != nil || w <= 0 || h <= 0 {
		return DefaultCols, DefaultRows
	}
	return w, h
}

// SizeOf returns Size for a reader when it is a terminal file; otherwise defaults.
func SizeOf(r io.Reader) (cols, rows int) {
	f := FileFromReader(r)
	if f == nil || !term.IsTerminal(int(f.Fd())) {
		return DefaultCols, DefaultRows
	}
	return Size(int(f.Fd()))
}

// SizeOfWriter is like SizeOf but for an output stream (stdout).
func SizeOfWriter(w io.Writer) (cols, rows int) {
	f := FileFromWriter(w)
	if f == nil || !term.IsTerminal(int(f.Fd())) {
		return DefaultCols, DefaultRows
	}
	return Size(int(f.Fd()))
}

// IsTerminalWriter reports whether w is a terminal file.
func IsTerminalWriter(w io.Writer) bool {
	f := FileFromWriter(w)
	return f != nil && term.IsTerminal(int(f.Fd()))
}

// FileFromWriter peels Unwrap() wrappers (e.g. CRLFWriter) to reach *os.File.
func FileFromWriter(w io.Writer) *os.File {
	for w != nil {
		if f, ok := w.(*os.File); ok {
			return f
		}
		u, ok := w.(interface{ Unwrap() io.Writer })
		if !ok {
			return nil
		}
		next := u.Unwrap()
		if next == nil || next == w {
			return nil
		}
		w = next
	}
	return nil
}

// FileFromReader peels Unwrap() wrappers to reach *os.File.
func FileFromReader(r io.Reader) *os.File {
	for r != nil {
		if f, ok := r.(*os.File); ok {
			return f
		}
		u, ok := r.(interface{ Unwrap() io.Reader })
		if !ok {
			return nil
		}
		next := u.Unwrap()
		if next == nil || next == r {
			return nil
		}
		r = next
	}
	return nil
}

// ApplyAuto raises dimensions below the AUTO floors; there is no upper clamp.
func ApplyAuto(cols, rows int) (int, int) {
	if cols < MinCols {
		cols = MinCols
	}
	if rows < MinRows {
		rows = MinRows
	}
	return cols, rows
}

// CapsOf returns capability flags for the writer/reader pair.
// A non-terminal has every flag false. A terminal assumes ANSI/console basics.
func CapsOf(out io.Writer) Caps {
	f := FileFromWriter(out)
	if f == nil || !term.IsTerminal(int(f.Fd())) {
		return Caps{}
	}
	return Caps{
		CursorAddress: true,
		Clear:         true,
		Reverse:       true,
		Color:         true,
		TrueColor:     false,
	}
}

// MakeRaw puts a terminal stdin into raw mode. Restore undoes it.
// Non-terminal is a no-op.
func MakeRaw(r io.Reader) (restore func(), err error) {
	f := FileFromReader(r)
	if f == nil || !term.IsTerminal(int(f.Fd())) {
		return func() {}, nil
	}
	fd := int(f.Fd())
	old, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return func() {
		_ = term.Restore(fd, old)
	}, nil
}
