package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/internal/symbols"
)

func handleRelease(ctx *context.Context, cmd Command) error {
	args := strings.TrimSpace(cmd.Args)
	if args == "" {
		return fmt.Errorf("*** RELEASE requires variable name or ALL")
	}

	if strings.EqualFold(args, "ALL") {
		ctx.Variables.Clear()
		return nil
	}

	names := splitCommaOutsideParens(args)
	for _, part := range names {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if strings.EqualFold(name, "ALL") {
			ctx.Variables.Clear()
			return nil
		}
		if err := symbols.ValidateName(name); err != nil {
			return fmt.Errorf("*** Invalid memory variable name %s: %v", name, err)
		}
		if _, err := ctx.Variables.Delete(name); err != nil {
			return err
		}
	}

	return nil
}
