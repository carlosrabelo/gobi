package repl

import (
	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func handleQuit(ctx *context.Context, cmd Command) error {
	return errQuit
}
