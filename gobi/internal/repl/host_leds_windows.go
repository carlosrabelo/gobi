//go:build windows

package repl

import (
	"sync"

	"golang.org/x/sys/windows"
)

const (
	vkCapital = 0x14
	vkNumLock = 0x90
)

var (
	ledMu                       sync.Mutex
	user32                      = windows.NewLazySystemDLL("user32.dll")
	procGetKeyState             = user32.NewProc("GetKeyState")
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
	ledCapsCached = keyToggleOn(vkCapital)
	ledNumCached = keyToggleOn(vkNumLock)
}

func keyToggleOn(vk uintptr) bool {
	r, _, err := procGetKeyState.Call(vk)
	if err != windows.ERROR_SUCCESS && err != nil {
		// LazyDLL Call returns err for non-zero GetLastError; low bit is still valid.
		_ = err
	}
	return r&1 != 0
}
