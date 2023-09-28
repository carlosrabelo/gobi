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
// The line is case-insensitive; the Verb field is always upper-cased.
// White-space is collapsed, and quoted strings (' or ") remain intact.
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

	// REMARK echoes the rest of the line verbatim, so its text must not be
	// tokenized or scanned for FOR/WHILE/TO/FROM clause keywords.
	if cmd.Verb == "REMARK" {
		cmd.Args = rest
		return cmd
	}

	tokens := Tokenize(rest)
	cmd.ForClause, cmd.WhileClause, cmd.ToClause, cmd.FromClause, cmd.Args = extractClauses(tokens)

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

func tokenize(s string) []string {
	return Tokenize(s)
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

func extractClauses(tokens []string) (forClause, whileClause, toClause, fromClause, args string) {
	var result []string
	i := 0

	for i < len(tokens) {
		tok := tokens[i]
		upper := strings.ToUpper(tok)

		switch upper {
		case "FOR":
			end := nextClauseIndex(tokens, i+1)
			forClause = JoinTokens(tokens[i+1 : end])
			i = end

		case "WHILE":
			end := nextClauseIndex(tokens, i+1)
			whileClause = JoinTokens(tokens[i+1 : end])
			i = end

		case "TO":
			end := nextClauseIndex(tokens, i+1)
			toClause = JoinTokens(tokens[i+1 : end])
			i = end

		case "FROM":
			if i+1 < len(tokens) && strings.ToUpper(tokens[i+1]) != "ON" {
				fromClause = tokens[i+1]
				i += 2
			} else {
				i++
			}

		default:
			result = append(result, tok)
			i++
		}
	}

	args = JoinTokens(result)
	return
}

func nextClauseIndex(tokens []string, start int) int {
	for i := start; i < len(tokens); i++ {
		upper := strings.ToUpper(tokens[i])
		if upper == "FOR" || upper == "WHILE" || upper == "TO" || upper == "FROM" || upper == "FIELD" {
			return i
		}
	}
	return len(tokens)
}

func joinTokens(tokens []string) string {
	return JoinTokens(tokens)
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
