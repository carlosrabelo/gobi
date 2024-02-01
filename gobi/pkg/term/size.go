package term

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// winsize mirrors the kernel structure filled by the TIOCGWINSZ ioctl.
type winsize struct {
	rows uint16
	cols uint16
	x    uint16
	y    uint16
}

// Size returns the terminal dimensions in columns and rows for file.
// It fails with ErrNotTerminal when file is nil or not a terminal device.
func Size(file *os.File) (cols, rows int, err error) {
	if file == nil {
		return 0, 0, ErrNotTerminal
	}

	var ws winsize
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, file.Fd(), syscall.TIOCGWINSZ,
		uintptr(unsafe.Pointer(&ws)), 0, 0, 0)
	if errno != 0 {
		return 0, 0, ErrNotTerminal
	}
	if ws.cols == 0 || ws.rows == 0 {
		return 0, 0, fmt.Errorf("term: terminal reported zero size")
	}
	return int(ws.cols), int(ws.rows), nil
}
