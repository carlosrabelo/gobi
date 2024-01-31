//go:build !unix

package term

import "time"

// waitReadable is a no-op off Unix; callers fall back to SetReadDeadline or
// treat a missing follow-up byte after Esc as bare Esc.
func (kr *Reader) waitReadable(d time.Duration) bool {
	_ = d
	return false
}
