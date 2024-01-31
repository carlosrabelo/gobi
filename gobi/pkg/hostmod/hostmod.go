// Package hostmod holds LOAD/CALL session modules (spec 038).
package hostmod

import (
	"fmt"
	"sync"
)

// Func is a CALL entry point.
type Func func(args []string) error

var (
	mu       sync.Mutex
	builtins = map[string]Func{}
)

// RegisterBuiltin registers an in-process module for tests and hosts without .so plugins.
func RegisterBuiltin(name string, fn Func) {
	mu.Lock()
	defer mu.Unlock()
	builtins[name] = fn
}

// LookupBuiltin returns a registered builtin.
func LookupBuiltin(name string) (Func, bool) {
	mu.Lock()
	defer mu.Unlock()
	fn, ok := builtins[name]
	return fn, ok
}

// ClearBuiltins removes all builtins (tests).
func ClearBuiltins() {
	mu.Lock()
	defer mu.Unlock()
	builtins = map[string]Func{}
}

// ErrMissing is returned when CALL cannot find an entry.
func ErrMissing(name string) error {
	return fmt.Errorf("module entry %s not found", name)
}
