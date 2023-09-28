package repl

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// confirmSafety prints question and returns true only when the next line starts with Y/y.
// Any other line, including EOF, cancels with a *** line.
func confirmSafety(ctx *context.Context, question string) (bool, error) {
	fmt.Fprintln(ctx.Stdout, question)
	line, err := ctx.StdinReader().ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	line = strings.TrimRight(line, "\r\n")
	for _, r := range line {
		if unicode.IsSpace(r) {
			continue
		}
		if r == 'Y' || r == 'y' {
			return true, nil
		}
		break
	}
	return false, fail(ctx, "cancelled")
}

func readPromptLine(ctx *context.Context, prompt string) (string, error) {
	fmt.Fprint(ctx.Stdout, prompt)
	line, err := ctx.StdinReader().ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
