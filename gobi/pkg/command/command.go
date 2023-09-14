package command

import "strings"

// Command is one split command line.
type Command struct {
	Verb    string // upper-case verb
	VerbRaw string // original spelling
	Args    string // positional argument text
	Clauses map[string]string
}

// Split parses one command line into a verb and clauses.
func Split(line string) Command {
	line = stripAmpComment(line)
	line = strings.TrimSpace(line)
	cmd := Command{Clauses: map[string]string{}}
	if line == "" {
		return cmd
	}

	verbRaw, rest := takeVerb(line)
	cmd.VerbRaw = verbRaw
	cmd.Verb = strings.ToUpper(verbRaw)

	if cmd.Verb == "REMARK" {
		cmd.Args = strings.TrimSpace(rest)
		return cmd
	}

	tokens := tokenize(rest)
	var args []string
	i := 0
	for i < len(tokens) {
		key := strings.ToUpper(tokens[i])
		switch key {
		case "FOR", "WHILE", "TO":
			i++
			var parts []string
			for i < len(tokens) {
				nk := strings.ToUpper(tokens[i])
				if nk == "FOR" || nk == "WHILE" || nk == "TO" || nk == "FROM" || nk == "FIELD" {
					break
				}
				parts = append(parts, tokens[i])
				i++
			}
			cmd.Clauses[key] = strings.Join(parts, " ")
		case "FROM":
			i++
			if i < len(tokens) && strings.ToUpper(tokens[i]) != "ON" {
				cmd.Clauses["FROM"] = tokens[i]
				i++
			} else {
				args = append(args, tokens[i-1])
			}
		default:
			args = append(args, tokens[i])
			i++
		}
	}
	cmd.Args = strings.Join(args, " ")
	return cmd
}

func takeVerb(line string) (verb, rest string) {
	if strings.HasPrefix(line, "??") {
		return "??", strings.TrimSpace(line[2:])
	}
	if strings.HasPrefix(line, "?") {
		return "?", strings.TrimSpace(line[1:])
	}
	tokens := tokenize(line)
	if len(tokens) == 0 {
		return "", ""
	}
	verb = tokens[0]
	// Re-join the remainder from the original line after the first token.
	rest = strings.TrimSpace(line[len(verb):])
	return verb, rest
}

func stripAmpComment(s string) string {
	inSingle, inDouble := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case !inSingle && !inDouble && c == '&' && i+1 < len(s) && s[i+1] == '&':
			return s[:i]
		}
	}
	return s
}

func tokenize(s string) []string {
	var tokens []string
	s = strings.TrimSpace(s)
	for len(s) > 0 {
		s = strings.TrimLeft(s, " \t")
		if s == "" {
			break
		}
		if s[0] == '"' || s[0] == '\'' {
			q := s[0]
			j := 1
			for j < len(s) && s[j] != q {
				j++
			}
			if j < len(s) {
				j++ // include closing quote
			}
			tokens = append(tokens, s[:j])
			s = s[j:]
			continue
		}
		j := 0
		for j < len(s) && s[j] != ' ' && s[j] != '\t' && s[j] != '"' && s[j] != '\'' {
			j++
		}
		tokens = append(tokens, s[:j])
		s = s[j:]
	}
	return tokens
}
