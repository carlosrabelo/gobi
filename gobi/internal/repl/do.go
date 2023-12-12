package repl

import (
	"fmt"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/script"
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

	_, err := loadScript(ctx, args)
	return err
}

func loadScript(ctx *context.Context, filename string) (*script.Program, error) {
	path := resolveDataPath(ctx, filename, ".prg")

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("*** Command file not found")
		}
		return nil, fmt.Errorf("*** %s", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("*** not a command file")
	}

	return &script.Program{Path: path}, nil
}
