package repl

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// applySetDefault updates the default data directory and prints the dBase confirmation line.
func applySetDefault(ctx *context.Context, args string) error {
	rawParts := strings.Fields(strings.TrimSpace(args))
	if len(rawParts) < 2 {
		return fmt.Errorf("*** SET DEFAULT requires a drive/directory argument")
	}

	dir := rawParts[len(rawParts)-1]
	ctx.Config.DefaultDir = filepath.Clean(dir)
	fmt.Fprintf(ctx.Stdout, "Default directory: %s\r\n", ctx.Config.DefaultDir)
	return nil
}

// resolveDataPath maps a relative filename through SET DEFAULT and optionally appends an extension.
func resolveDataPath(ctx *context.Context, filename, defaultExt string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return filename
	}
	if defaultExt != "" && !strings.Contains(filename, ".") {
		filename += defaultExt
	}
	if ctx == nil || ctx.Config == nil || ctx.Config.DefaultDir == "" || filepath.IsAbs(filename) {
		return filepath.Clean(filename)
	}
	return filepath.Clean(filepath.Join(ctx.Config.DefaultDir, filename))
}

// resolveDBFFilePath maps a database filename through SET DEFAULT, appending .dbf when needed.
func resolveDBFFilePath(ctx *context.Context, filename string) string {
	return resolveDataPath(ctx, filename, ".dbf")
}

// resolveNDXFilePath maps an index filename through SET DEFAULT, appending .ndx when needed.
func resolveNDXFilePath(ctx *context.Context, filename string) string {
	return resolveDataPath(ctx, filename, ".ndx")
}

// resolveMemFilePath maps a memory file through SET DEFAULT, appending .mem when needed.
func resolveMemFilePath(ctx *context.Context, filename string) string {
	return resolveDataPath(ctx, filename, ".mem")
}

// resolveTextImportPath maps a text import file through SET DEFAULT, appending .txt when needed.
func resolveTextImportPath(ctx *context.Context, filename string) string {
	return resolveDataPath(ctx, filename, ".txt")
}

// resolveOutputPath maps a generic output filename through SET DEFAULT without adding an extension.
func resolveOutputPath(ctx *context.Context, filename string) string {
	return resolveDataPath(ctx, filename, "")
}
