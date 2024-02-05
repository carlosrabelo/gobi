//go:build !unix && !windows

package hostterm

import "sync"

var (
	resizeMu  sync.Mutex
	resizeFns []func(cols, rows int)
)

// OnResize registers fn for host window size changes. Unsupported hosts only
// store the callback for NotifyResizeForTest.
func OnResize(fd int, fn func(cols, rows int)) {
	if fn == nil {
		return
	}
	resizeMu.Lock()
	resizeFns = append(resizeFns, fn)
	resizeMu.Unlock()
	_ = fd
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
