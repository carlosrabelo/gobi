//go:build unix

package hostterm

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	resizeMu   sync.Mutex
	resizeFns  []func(cols, rows int)
	resizeOnce sync.Once
)

// OnResize registers fn to run when the host window size changes (SIGWINCH).
// fd is reserved for Windows parity; on Unix the process signal is used.
func OnResize(fd int, fn func(cols, rows int)) {
	if fn == nil {
		return
	}
	resizeMu.Lock()
	resizeFns = append(resizeFns, fn)
	resizeMu.Unlock()
	resizeOnce.Do(func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGWINCH)
		go func() {
			for range ch {
				c, r := Size(fd)
				c, r = ApplyAuto(c, r)
				resizeMu.Lock()
				fns := append([]func(cols, rows int){}, resizeFns...)
				resizeMu.Unlock()
				for _, f := range fns {
					f(c, r)
				}
			}
		}()
	})
}

// NotifyResizeForTest invokes registered callbacks with cols, rows (tests only).
func NotifyResizeForTest(cols, rows int) {
	cols, rows = ApplyAuto(cols, rows)
	resizeMu.Lock()
	fns := append([]func(cols, rows int){}, resizeFns...)
	resizeMu.Unlock()
	for _, f := range fns {
		f(cols, rows)
	}
}
