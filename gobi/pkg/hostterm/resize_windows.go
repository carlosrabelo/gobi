//go:build windows

package hostterm

import (
	"sync"
	"time"
)

var (
	resizeMu   sync.Mutex
	resizeFns  []func(cols, rows int)
	resizeOnce sync.Once
)

// OnResize registers fn for console buffer size changes. Windows has no
// SIGWINCH; a short poll of GetSize (via golang.org/x/term) detects resizes.
func OnResize(fd int, fn func(cols, rows int)) {
	if fn == nil {
		return
	}
	resizeMu.Lock()
	resizeFns = append(resizeFns, fn)
	resizeMu.Unlock()
	resizeOnce.Do(func() {
		go pollConsoleResize(fd)
	})
}

func pollConsoleResize(fd int) {
	lastC, lastR := Size(fd)
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for range tick.C {
		c, r := Size(fd)
		if c == lastC && r == lastR {
			continue
		}
		lastC, lastR = c, r
		c, r = ApplyAuto(c, r)
		resizeMu.Lock()
		fns := append([]func(cols, rows int){}, resizeFns...)
		resizeMu.Unlock()
		for _, f := range fns {
			f(c, r)
		}
	}
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
