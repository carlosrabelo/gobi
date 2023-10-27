package repl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// recordScope is a parsed dBase II scope clause on a record command.
type recordScope struct {
	all      bool // ALL: every record from the top of the file
	next     int  // NEXT <n>: n records starting at the current one; 0 when absent
	explicit bool // a scope clause was present in the command
}

// parseScopeClause strips a leading ALL or NEXT <n> scope from a command's
// argument text, returning the scope and the remaining arguments verbatim.
func parseScopeClause(args string) (recordScope, string, error) {
	word, rest := splitLeadingWord(args)
	switch strings.ToUpper(word) {
	case "ALL":
		return recordScope{all: true, explicit: true}, rest, nil
	case "NEXT":
		numWord, tail := splitLeadingWord(rest)
		n, err := strconv.Atoi(numWord)
		if err != nil || n < 1 {
			return recordScope{}, "", fmt.Errorf("*** NEXT requires a positive record count")
		}
		return recordScope{next: n, explicit: true}, tail, nil
	}
	return recordScope{}, strings.TrimSpace(args), nil
}

// splitLeadingWord returns the first whitespace-delimited word of s and the
// remaining text with leading whitespace removed.
func splitLeadingWord(s string) (string, string) {
	s = strings.TrimLeft(s, " \t")
	idx := strings.IndexAny(s, " \t")
	if idx < 0 {
		return s, ""
	}
	return s[:idx], strings.TrimLeft(s[idx:], " \t")
}

// scanRange computes the first record index and the maximum number of records
// to traverse for a record command (limit 0 means unlimited). When no scope
// is given, commands with singleDefault target only the current record unless
// FOR or WHILE clauses widen the scan, matching dBase II defaults.
func scanRange(area *context.WorkArea, scope recordScope, hasFor, hasWhile, singleDefault bool) (start, limit int) {
	switch {
	case scope.next > 0:
		return area.RecordNo, scope.next
	case scope.all:
		if hasWhile {
			return area.RecordNo, 0
		}
		return 0, 0
	case singleDefault && !hasFor && !hasWhile:
		return area.RecordNo, 1
	case hasWhile:
		return area.RecordNo, 0
	default:
		return 0, 0
	}
}
