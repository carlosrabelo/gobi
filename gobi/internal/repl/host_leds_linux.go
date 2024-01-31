//go:build linux

package repl

import (
	"os"
	"sync"
	"syscall"
	"unsafe"
)

// Linux KDGETLED — keyboard LED flags (console/tty).
const (
	kdGetLED = 0x4B31
	ledCaps  = 0x04
	ledNum   = 0x02
)

var (
	ledMu                       sync.Mutex
	ledCapsCached, ledNumCached bool
)

func hostCapsOn() bool {
	probeLEDs()
	return ledCapsCached
}

func hostNumOn() bool {
	probeLEDs()
	return ledNumCached
}

func probeLEDs() {
	ledMu.Lock()
	defer ledMu.Unlock()
	// Re-read each paint so Caps/Num toggle while Gobi runs.
	ledCapsCached, ledNumCached = false, false
	f := os.Stdin
	if f == nil {
		return
	}
	var leds byte
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), kdGetLED, uintptr(unsafe.Pointer(&leds)))
	if errno != 0 {
		return // pts/SSH often cannot KDGETLED; leave indicators blank
	}
	ledCapsCached = leds&ledCaps != 0
	ledNumCached = leds&ledNum != 0
}
