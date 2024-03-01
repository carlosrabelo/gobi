package repl

import (
	"fmt"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// handleTextInteractive rejects TEXT at the dot prompt. The block construct
// only exists inside command files, where the script reader captures the
// literal lines up to the matching ENDTEXT.
func handleTextInteractive(ctx *context.Context, cmd Command) error {
	return fmt.Errorf("*** TEXT is only valid in command files")
}

// handleEndTextInteractive reports a stray ENDTEXT, either typed at the dot
// prompt or left in a script without an opening TEXT.
func handleEndTextInteractive(ctx *context.Context, cmd Command) error {
	return fmt.Errorf("*** ENDTEXT without TEXT")
}
