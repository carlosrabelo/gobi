package expr

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Kind is the runtime type of a Value.
type Kind int

const (
	KindString Kind = iota
	KindNumber
	KindLogical
	KindDate
	KindMemo
)

// Value is a string, number, logical, date, or memo result.
type Value struct {
	Kind Kind
	Str  string
	Num  float64
	Log  bool
	// Date is a calendar day when Kind is KindDate. Empty is EmptyDate.
	Date time.Time
	// EmptyDate is true for an empty date (not a calendar day).
	EmptyDate bool
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

// DateVal builds a date value for the given calendar day (local, truncated).
func DateVal(t time.Time) Value {
	y, m, d := t.Date()
	return Value{Kind: KindDate, Date: time.Date(y, m, d, 0, 0, 0, 0, time.Local)}
}

// EmptyDateVal builds an empty date.
func EmptyDateVal() Value {
	return Value{Kind: KindDate, EmptyDate: true}
}

// MemoVal builds a memo text value.
func MemoVal(s string) Value {
	return Value{Kind: KindMemo, Str: s}
}

// YYYYMMDD returns eight digits for a date, or eight spaces when empty.
func (v Value) YYYYMMDD() string {
	if v.Kind != KindDate || v.EmptyDate {
		return "        "
	}
	return v.Date.Format("20060102")
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
	case KindDate:
		return FormatDate(v, DateAmerican, false)
	case KindMemo:
		return v.Str
	default:
		return v.Str
	}
}

func compareDates(a, b Value) int {
	if a.EmptyDate && b.EmptyDate {
		return 0
	}
	if a.EmptyDate {
		return -1
	}
	if b.EmptyDate {
		return 1
	}
	as := a.Date.Unix()
	bs := b.Date.Unix()
	switch {
	case as < bs:
		return -1
	case as > bs:
		return 1
	default:
		return 0
	}
}

// ParseYYYYMMDD parses eight digits into a date. Spaces mean empty.
func ParseYYYYMMDD(s string) (Value, error) {
	if len(s) < 8 {
		s = s + strings.Repeat(" ", 8-len(s))
	} else {
		s = s[:8]
	}
	if strings.TrimSpace(s) == "" {
		return EmptyDateVal(), nil
	}
	t, err := time.ParseInLocation("20060102", s, time.Local)
	if err != nil {
		return EmptyDateVal(), fmt.Errorf("bad date %q", s)
	}
	return DateVal(t), nil
}
