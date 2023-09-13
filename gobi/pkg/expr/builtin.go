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
	case "RECNO":
		return builtinRECNO(args, a)
	case "DELETED":
		return builtinDELETED(args, a)
	case "FOUND":
		return builtinFOUND(args, a)
	case "TRIM":
		return builtinTRIM(args)
	case "UPPER":
		return builtinUPPER(args)
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

func builtinRECNO(args []Value, a Area) (Value, error) {
	if err := requireArity("RECNO", args, 0); err != nil {
		return Value{}, err
	}
	if !a.HasTable {
		return NumberVal(0), nil
	}
	return NumberVal(float64(a.RecNo)), nil
}

func hasCurrentRecord(a Area) bool {
	return a.HasTable && a.RecNo >= 1 && a.RecNo <= a.RecCount
}

func builtinDELETED(args []Value, a Area) (Value, error) {
	if err := requireArity("DELETED", args, 0); err != nil {
		return Value{}, err
	}
	if !hasCurrentRecord(a) {
		return LogicalVal(false), nil
	}
	return LogicalVal(a.Deleted), nil
}

func builtinFOUND(args []Value, a Area) (Value, error) {
	if err := requireArity("FOUND", args, 0); err != nil {
		return Value{}, err
	}
	return LogicalVal(a.Found), nil
}

func builtinTRIM(args []Value) (Value, error) {
	if err := requireArity("TRIM", args, 1); err != nil {
		return Value{}, err
	}
	return StringVal(strings.TrimRight(args[0].String(), " ")), nil
}

func builtinUPPER(args []Value) (Value, error) {
	if err := requireArity("UPPER", args, 1); err != nil {
		return Value{}, err
	}
	return StringVal(strings.ToUpper(args[0].String())), nil
}
