package expr

import (
	"fmt"
	"unicode"
)

// ExpandMacros replaces &name once with the character value of that memory variable.
// Names are letters and digits after &. Missing or non-string variables are errors.
func ExpandMacros(line string, lookup func(name string) (Value, bool)) (string, error) {
	if lookup == nil {
		return line, nil
	}
	var out []byte
	i := 0
	for i < len(line) {
		if line[i] != '&' {
			out = append(out, line[i])
			i++
			continue
		}
		// Skip quoted regions: if we are inside a string, leave & alone.
		// Substitution is whole-line before parse; still replace outside quotes.
		j := i + 1
		if j >= len(line) || !isMacroNameStart(line[j]) {
			out = append(out, '&')
			i++
			continue
		}
		for j < len(line) && isMacroNameCont(line[j]) {
			j++
		}
		name := line[i+1 : j]
		v, ok := lookup(name)
		if !ok {
			return "", fmt.Errorf("macro variable %s not found", name)
		}
		if v.Kind != KindString {
			return "", fmt.Errorf("macro variable %s is not a character value", name)
		}
		out = append(out, v.Str...)
		i = j
	}
	return string(out), nil
}

func isMacroNameStart(c byte) bool {
	return unicode.IsLetter(rune(c))
}

func isMacroNameCont(c byte) bool {
	return unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c))
}
