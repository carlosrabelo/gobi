package repl

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

var errQuit = errors.New("quit")

const prompt = ". "

// Run starts the interactive REPL loop, reading commands from stdin.
func Run(ctx *context.Context) error {
	err := runBuffered(ctx)
	if err == errQuit {
		return nil
	}
	return err
}

func runBuffered(ctx *context.Context) error {
	reader := ctx.StdinReader()

	for {
		fmt.Fprint(ctx.Stdout, prompt)

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Fprintln(ctx.Stdout)
				return nil
			}
			return fmt.Errorf("read error: %w", err)
		}

		line = strings.TrimRight(line, "\r\n")
		if err := processLine(ctx, line); err != nil {
			return err
		}
	}
}

func processLine(ctx *context.Context, line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	if strings.EqualFold(line, "QUIT") {
		return errQuit
	}
	fmt.Fprintf(ctx.Stderr, "*** Unrecognized command verb\n")
	return nil
}
