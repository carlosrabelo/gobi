//go:build !unix

package repl

import (
	"os"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func tryFileLock(ctx *context.Context) bool {
	a := ctx.Active()
	if !a.HasTable() || a.Path == "" {
		return false
	}
	if a.Locked {
		return true
	}
	// Best-effort: hold an exclusive open handle.
	f, err := os.OpenFile(a.Path, os.O_RDWR, 0)
	if err != nil {
		return false
	}
	a.LockFile = f
	a.Locked = true
	return true
}

func releaseAreaLock(a *context.Area) {
	if a == nil {
		return
	}
	if a.LockFile != nil {
		_ = a.LockFile.Close()
		a.LockFile = nil
	}
	a.Locked = false
}
