package repl

import (
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

// Scope describes ALL / NEXT n / RECORD n.
type Scope struct {
	Kind string // "", "ALL", "NEXT", "RECORD"
	N    int
	Off  bool // LIST/DISPLAY OFF
	Rest string
}

func parseScope(cmd command.Command) Scope {
	toks := strings.Fields(cmd.Args)
	var s Scope
	i := 0
	for i < len(toks) {
		u := strings.ToUpper(toks[i])
		switch u {
		case "ALL":
			s.Kind = "ALL"
			i++
		case "NEXT":
			s.Kind = "NEXT"
			i++
			if i < len(toks) {
				s.N, _ = strconv.Atoi(toks[i])
				i++
			}
		case "RECORD":
			s.Kind = "RECORD"
			i++
			if i < len(toks) {
				s.N, _ = strconv.Atoi(toks[i])
				i++
			}
		case "OFF":
			s.Off = true
			i++
		default:
			s.Rest = strings.Join(toks[i:], " ")
			return s
		}
	}
	return s
}

func scopeRange(count, cursor int, s Scope) (start, end int) {
	switch s.Kind {
	case "ALL":
		return 1, count
	case "NEXT":
		start = cursor
		if start < 1 {
			start = 1
		}
		end = start + s.N - 1
		if end > count {
			end = count
		}
		return start, end
	case "RECORD":
		if s.N < 1 || s.N > count {
			return 1, 0 // empty
		}
		return s.N, s.N
	default:
		return 0, 0 // caller uses its default
	}
}
