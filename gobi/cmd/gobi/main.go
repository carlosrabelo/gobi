package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/carlosrabelo/gobi/gobi/internal/repl"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("gobi", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	commands := fs.String("e", "", "commands separated by ;")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	rest := fs.Args()
	if *commands != "" && len(rest) > 0 {
		fmt.Fprintln(os.Stderr, "*** -e cannot be combined with a file")
		return 1
	}
	if len(rest) > 1 {
		fmt.Fprintln(os.Stderr, "*** too many file arguments")
		return 1
	}
	file := ""
	if len(rest) == 1 {
		file = rest[0]
	}
	return repl.RunCLI(*commands, file)
}
