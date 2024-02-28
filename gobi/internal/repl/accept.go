package repl

import (
	"fmt"
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/internal/symbols"
)

// handleAccept implements ACCEPT ['prompt'] TO <memvar>, reading a line of
// text from the console and storing it as a character memory variable.
func handleAccept(ctx *context.Context, cmd Command) error {
	target := strings.ToUpper(strings.TrimSpace(cmd.ToClause))
	if target == "" {
		return fmt.Errorf("*** ACCEPT requires TO <variable>")
	}
	if err := symbols.ValidateName(target); err != nil {
		return fmt.Errorf("*** Invalid ACCEPT target %s: %v", target, err)
	}

	prompt, err := parseConsolePrompt("ACCEPT", cmd.Args)
	if err != nil {
		return err
	}

	line, err := readAppendLine(ctx, ctx.StdinReader(), consolePromptText(prompt))
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return fmt.Errorf("*** Error reading ACCEPT input: %w", err)
	}

	if err := ctx.Variables.Set(target, line); err != nil {
		return fmt.Errorf("*** Error storing %s: %v", target, err)
	}
	return nil
}

// parseConsolePrompt validates and unquotes the optional prompt string of
// console input commands (ACCEPT, INPUT).
func parseConsolePrompt(verb, args string) (string, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", nil
	}
	if len(args) >= 2 {
		switch args[0] {
		case '\'':
			if args[len(args)-1] == '\'' {
				return args[1 : len(args)-1], nil
			}
		case '"':
			if args[len(args)-1] == '"' {
				return args[1 : len(args)-1], nil
			}
		}
	}
	return "", fmt.Errorf("*** %s prompt must be a quoted string", verb)
}

// consolePromptText renders the dBase II console input marker: the prompt
// text followed by ": ", or a bare ": " when no prompt was given.
func consolePromptText(prompt string) string {
	if prompt == "" {
		return ": "
	}
	return prompt + ": "
}
