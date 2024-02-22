package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// RunInline executes semicolon-separated dBase commands and exits without starting the REPL.
func RunInline(ctx *context.Context, input string) error {
	for _, line := range splitInlineCommands(input) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		cmd := ParseCommand(line)
		if cmd.Verb == "QUIT" {
			return nil
		}

		if err := commandMux.Dispatch(ctx, cmd); err != nil {
			fmt.Fprintln(ctx.Stderr, err.Error())
			return err
		}
	}
	return nil
}

func splitInlineCommands(input string) []string {
	var parts []string
	var cur strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
			cur.WriteByte(ch)
		case ch == '"' && !inSingle:
			inDouble = !inDouble
			cur.WriteByte(ch)
		case ch == ';' && !inSingle && !inDouble:
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(ch)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}
