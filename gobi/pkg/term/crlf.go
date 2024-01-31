package term

import "io"

// CRLFWriter translates lone LF to CRLF. Needed when the terminal is in raw
// mode: MakeRaw clears OPOST/ONLCR, so a bare '\n' advances the row without
// returning to column 0.
type CRLFWriter struct {
	W io.Writer
}

// Unwrap returns the underlying writer so host size/caps probes can reach the
// real *os.File after the interactive loop wraps stdout.
func (c *CRLFWriter) Unwrap() io.Writer {
	if c == nil {
		return nil
	}
	return c.W
}

func (c *CRLFWriter) Write(p []byte) (int, error) {
	if c == nil || c.W == nil {
		return 0, io.ErrClosedPipe
	}
	var buf []byte
	for i := 0; i < len(p); i++ {
		if p[i] == '\n' && (i == 0 || p[i-1] != '\r') {
			buf = append(buf, '\r', '\n')
			continue
		}
		buf = append(buf, p[i])
	}
	if _, err := c.W.Write(buf); err != nil {
		return 0, err
	}
	return len(p), nil
}
