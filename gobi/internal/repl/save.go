package repl

import (
	"fmt"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/mem"
)

func handleSave(ctx *context.Context, cmd Command) error {
	if strings.TrimSpace(cmd.ToClause) == "" {
		return fmt.Errorf("*** SAVE requires TO clause")
	}

	filePath := resolveMemFilePath(ctx, cmd.ToClause)
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("*** Could not create memory file: %w", err)
	}
	defer f.Close()

	vars := make([]mem.Variable, 0, ctx.Variables.Len())
	for _, name := range ctx.Variables.Names() {
		val, _ := ctx.Variables.Get(name)
		vars = append(vars, mem.Variable{Name: name, Value: val})
	}

	if err := mem.Write(f, vars); err != nil {
		return fmt.Errorf("*** Error writing memory file: %w", err)
	}

	return nil
}
