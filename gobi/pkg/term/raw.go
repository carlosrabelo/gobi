package term

import (
	"io"
	"os"

	"golang.org/x/term"
)

// IsTerminal reports whether r is an *os.File attached to a terminal.
func IsTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// EnterRaw puts stdin into raw mode when r is a terminal file.
// The returned restore must be called to put the terminal back.
// When r is not a terminal, restore is a no-op and err is nil.
func EnterRaw(r io.Reader) (restore func(), err error) {
	f, ok := r.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
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
