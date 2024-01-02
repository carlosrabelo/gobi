package repl

import (
	"fmt"
	"io"
	"unicode"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// confirmSafety prints question and returns true only when the next line starts with Y/y.
// Any other line, including EOF, cancels with a *** line.
func confirmSafety(ctx *context.Context, question string) (bool, error) {
	fmt.Fprintln(ctx.Out(), question)
	line, err := readInteractiveLine(ctx)
	if err != nil && err != io.EOF {
		return false, err
	}
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
	fmt.Fprint(ctx.Out(), prompt)
	line, err := readInteractiveLine(ctx)
	if err != nil && err != io.EOF {
		return "", err
	}
	return line, nil
}
