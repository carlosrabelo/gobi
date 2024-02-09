package repl

import (
	"fmt"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

// RunCLI starts a session from an -e command string and an optional program file.
// Empty commands and an empty file start the interactive prompt.
func RunCLI(commands, file string) int {
	ctx := context.New()
	mux := NewMux()

	if commands != "" {
		return runCommands(ctx, mux, commands)
	}
	if file != "" {
		err := mux.Dispatch(ctx, command.Split("DO "+file))
		if err != nil {
			return 1
		}
		return 0
	}
	home, _ := os.UserHomeDir()
	hist := NewHistory(home)
	hist.SetCap(ctx.Config.HistoryTo)
	_ = hist.Load()
	if err := RunWithHistory(ctx, mux, hist); err != nil {
		fmt.Fprintln(os.Stderr, "***", err.Error())
		return 1
	}
	return 0
}

func runCommands(ctx *context.Context, mux *Mux, src string) int {
	parts := splitSemi(src)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		cmd := command.Split(p)
		err := mux.Dispatch(ctx, cmd)
		if err == ErrQuit {
			return 0
		}
		if err != nil {
			return 1
		}
	}
	return 0
}

// splitSemi splits on semicolons that are outside quotes.
func splitSemi(s string) []string {
	var parts []string
	var b strings.Builder
	inSingle, inDouble := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
			b.WriteByte(c)
		case c == '"' && !inSingle:
			inDouble = !inDouble
			b.WriteByte(c)
		case c == ';' && !inSingle && !inDouble:
			parts = append(parts, b.String())
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	parts = append(parts, b.String())
	return parts
}
