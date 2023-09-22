package repl

import (
	"errors"
	"fmt"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func fail(ctx *context.Context, format string, args ...any) error {
	msg := "*** " + fmt.Sprintf(format, args...)
	fmt.Fprintln(ctx.Stderr, msg)
	return errors.New(msg)
}
