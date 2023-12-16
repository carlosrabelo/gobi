package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func handleDo(ctx *context.Context, cmd Command) error {
	args := strings.TrimSpace(cmd.Args)
	if args == "" {
		return fmt.Errorf("*** DO requires a command file name")
	}

	upper := strings.ToUpper(args)
	switch {
	case upper == "WHILE" || strings.HasPrefix(upper, "WHILE "):
		return fmt.Errorf("*** DO WHILE is only valid in command files")
	case upper == "CASE" || strings.HasPrefix(upper, "CASE "):
		return fmt.Errorf("*** DO CASE is only valid in command files")
	}

	return RunScript(ctx, args)
}

// handleCaseInteractive rejects a CASE branch typed at the dot prompt or left
// in a script outside a DO CASE structure.
func handleCaseInteractive(ctx *context.Context, cmd Command) error {
	return fmt.Errorf("*** CASE without matching DO CASE")
}

// handleOtherwiseInteractive rejects a stray OTHERWISE branch.
func handleOtherwiseInteractive(ctx *context.Context, cmd Command) error {
	return fmt.Errorf("*** OTHERWISE without matching DO CASE")
}

// handleEndCaseInteractive rejects a stray ENDCASE.
func handleEndCaseInteractive(ctx *context.Context, cmd Command) error {
	return fmt.Errorf("*** ENDCASE without matching DO CASE")
}
