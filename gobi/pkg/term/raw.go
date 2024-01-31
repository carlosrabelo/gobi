package term

import (
	"io"

	"github.com/carlosrabelo/gobi/gobi/pkg/hostterm"
)

// IsTerminal reports whether r is attached to a host terminal.
func IsTerminal(r io.Reader) bool {
	return hostterm.IsTerminal(r)
}

// IsTerminalWriter reports whether w is attached to a host terminal.
func IsTerminalWriter(w io.Writer) bool {
	return hostterm.IsTerminalWriter(w)
}

// EnterRaw puts stdin into raw mode when r is a terminal.
// The returned restore must be called to put the terminal back.
// When r is not a terminal, restore is a no-op and err is nil.
func EnterRaw(r io.Reader) (restore func(), err error) {
	return hostterm.MakeRaw(r)
}
