package repl

import "github.com/carlosrabelo/gobi/gobi/pkg/command"

// Command represents a parsed dBase II command line.
type Command = command.Command

// ParseCommand splits a dBase II command line into verb and clauses.
func ParseCommand(line string) Command {
	return command.Parse(line)
}

func tokenize(s string) []string {
	return command.Tokenize(s)
}

func joinTokens(tokens []string) string {
	return command.JoinTokens(tokens)
}
