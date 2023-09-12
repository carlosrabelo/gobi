package expr

import (
	"errors"
	"fmt"
	"strings"
)

// ErrUnknownFunction means the name is not a built-in in this package.
var ErrUnknownFunction = errors.New("unknown function")

// CallBuiltin evaluates a built-in by name. Names match without regard to case.
func CallBuiltin(name string, args []Value, a Area) (Value, error) {
	switch strings.ToUpper(name) {
	case "EOF":
		return builtinEOF(args, a)
	default:
		return Value{}, fmt.Errorf("%w: %s", ErrUnknownFunction, name)
	}
}

func builtinEOF(args []Value, a Area) (Value, error) {
	if len(args) != 0 {
		return Value{}, fmt.Errorf("EOF: wrong number of arguments")
	}
	if !a.HasTable {
		return LogicalVal(true), nil
	}
	return LogicalVal(a.RecNo > a.RecCount), nil
}
