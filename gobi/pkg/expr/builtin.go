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
	case "BOF":
		return builtinBOF(args, a)
	default:
		return Value{}, fmt.Errorf("%w: %s", ErrUnknownFunction, name)
	}
}

func requireArity(fn string, args []Value, n int) error {
	if len(args) != n {
		return fmt.Errorf("%s: wrong number of arguments", fn)
	}
	return nil
}

func builtinEOF(args []Value, a Area) (Value, error) {
	if err := requireArity("EOF", args, 0); err != nil {
		return Value{}, err
	}
	if !a.HasTable {
		return LogicalVal(true), nil
	}
	return LogicalVal(a.RecNo > a.RecCount), nil
}

func builtinBOF(args []Value, a Area) (Value, error) {
	if err := requireArity("BOF", args, 0); err != nil {
		return Value{}, err
	}
	if !a.HasTable {
		return LogicalVal(false), nil
	}
	return LogicalVal(a.BOF), nil
}
