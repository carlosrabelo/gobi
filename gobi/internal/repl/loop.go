package repl

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

const maxLine = 254

// Run writes the dot prompt, reads lines, and dispatches verbs until QUIT or EOF.
func Run(ctx *context.Context, mux *Mux) error {
	for {
		fmt.Fprint(ctx.Stdout, ". ")
		line, err := ctx.StdinReader().ReadString('\n')
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				return nil
			}
		} else if err != nil {
			return err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}
		if len(line) > maxLine {
			fmt.Fprintln(ctx.Stderr, "*** Line too long")
			continue
		}
		cmd := command.Split(line)
		err = mux.Dispatch(ctx, cmd)
		if errors.Is(err, ErrQuit) {
			return nil
		}
		if err != nil {
			continue
		}
	}
}
