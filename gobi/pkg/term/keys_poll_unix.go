//go:build unix

package term

import (
	"time"

	"golang.org/x/sys/unix"
)

// waitReadable reports whether kr.file has data within d (0 = immediate poll).
func (kr *Reader) waitReadable(d time.Duration) bool {
	if kr.file == nil {
		return false
	}
	fd := int(kr.file.Fd())
	ms := int(d / time.Millisecond)
	if d > 0 && ms == 0 {
		ms = 1
	}
	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	n, err := unix.Poll(fds, ms)
	if err != nil || n <= 0 {
		return false
	}
	return fds[0].Revents&(unix.POLLIN|unix.POLLERR|unix.POLLHUP) != 0
}
