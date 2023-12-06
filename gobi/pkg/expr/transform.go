package expr

import (
	"fmt"
	"math"
	"strings"
)

// Transform formats v with a PICTURE string (034).
func Transform(v Value, picture string) (string, error) {
	pic := picture
	funcs := ""
	template := pic
	if strings.HasPrefix(pic, "@") {
		rest := pic[1:]
		if i := strings.IndexByte(rest, ' '); i >= 0 {
			funcs = strings.ToUpper(rest[:i])
			template = rest[i+1:]
		} else {
			funcs = strings.ToUpper(rest)
			template = ""
		}
	}
	if v.Kind == KindDate {
		order := DateAmerican
		if strings.Contains(funcs, "E") {
			order = DateBritish
		}
		return FormatDate(v, order, false), nil
	}
	if v.Kind == KindNumber || (template != "" && looksNumericTemplate(template)) {
		return transformNumber(v, funcs, template), nil
	}
	return transformString(v.String(), funcs, template), nil
}

func looksNumericTemplate(t string) bool {
	for _, c := range t {
		if c == '9' || c == '#' || c == '.' || c == ',' || c == '$' || c == '*' {
			return true
		}
	}
	return false
}

func transformNumber(v Value, funcs, template string) string {
	n := v.Num
	if v.Kind != KindNumber {
		n, _ = asNumber(v)
	}
	neg := n < 0
	if neg {
		n = -n
	}
	if strings.Contains(funcs, "Z") && n == 0 {
		return strings.Repeat(" ", len(template))
	}
	dec := 0
	if i := strings.IndexByte(template, '.'); i >= 0 {
		dec = strings.Count(template[i+1:], "9") + strings.Count(template[i+1:], "#")
	}
	pow := math.Pow(10, float64(dec))
	n = math.Round(n*pow) / pow
	s := fmt.Sprintf("%.*f", dec, n)
	if template != "" {
		s = applyNumericTemplate(s, template)
	}
	if strings.Contains(funcs, "B") {
		s = strings.TrimLeft(s, " ")
	}
	if neg {
		if strings.Contains(funcs, "(") {
			s = "(" + strings.TrimSpace(s) + ")"
		} else {
			s = "-" + strings.TrimSpace(s)
		}
	}
	if strings.Contains(funcs, "C") && !neg {
		s += " CR"
	}
	if strings.Contains(funcs, "X") && neg {
		s = strings.TrimPrefix(s, "-") + " DB"
	}
	return s
}

func applyNumericTemplate(num, template string) string {
	// simple: build from right, inserting commas
	parts := strings.SplitN(num, ".", 2)
	intPart := parts[0]
	decPart := ""
	if len(parts) == 2 {
		decPart = parts[1]
	}
	var out []byte
	ti := len(template) - 1
	if di := strings.LastIndexByte(template, '.'); di >= 0 {
		// decimals
		decT := template[di+1:]
		for len(decPart) < len(decT) {
			decPart += "0"
		}
		if len(decPart) > len(decT) {
			decPart = decPart[:len(decT)]
		}
		template = template[:di]
		ti = len(template) - 1
		out = append([]byte("."+decPart), out...)
	}
	ii := len(intPart) - 1
	for ti >= 0 {
		c := template[ti]
		ti--
		switch c {
		case '9', '#', '$', '*':
			if ii >= 0 {
				out = append([]byte{intPart[ii]}, out...)
				ii--
			} else if c == '*' {
				out = append([]byte{'*'}, out...)
			} else if c == '$' {
				out = append([]byte{'$'}, out...)
			} else {
				out = append([]byte{' '}, out...)
			}
		case ',':
			if ii >= 0 {
				out = append([]byte{','}, out...)
			}
		default:
			out = append([]byte{c}, out...)
		}
	}
	for ii >= 0 {
		out = append([]byte{intPart[ii]}, out...)
		ii--
	}
	return string(out)
}

func transformString(s, funcs, template string) string {
	if strings.Contains(funcs, "!") {
		s = strings.ToUpper(s)
	}
	if template == "" {
		return s
	}
	var out strings.Builder
	si := 0
	for i := 0; i < len(template); i++ {
		c := template[i]
		switch c {
		case '9', '#', 'A', 'N', 'X', 'Y', '!', 'L':
			if si < len(s) {
				ch := s[si]
				si++
				if c == '!' {
					ch = byte(strings.ToUpper(string(ch))[0])
				}
				out.WriteByte(ch)
			}
		default:
			out.WriteByte(c)
		}
	}
	return out.String()
}

func builtinTRANSFORM(args []Value, env Env) (Value, error) {
	if err := requireArity("TRANSFORM", args, 2); err != nil {
		return Value{}, err
	}
	s, err := Transform(args[0], args[1].String())
	if err != nil {
		return Value{}, err
	}
	return StringVal(s), nil
}
