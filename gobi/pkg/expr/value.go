package expr

import "strconv"

// Kind is the runtime type of a Value.
type Kind int

const (
	KindString Kind = iota
	KindNumber
	KindLogical
)

// Value is a string, number, or logical result.
type Value struct {
	Kind Kind
	Str  string
	Num  float64
	Log  bool
}

// StringVal builds a string value.
func StringVal(s string) Value {
	return Value{Kind: KindString, Str: s}
}

// NumberVal builds a number value.
func NumberVal(n float64) Value {
	return Value{Kind: KindNumber, Num: n}
}

// LogicalVal builds a logical value.
func LogicalVal(b bool) Value {
	return Value{Kind: KindLogical, Log: b}
}

// String formats the value for display.
func (v Value) String() string {
	switch v.Kind {
	case KindLogical:
		if v.Log {
			return ".T."
		}
		return ".F."
	case KindNumber:
		return strconv.FormatFloat(v.Num, 'g', -1, 64)
	default:
		return v.Str
	}
}
