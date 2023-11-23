package expr

import (
	"fmt"
	"strings"
	"time"
)

// Date order names from SET DATE.
const (
	DateAmerican = "AMERICAN"
	DateANSI     = "ANSI"
	DateBritish  = "BRITISH"
	DateItalian  = "ITALIAN"
	DateFrench   = "FRENCH"
	DateGerman   = "GERMAN"
)

// FormatDate renders a date using order and century.
func FormatDate(v Value, order string, century bool) string {
	if v.Kind != KindDate || v.EmptyDate {
		if century {
			return "  /  /    "
		}
		return "  /  /  "
	}
	y := v.Date.Year() % 100
	y4 := v.Date.Year()
	m := int(v.Date.Month())
	d := v.Date.Day()
	ys := fmt.Sprintf("%02d", y)
	if century {
		ys = fmt.Sprintf("%04d", y4)
	}
	ms := fmt.Sprintf("%02d", m)
	ds := fmt.Sprintf("%02d", d)
	switch strings.ToUpper(order) {
	case DateANSI:
		return ys + "." + ms + "." + ds
	case DateBritish, DateFrench:
		return ds + "/" + ms + "/" + ys
	case DateItalian:
		return ds + "-" + ms + "-" + ys
	case DateGerman:
		return ds + "." + ms + "." + ys
	default: // AMERICAN
		return ms + "/" + ds + "/" + ys
	}
}

// ParseDateString parses s in the given SET DATE order. Invalid → empty date.
func ParseDateString(s string, order string) Value {
	s = strings.TrimSpace(s)
	if s == "" {
		return EmptyDateVal()
	}
	order = strings.ToUpper(order)
	var layout string
	switch order {
	case DateANSI:
		layout = "06.01.02"
		if len(s) >= 10 {
			layout = "2006.01.02"
		}
	case DateBritish, DateFrench:
		layout = "02/01/06"
		if looksCentury(s, '/') {
			layout = "02/01/2006"
		}
	case DateItalian:
		layout = "02-01-06"
		if looksCentury(s, '-') {
			layout = "02-01-2006"
		}
	case DateGerman:
		layout = "02.01.06"
		if looksCentury(s, '.') {
			layout = "02.01.2006"
		}
	default:
		layout = "01/02/06"
		if looksCentury(s, '/') {
			layout = "01/02/2006"
		}
	}
	t, err := time.ParseInLocation(layout, s, time.Local)
	if err != nil {
		return EmptyDateVal()
	}
	return DateVal(t)
}

func looksCentury(s string, sep byte) bool {
	parts := strings.Split(s, string(sep))
	if len(parts) != 3 {
		return false
	}
	return len(parts[2]) == 4 || (sep == '.' && len(parts[0]) == 4) // ANSI YY.MM.DD vs YYYY
}

var monthNames = []string{
	"", "January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

var dowNames = []string{
	"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday",
}

func dateAdd(left, right Value) (Value, error) {
	if left.Kind == KindDate && right.Kind == KindNumber {
		if left.EmptyDate {
			return Value{}, fmt.Errorf("empty date in arithmetic")
		}
		return DateVal(left.Date.AddDate(0, 0, int(right.Num))), nil
	}
	if right.Kind == KindDate && left.Kind == KindNumber {
		if right.EmptyDate {
			return Value{}, fmt.Errorf("empty date in arithmetic")
		}
		return DateVal(right.Date.AddDate(0, 0, int(left.Num))), nil
	}
	return Value{}, fmt.Errorf("type mismatch")
}

func dateSub(left, right Value) (Value, error) {
	if left.Kind == KindDate && right.Kind == KindDate {
		if left.EmptyDate || right.EmptyDate {
			return Value{}, fmt.Errorf("empty date in arithmetic")
		}
		days := left.Date.Sub(right.Date).Hours() / 24
		return NumberVal(days), nil
	}
	if left.Kind == KindDate && right.Kind == KindNumber {
		if left.EmptyDate {
			return Value{}, fmt.Errorf("empty date in arithmetic")
		}
		return DateVal(left.Date.AddDate(0, 0, -int(right.Num))), nil
	}
	return Value{}, fmt.Errorf("type mismatch")
}
