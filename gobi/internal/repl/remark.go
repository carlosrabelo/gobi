package repl

import (
	"fmt"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// handleRemark echoes the REMARK text verbatim to the output, matching the
// dBase II behavior where REMARK is a display command rather than a comment.
func handleRemark(ctx *context.Context, cmd Command) error {
	fmt.Fprintf(ctx.Stdout, "%s\r\n", cmd.Args)
	return nil
}

// handleNote silently ignores a NOTE comment typed at the dot prompt. Inside
// command files the script reader filters NOTE lines out as remarks, so this
// handler only serves interactive sessions.
func handleNote(ctx *context.Context, cmd Command) error {
	return nil
}
