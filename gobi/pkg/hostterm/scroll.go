package hostterm

import (
	"fmt"
	"io"
)

// SetScrollRegion limits scrolling to 1-based inclusive rows [top, bottom].
// Used so STATUS chrome at the bottom is not pushed by the dot prompt.
func SetScrollRegion(out io.Writer, top, bottom int) error {
	if top < 1 {
		top = 1
	}
	if bottom < top {
		bottom = top
	}
	_, err := fmt.Fprintf(out, "\x1b[%d;%dr", top, bottom)
	return err
}

// ResetScrollRegion restores full-screen scrolling (CSI r).
func ResetScrollRegion(out io.Writer) error {
	_, err := io.WriteString(out, "\x1b[r")
	return err
}
