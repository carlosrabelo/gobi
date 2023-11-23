package expr

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ErrUnknownFunction means the name is not a built-in in this package.
var ErrUnknownFunction = errors.New("unknown function")

// CallBuiltin evaluates a built-in by name. Names match without regard to case.
func CallBuiltin(name string, args []Value, a Area, env Env) (Value, error) {
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
	case "TRIM", "RTRIM":
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
	case "ASC":
		return builtinASC(args)
	case "AT":
		return builtinAT(args)
	case "LEFT":
		return builtinLEFT(args)
	case "RIGHT":
		return builtinRIGHT(args)
	case "LTRIM":
		return builtinLTRIM(args)
	case "SPACE":
		return builtinSPACE(args)
	case "REPLICATE":
		return builtinREPLICATE(args)
	case "STUFF":
		return builtinSTUFF(args)
	case "FILE":
		return builtinFILE(args, env)
	case "INT":
		return builtinINT(args)
	case "ROUND":
		return builtinROUND(args)
	case "ABS":
		return builtinABS(args)
	case "MAX":
		return builtinMAX(args)
	case "MIN":
		return builtinMIN(args)
	case "MOD":
		return builtinMOD(args)
	case "SQRT":
		return builtinSQRT(args)
	case "LOG":
		return builtinLOG(args)
	case "EXP":
		return builtinEXP(args)
	case "TYPE":
		return builtinTYPE(args, env)
	case "IIF":
		return builtinIIF(args)
	case "DISKSPACE":
		return builtinDISKSPACE(args, env)
	case "DATE":
		return builtinDATE(args, env)
	case "CTOD":
		return builtinCTOD(args, env)
	case "DTOC":
		return builtinDTOC(args, env)
	case "DAY":
		return builtinDAY(args)
	case "MONTH":
		return builtinMONTH(args)
	case "YEAR":
		return builtinYEAR(args)
	case "CMONTH":
		return builtinCMONTH(args)
	case "CDOW":
		return builtinCDOW(args)
	case "DOW":
		return builtinDOW(args)
	case "TIME":
		return builtinTIME(args)
	case "LUPDATE":
		return builtinLUPDATE(args, env)
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

func builtinASC(args []Value) (Value, error) {
	if err := requireArity("ASC", args, 1); err != nil {
		return Value{}, err
	}
	s := args[0].String()
	if s == "" {
		return Value{}, fmt.Errorf("ASC: empty string")
	}
	return NumberVal(float64(s[0])), nil
}

func builtinAT(args []Value) (Value, error) {
	if err := requireArity("AT", args, 2); err != nil {
		return Value{}, err
	}
	sub := args[0].String()
	s := args[1].String()
	if sub == "" {
		return NumberVal(0), nil
	}
	i := strings.Index(s, sub)
	if i < 0 {
		return NumberVal(0), nil
	}
	return NumberVal(float64(i + 1)), nil
}

func builtinLEFT(args []Value) (Value, error) {
	if err := requireArity("LEFT", args, 2); err != nil {
		return Value{}, err
	}
	s := args[0].String()
	nF, err := requireNumber("LEFT", args[1])
	if err != nil {
		return Value{}, err
	}
	n := int(nF)
	if n < 1 {
		return StringVal(""), nil
	}
	if n > len(s) {
		n = len(s)
	}
	return StringVal(s[:n]), nil
}

func builtinRIGHT(args []Value) (Value, error) {
	if err := requireArity("RIGHT", args, 2); err != nil {
		return Value{}, err
	}
	s := args[0].String()
	nF, err := requireNumber("RIGHT", args[1])
	if err != nil {
		return Value{}, err
	}
	n := int(nF)
	if n < 1 {
		return StringVal(""), nil
	}
	if n > len(s) {
		n = len(s)
	}
	return StringVal(s[len(s)-n:]), nil
}

func builtinLTRIM(args []Value) (Value, error) {
	if err := requireArity("LTRIM", args, 1); err != nil {
		return Value{}, err
	}
	return StringVal(strings.TrimLeft(args[0].String(), " ")), nil
}

func builtinSPACE(args []Value) (Value, error) {
	if err := requireArity("SPACE", args, 1); err != nil {
		return Value{}, err
	}
	nF, err := requireNumber("SPACE", args[0])
	if err != nil {
		return Value{}, err
	}
	n := int(nF)
	if n < 0 {
		n = 0
	}
	if n > 254 {
		n = 254
	}
	return StringVal(strings.Repeat(" ", n)), nil
}

func builtinREPLICATE(args []Value) (Value, error) {
	if err := requireArity("REPLICATE", args, 2); err != nil {
		return Value{}, err
	}
	s := args[0].String()
	nF, err := requireNumber("REPLICATE", args[1])
	if err != nil {
		return Value{}, err
	}
	n := int(nF)
	if n < 1 || s == "" {
		return StringVal(""), nil
	}
	var b strings.Builder
	for i := 0; i < n && b.Len() < 254; i++ {
		remain := 254 - b.Len()
		if remain >= len(s) {
			b.WriteString(s)
		} else {
			b.WriteString(s[:remain])
		}
	}
	return StringVal(b.String()), nil
}

func builtinSTUFF(args []Value) (Value, error) {
	if err := requireArity("STUFF", args, 4); err != nil {
		return Value{}, err
	}
	s := args[0].String()
	startF, err := requireNumber("STUFF", args[1])
	if err != nil {
		return Value{}, err
	}
	nF, err := requireNumber("STUFF", args[2])
	if err != nil {
		return Value{}, err
	}
	rep := args[3].String()
	start := int(startF)
	n := int(nF)
	if start < 1 {
		start = 1
	}
	if n < 0 {
		n = 0
	}
	i := start - 1
	if i > len(s) {
		i = len(s)
	}
	end := i + n
	if end > len(s) {
		end = len(s)
	}
	return StringVal(s[:i] + rep + s[end:]), nil
}

func builtinFILE(args []Value, env Env) (Value, error) {
	if err := requireArity("FILE", args, 1); err != nil {
		return Value{}, err
	}
	name := args[0].String()
	base := "."
	if env != nil {
		if p, ok := env.Default(); ok && p != "" {
			base = p
		}
	}
	path := name
	if !filepath.IsAbs(name) {
		path = filepath.Join(base, name)
	}
	_, err := os.Stat(path)
	return LogicalVal(err == nil), nil
}

func builtinINT(args []Value) (Value, error) {
	if err := requireArity("INT", args, 1); err != nil {
		return Value{}, err
	}
	n, err := requireNumber("INT", args[0])
	if err != nil {
		return Value{}, err
	}
	return NumberVal(math.Trunc(n)), nil
}

func builtinROUND(args []Value) (Value, error) {
	if err := requireArity("ROUND", args, 2); err != nil {
		return Value{}, err
	}
	n, err := requireNumber("ROUND", args[0])
	if err != nil {
		return Value{}, err
	}
	dF, err := requireNumber("ROUND", args[1])
	if err != nil {
		return Value{}, err
	}
	d := int(dF)
	pow := math.Pow(10, float64(d))
	return NumberVal(math.Round(n*pow) / pow), nil
}

func builtinABS(args []Value) (Value, error) {
	if err := requireArity("ABS", args, 1); err != nil {
		return Value{}, err
	}
	n, err := requireNumber("ABS", args[0])
	if err != nil {
		return Value{}, err
	}
	return NumberVal(math.Abs(n)), nil
}

func builtinMAX(args []Value) (Value, error) {
	if err := requireArity("MAX", args, 2); err != nil {
		return Value{}, err
	}
	return maxMin(args[0], args[1], true)
}

func builtinMIN(args []Value) (Value, error) {
	if err := requireArity("MIN", args, 2); err != nil {
		return Value{}, err
	}
	return maxMin(args[0], args[1], false)
}

func maxMin(a, b Value, wantMax bool) (Value, error) {
	if a.Kind == KindNumber && b.Kind == KindNumber {
		if wantMax {
			if a.Num >= b.Num {
				return a, nil
			}
			return b, nil
		}
		if a.Num <= b.Num {
			return a, nil
		}
		return b, nil
	}
	// Dates compare as days when both are dates (KindDate added in 026).
	if a.Kind == KindDate && b.Kind == KindDate {
		cmp := compareDates(a, b)
		if wantMax {
			if cmp >= 0 {
				return a, nil
			}
			return b, nil
		}
		if cmp <= 0 {
			return a, nil
		}
		return b, nil
	}
	as := a.String()
	bs := b.String()
	if wantMax {
		if as >= bs {
			return a, nil
		}
		return b, nil
	}
	if as <= bs {
		return a, nil
	}
	return b, nil
}

func builtinMOD(args []Value) (Value, error) {
	if err := requireArity("MOD", args, 2); err != nil {
		return Value{}, err
	}
	a, err := requireNumber("MOD", args[0])
	if err != nil {
		return Value{}, err
	}
	b, err := requireNumber("MOD", args[1])
	if err != nil {
		return Value{}, err
	}
	if b == 0 {
		return Value{}, fmt.Errorf("MOD: division by zero")
	}
	return NumberVal(math.Mod(a, b)), nil
}

func builtinSQRT(args []Value) (Value, error) {
	if err := requireArity("SQRT", args, 1); err != nil {
		return Value{}, err
	}
	n, err := requireNumber("SQRT", args[0])
	if err != nil {
		return Value{}, err
	}
	if n < 0 {
		return Value{}, fmt.Errorf("SQRT: negative argument")
	}
	return NumberVal(math.Sqrt(n)), nil
}

func builtinLOG(args []Value) (Value, error) {
	if err := requireArity("LOG", args, 1); err != nil {
		return Value{}, err
	}
	n, err := requireNumber("LOG", args[0])
	if err != nil {
		return Value{}, err
	}
	if n <= 0 {
		return Value{}, fmt.Errorf("LOG: non-positive argument")
	}
	return NumberVal(math.Log(n)), nil
}

func builtinEXP(args []Value) (Value, error) {
	if err := requireArity("EXP", args, 1); err != nil {
		return Value{}, err
	}
	n, err := requireNumber("EXP", args[0])
	if err != nil {
		return Value{}, err
	}
	return NumberVal(math.Exp(n)), nil
}

func builtinTYPE(args []Value, env Env) (Value, error) {
	if err := requireArity("TYPE", args, 1); err != nil {
		return Value{}, err
	}
	src := args[0].String()
	tree, errs := Parse(src)
	if len(errs) != 0 || tree == nil {
		return StringVal("U"), nil
	}
	v, err := Eval(tree, env)
	if err != nil {
		return StringVal("U"), nil
	}
	switch v.Kind {
	case KindString:
		return StringVal("C"), nil
	case KindNumber:
		return StringVal("N"), nil
	case KindLogical:
		return StringVal("L"), nil
	case KindDate:
		return StringVal("D"), nil
	case KindMemo:
		return StringVal("M"), nil
	default:
		return StringVal("U"), nil
	}
}

func builtinIIF(args []Value) (Value, error) {
	if err := requireArity("IIF", args, 3); err != nil {
		return Value{}, err
	}
	cond, err := asLogical(args[0])
	if err != nil {
		return Value{}, fmt.Errorf("IIF: wrong type")
	}
	if cond {
		return args[1], nil
	}
	return args[2], nil
}

func builtinDISKSPACE(args []Value, env Env) (Value, error) {
	if err := requireArity("DISKSPACE", args, 0); err != nil {
		return Value{}, err
	}
	base := "."
	if env != nil {
		if p, ok := env.Default(); ok && p != "" {
			base = p
		}
	}
	var st os.FileInfo
	var err error
	st, err = os.Stat(base)
	if err != nil {
		return Value{}, fmt.Errorf("DISKSPACE: %w", err)
	}
	_ = st
	free, err := diskFreeBytes(base)
	if err != nil {
		return Value{}, fmt.Errorf("DISKSPACE: %w", err)
	}
	return NumberVal(float64(free)), nil
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

func dateOrder(env Env) string {
	if env != nil {
		if o, _, ok := env.DateSettings(); ok && o != "" {
			return o
		}
	}
	return DateAmerican
}

func dateCentury(env Env) bool {
	if env != nil {
		if _, c, ok := env.DateSettings(); ok {
			return c
		}
	}
	return false
}

func requireDate(fn string, v Value) (Value, error) {
	if v.Kind != KindDate {
		return Value{}, fmt.Errorf("%s: wrong type", fn)
	}
	return v, nil
}

func builtinDATE(args []Value, env Env) (Value, error) {
	if err := requireArity("DATE", args, 0); err != nil {
		return Value{}, err
	}
	if env != nil {
		if t, ok := env.Today(); ok {
			return DateVal(t), nil
		}
	}
	return DateVal(time.Now()), nil
}

func builtinCTOD(args []Value, env Env) (Value, error) {
	if err := requireArity("CTOD", args, 1); err != nil {
		return Value{}, err
	}
	return ParseDateString(args[0].String(), dateOrder(env)), nil
}

func builtinDTOC(args []Value, env Env) (Value, error) {
	if err := requireArity("DTOC", args, 1); err != nil {
		return Value{}, err
	}
	d, err := requireDate("DTOC", args[0])
	if err != nil {
		return Value{}, err
	}
	return StringVal(FormatDate(d, dateOrder(env), dateCentury(env))), nil
}

func builtinDAY(args []Value) (Value, error) {
	if err := requireArity("DAY", args, 1); err != nil {
		return Value{}, err
	}
	d, err := requireDate("DAY", args[0])
	if err != nil {
		return Value{}, err
	}
	if d.EmptyDate {
		return NumberVal(0), nil
	}
	return NumberVal(float64(d.Date.Day())), nil
}

func builtinMONTH(args []Value) (Value, error) {
	if err := requireArity("MONTH", args, 1); err != nil {
		return Value{}, err
	}
	d, err := requireDate("MONTH", args[0])
	if err != nil {
		return Value{}, err
	}
	if d.EmptyDate {
		return NumberVal(0), nil
	}
	return NumberVal(float64(d.Date.Month())), nil
}

func builtinYEAR(args []Value) (Value, error) {
	if err := requireArity("YEAR", args, 1); err != nil {
		return Value{}, err
	}
	d, err := requireDate("YEAR", args[0])
	if err != nil {
		return Value{}, err
	}
	if d.EmptyDate {
		return NumberVal(0), nil
	}
	return NumberVal(float64(d.Date.Year())), nil
}

func builtinCMONTH(args []Value) (Value, error) {
	if err := requireArity("CMONTH", args, 1); err != nil {
		return Value{}, err
	}
	d, err := requireDate("CMONTH", args[0])
	if err != nil {
		return Value{}, err
	}
	if d.EmptyDate {
		return StringVal(""), nil
	}
	return StringVal(monthNames[int(d.Date.Month())]), nil
}

func builtinCDOW(args []Value) (Value, error) {
	if err := requireArity("CDOW", args, 1); err != nil {
		return Value{}, err
	}
	d, err := requireDate("CDOW", args[0])
	if err != nil {
		return Value{}, err
	}
	if d.EmptyDate {
		return StringVal(""), nil
	}
	return StringVal(dowNames[d.Date.Weekday()]), nil
}

func builtinDOW(args []Value) (Value, error) {
	if err := requireArity("DOW", args, 1); err != nil {
		return Value{}, err
	}
	d, err := requireDate("DOW", args[0])
	if err != nil {
		return Value{}, err
	}
	if d.EmptyDate {
		return NumberVal(0), nil
	}
	// Sunday = 1 … Saturday = 7
	return NumberVal(float64(d.Date.Weekday()) + 1), nil
}

func builtinTIME(args []Value) (Value, error) {
	if err := requireArity("TIME", args, 0); err != nil {
		return Value{}, err
	}
	return StringVal(time.Now().Format("15:04:05")), nil
}

func builtinLUPDATE(args []Value, env Env) (Value, error) {
	if err := requireArity("LUPDATE", args, 0); err != nil {
		return Value{}, err
	}
	if env == nil {
		return EmptyDateVal(), nil
	}
	return env.LUpdate(), nil
}
