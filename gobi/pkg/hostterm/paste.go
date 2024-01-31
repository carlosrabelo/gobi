package hostterm

import "io"

// EnableBracketedPaste asks the host terminal to wrap pasted text in
// CSI 200 ~ … CSI 201 ~ so the session can accept clipboard paste in raw mode.
func EnableBracketedPaste(out io.Writer) error {
	_, err := io.WriteString(out, "\x1b[?2004h")
	return err
}

// DisableBracketedPaste turns bracketed paste off (pair with EnableBracketedPaste).
func DisableBracketedPaste(out io.Writer) error {
	_, err := io.WriteString(out, "\x1b[?2004l")
	return err
}
