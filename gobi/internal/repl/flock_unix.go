//go:build unix

package repl

import (
	"os"
	"syscall"

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
	f, err := os.OpenFile(a.Path, os.O_RDWR, 0)
	if err != nil {
		return false
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
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
		_ = syscall.Flock(int(a.LockFile.Fd()), syscall.LOCK_UN)
		_ = a.LockFile.Close()
		a.LockFile = nil
	}
	a.Locked = false
}
