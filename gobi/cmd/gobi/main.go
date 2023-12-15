package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/internal/repl"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("gobi", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	eFlag := fs.String("e", "", "commands separated by ;")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	rest := fs.Args()
	if *eFlag != "" && len(rest) > 0 {
		fmt.Fprintln(os.Stderr, "*** -e cannot be combined with a file")
		return 1
	}
	if len(rest) > 1 {
		fmt.Fprintln(os.Stderr, "*** too many file arguments")
		return 1
	}

	ctx := context.New()
	mux := repl.NewMux()

	if *eFlag != "" {
		return runE(ctx, mux, *eFlag)
	}
	if len(rest) == 1 {
		err := mux.Dispatch(ctx, command.Split("DO "+rest[0]))
		if err != nil {
			return 1
		}
		return 0
	}
	home, _ := os.UserHomeDir()
	hist := repl.NewHistory(home)
	hist.SetCap(ctx.Config.HistoryTo)
	_ = hist.Load()
	if err := repl.RunWithHistory(ctx, mux, hist); err != nil {
		fmt.Fprintln(os.Stderr, "***", err.Error())
		return 1
	}
	return 0
}

func runE(ctx *context.Context, mux *repl.Mux, src string) int {
	parts := splitSemi(src)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		cmd := command.Split(p)
		err := mux.Dispatch(ctx, cmd)
		if err == repl.ErrQuit {
			return 0
		}
		if err != nil {
			return 1
		}
	}
	return 0
}

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
