package repl

import (
	"fmt"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/mem"
)

func handleRestore(ctx *context.Context, cmd Command) error {
	if strings.TrimSpace(cmd.FromClause) == "" {
		return fmt.Errorf("*** RESTORE requires FROM clause")
	}

	filePath := resolveMemFilePath(ctx, cmd.FromClause)
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("*** Could not open file: %w", err)
	}
	defer f.Close()

	vars, err := mem.Read(f)
	if err != nil {
		return fmt.Errorf("*** Error reading memory file: %w", err)
	}

	ctx.Variables.Clear()
	for _, v := range vars {
		if err := ctx.Variables.Set(v.Name, v.Value); err != nil {
			return err
		}
	}

	return nil
}
