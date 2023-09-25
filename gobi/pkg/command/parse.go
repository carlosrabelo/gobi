// Package command parses dBase II command lines.
package command

import (
	"strings"
	"unicode"
)

// Command represents a parsed dBase II command line.
type Command struct {
	Verb        string
	RawVerb     string
	Args        string
	ForClause   string
	WhileClause string
	ToClause    string
	FromClause  string
}

// Parse splits line into verb and clauses.
func Parse(line string) Command {
	line = strings.TrimSpace(line)
	if line == "" {
		return Command{}
	}

	verb, rest := splitFirstWord(line)
	cmd := Command{
		Verb:    strings.ToUpper(verb),
		RawVerb: verb,
	}
	if rest == "" {
		return cmd
	}

	tokens := Tokenize(rest)
	cmd.ForClause, cmd.Args = extractForClause(tokens)
	return cmd
}

func splitFirstWord(s string) (string, string) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "??") {
		return "??", strings.TrimLeft(s[2:], " ")
	}
	if strings.HasPrefix(s, "?") {
		return "?", strings.TrimLeft(s[1:], " ")
	}

	idx := strings.IndexFunc(s, unicode.IsSpace)
	if idx < 0 {
		return s, ""
	}
	return s[:idx], strings.TrimLeft(s[idx:], " ")
}

// Tokenize splits s into whitespace-delimited tokens, preserving quoted strings.
func Tokenize(s string) []string {
	var tokens []string
	var cur strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
			cur.WriteByte(ch)
		case ch == '"' && !inSingle:
			inDouble = !inDouble
			cur.WriteByte(ch)
		case unicode.IsSpace(rune(ch)) && !inSingle && !inDouble:
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(ch)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func extractForClause(tokens []string) (forClause, args string) {
	var result []string
	i := 0
	for i < len(tokens) {
		if strings.EqualFold(tokens[i], "FOR") {
			forClause = JoinTokens(tokens[i+1:])
			break
		}
		result = append(result, tokens[i])
		i++
	}
	args = JoinTokens(result)
	return
}

// JoinTokens rebuilds a command argument string from tokens.
func JoinTokens(tokens []string) string {
	switch len(tokens) {
	case 0:
		return ""
	case 1:
		return tokens[0]
	default:
		return strings.Join(tokens, " ")
	}
}
