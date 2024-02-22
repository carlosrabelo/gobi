package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/internal/repl"
)

func main() {
	inlineCmd := flag.String("e", "", "Execute inline commands and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gobi [script.prg] [-e \"commands\"]\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	ctx := context.New()

	if err := run(ctx, flag.Args(), *inlineCmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx *context.Context, args []string, inlineCmd string) error {
	if inlineCmd != "" && len(args) > 0 {
		return fmt.Errorf("cannot use -e with a script file")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many script files")
	}

	if inlineCmd != "" {
		return repl.RunInline(ctx, inlineCmd)
	}

	if len(args) == 1 {
		return repl.RunScript(ctx, args[0])
	}

	return repl.Run(ctx)
}
