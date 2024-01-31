package repl

import (
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
)

func init() {
	help.Register("MODIFY", "MODIFY STRUCTURE|COMMAND|REPORT|LABEL|… — change structure or design a form.")
}

func cmdMODIFY(ctx *context.Context, cmd command.Command) error {
	if handled, err := modifyDesignerCmd(ctx, cmd); handled {
		return err
	}
	args := strings.Fields(cmd.Args)
	if len(args) == 0 {
		return fail(ctx, "MODIFY STRUCTURE or MODIFY COMMAND required")
	}
	if strings.EqualFold(args[0], "STRUCTURE") {
		return cmdModifyStructure(ctx, cmd)
	}
	if strings.EqualFold(args[0], "COMMAND") {
		name := ""
		if len(args) >= 2 {
			name = args[1]
		}
		if name == "" {
			return fail(ctx, "MODIFY COMMAND requires a name")
		}
		return modifyCommand(ctx, name)
	}
	return fail(ctx, "MODIFY STRUCTURE or MODIFY COMMAND required")
}

func modifyCommand(ctx *context.Context, name string) error {
	if filepath.Ext(name) == "" {
		name += ".prg"
	}
	path, err := ctx.Resolve(name)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return editTextFile(ctx, path, "MODIFY COMMAND", nil)
}
