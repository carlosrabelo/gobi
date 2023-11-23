package expr

import "time"

// Env resolves memory variables, fields, and function calls.
type Env interface {
	// Memory returns a memory variable. ok is false if the name is absent.
	Memory(name string) (Value, bool)
	// Field returns a field. An empty alias means the active work area.
	Field(alias, name string) (Value, bool)
	// Call evaluates a function by name with already-evaluated arguments.
	Call(name string, args []Value) (Value, error)
	// Exact reports SET EXACT. ok is false when the environment does not report a mode.
	Exact() (on bool, ok bool)
	// Default returns SET DEFAULT. ok is false when the environment does not report a path.
	Default() (path string, ok bool)
	// DateSettings returns SET DATE order and SET CENTURY.
	DateSettings() (order string, century bool, ok bool)
	// Today returns the session DATE().
	Today() (day time.Time, ok bool)
	// LUpdate returns the open table's header date, or empty date when none.
	LUpdate() Value
}
