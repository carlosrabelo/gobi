package expr

import (
	"errors"
	"fmt"
	"strconv"
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
	case "LOWER":
		return builtinLOWER(args)
	case "LEN":
		return builtinLEN(args)
	case "SUBSTR":
		return builtinSUBSTR(args)
	case "VAL":
		return builtinVAL(args)
	case "STR":
		return builtinSTR(args)
	case "CHR":
		return builtinCHR(args)
	case "RECCOUNT":
		return builtinRECCOUNT(args, a)
	case "RECSIZE":
		return builtinRECSIZE(args, a)
	case "FIELD":
		return builtinFIELD(args, a)
	case "DBF":
		return builtinDBF(args, a)
	case "NDX":
		return builtinNDX(args, a)
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

func builtinLOWER(args []Value) (Value, error) {
	if err := requireArity("LOWER", args, 1); err != nil {
		return Value{}, err
	}
	return StringVal(strings.ToLower(args[0].String())), nil
}

func builtinLEN(args []Value) (Value, error) {
	if err := requireArity("LEN", args, 1); err != nil {
		return Value{}, err
	}
	return NumberVal(float64(len(args[0].String()))), nil
}

func requireNumber(fn string, v Value) (float64, error) {
	n, err := asNumber(v)
	if err != nil {
		return 0, fmt.Errorf("%s: wrong type", fn)
	}
	return n, nil
}

func builtinSUBSTR(args []Value) (Value, error) {
	if err := requireArity("SUBSTR", args, 3); err != nil {
		return Value{}, err
	}
	s := args[0].String()
	startF, err := requireNumber("SUBSTR", args[1])
	if err != nil {
		return Value{}, err
	}
	lenF, err := requireNumber("SUBSTR", args[2])
	if err != nil {
		return Value{}, err
	}
	start := int(startF)
	length := int(lenF)
	if start < 1 {
		start = 1
	}
	if length < 1 || start > len(s) {
		return StringVal(""), nil
	}
	i := start - 1
	end := i + length
	if end > len(s) {
		end = len(s)
	}
	return StringVal(s[i:end]), nil
}

func builtinVAL(args []Value) (Value, error) {
	if err := requireArity("VAL", args, 1); err != nil {
		return Value{}, err
	}
	s := strings.TrimSpace(args[0].String())
	if s == "" {
		return NumberVal(0), nil
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return NumberVal(0), nil
	}
	return NumberVal(n), nil
}

func builtinSTR(args []Value) (Value, error) {
	if len(args) < 1 || len(args) > 3 {
		return Value{}, fmt.Errorf("STR: wrong number of arguments")
	}
	n, err := requireNumber("STR", args[0])
	if err != nil {
		return Value{}, err
	}
	width := 10
	decimals := 0
	if len(args) >= 2 {
		w, err := requireNumber("STR", args[1])
		if err != nil {
			return Value{}, err
		}
		width = int(w)
	}
	if len(args) >= 3 {
		d, err := requireNumber("STR", args[2])
		if err != nil {
			return Value{}, err
		}
		decimals = int(d)
	}
	if width < 1 {
		return Value{}, fmt.Errorf("STR: wrong type")
	}
	s := fmt.Sprintf("%*.*f", width, decimals, n)
	if len(s) > width {
		return StringVal(strings.Repeat("*", width)), nil
	}
	return StringVal(s), nil
}

func builtinCHR(args []Value) (Value, error) {
	if err := requireArity("CHR", args, 1); err != nil {
		return Value{}, err
	}
	n, err := requireNumber("CHR", args[0])
	if err != nil {
		return Value{}, err
	}
	i := int(n)
	if float64(i) != n || i < 0 || i > 255 {
		return Value{}, fmt.Errorf("CHR: out of range")
	}
	return StringVal(string(byte(i))), nil
}

func builtinRECCOUNT(args []Value, a Area) (Value, error) {
	if err := requireArity("RECCOUNT", args, 0); err != nil {
		return Value{}, err
	}
	if !a.HasTable {
		return NumberVal(0), nil
	}
	return NumberVal(float64(a.RecCount)), nil
}

func builtinRECSIZE(args []Value, a Area) (Value, error) {
	if err := requireArity("RECSIZE", args, 0); err != nil {
		return Value{}, err
	}
	if !a.HasTable {
		return NumberVal(0), nil
	}
	return NumberVal(float64(a.RecSize)), nil
}

func builtinFIELD(args []Value, a Area) (Value, error) {
	if err := requireArity("FIELD", args, 1); err != nil {
		return Value{}, err
	}
	n, err := requireNumber("FIELD", args[0])
	if err != nil {
		return Value{}, err
	}
	i := int(n)
	if i < 1 || i > len(a.Fields) {
		return StringVal(""), nil
	}
	return StringVal(strings.ToUpper(a.Fields[i-1])), nil
}

func builtinDBF(args []Value, a Area) (Value, error) {
	if err := requireArity("DBF", args, 0); err != nil {
		return Value{}, err
	}
	if !a.HasTable {
		return StringVal(""), nil
	}
	return StringVal(a.DBF), nil
}

func builtinNDX(args []Value, a Area) (Value, error) {
	if err := requireArity("NDX", args, 1); err != nil {
		return Value{}, err
	}
	n, err := requireNumber("NDX", args[0])
	if err != nil {
		return Value{}, err
	}
	i := int(n)
	if i < 1 || i > 7 {
		return StringVal(""), nil
	}
	return StringVal(a.NDX[i-1]), nil
}
