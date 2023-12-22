package repl

import (
	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func handleCancel(ctx *context.Context, cmd Command) error {
	clearScriptExecution(ctx)
	return nil
}
