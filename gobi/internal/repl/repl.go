package repl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

var errQuit = errors.New("quit")

var commandMux *CommandMux

const prompt = ". "

// Run starts the interactive REPL loop, reading commands from stdin.
func Run(ctx *context.Context) error {
	hist := NewHistory(maxHistorySize)
	hist.Load()
	defer hist.Save()

	syncScreenSize(ctx)
	if err := presentClearScreen(ctx); err != nil {
		return err
	}

	// Try using raw terminal input for history navigation.
	stdFile, isFile := ctx.Stdin.(*os.File)
	if isFile && term.IsTerminal(stdFile) {
		err := runTerminal(ctx, stdFile, hist)
		if err == errQuit {
			return nil
		}
		return err
	}

	err := runBuffered(ctx, hist)
	if err == errQuit {
		return nil
	}
	return err
}

func runTerminal(ctx *context.Context, in *os.File, hist *History) error {
	tr := newTerminalReader(in, ctx.Stdout, prompt)

	for {
		syncScreenSize(ctx)
		line, err := tr.readLine(hist)
		if err != nil {
			if err == io.EOF {
				fmt.Fprintln(ctx.Stdout)
				return nil
			}
			return fmt.Errorf("read error: %w", err)
		}

		line = strings.TrimSpace(line)
		if err := processLine(ctx, line, hist); err != nil {
			return err
		}
	}
}

func runBuffered(ctx *context.Context, hist *History) error {
	reader := ctx.StdinReader()

	for {
		syncScreenSize(ctx)
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
		if err := processLine(ctx, line, hist); err != nil {
			return err
		}
	}
}

// processLine parses and processes a single command line.
// Returns errQuit when the QUIT command is entered.
func processLine(ctx *context.Context, line string, hist *History) error {
	if line == "" {
		return nil
	}

	hist.Add(line)

	cmd := ParseCommand(line)
	if cmd.Verb == "QUIT" {
		return errQuit
	}

	err := commandMux.Dispatch(ctx, cmd)
	if err != nil {
		fmt.Fprintln(ctx.Stderr, err.Error())
	}
	return nil
}
