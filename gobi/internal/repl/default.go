package repl

import (
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

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

func resolveDBFFilePath(ctx *context.Context, filename string) string {
	return resolveDataPath(ctx, filename, ".dbf")
}

func resolveMemFilePath(ctx *context.Context, filename string) string {
	return resolveDataPath(ctx, filename, ".mem")
}

func resolveTextImportPath(ctx *context.Context, filename string) string {
	return resolveDataPath(ctx, filename, ".txt")
}

func resolveOutputPath(ctx *context.Context, filename string) string {
	return resolveDataPath(ctx, filename, "")
}
