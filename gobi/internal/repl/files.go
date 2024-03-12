package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

// handleErase implements the dBase II ERASE command, which clears the
// screen and homes the cursor. File removal is done by DELETE FILE.
func handleErase(ctx *context.Context, cmd Command) error {
	if args := strings.TrimSpace(cmd.Args); args != "" {
		return fmt.Errorf("*** ERASE takes no arguments; use DELETE FILE %s", args)
	}
	return presentClearScreen(ctx)
}

func handleRename(ctx *context.Context, cmd Command) error {
	oldName := strings.TrimSpace(cmd.Args)
	newName := strings.TrimSpace(cmd.ToClause)
	if oldName == "" || newName == "" {
		return fmt.Errorf("*** RENAME requires <filename> TO <newname>")
	}
	return renameFile(ctx, oldName, newName)
}

// deleteFile removes a file from disk after checking it is not open in a
// work area (DELETE FILE).
func deleteFile(ctx *context.Context, filename string) error {
	path := resolveFileHookPath(ctx, filename)
	if err := assertFileNotOpen(ctx, path); err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("*** File not found: %s", filename)
		}
		return fmt.Errorf("*** Error accessing file: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("*** Error deleting file: %w", err)
	}
	talkPrint(ctx, "FILE HAS BEEN DELETED\r\n")
	return nil
}

func renameFile(ctx *context.Context, oldName, newName string) error {
	oldPath := resolveFileHookPath(ctx, oldName)
	newPath := resolveRenamedFilePath(ctx, oldPath, newName)

	if err := assertFileNotOpen(ctx, oldPath); err != nil {
		return err
	}
	if _, err := os.Stat(oldPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("*** File not found: %s", oldName)
		}
		return fmt.Errorf("*** Error accessing file: %w", err)
	}
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("*** File already exists: %s", newName)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("*** Error accessing file: %w", err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("*** Error renaming file: %w", err)
	}
	return nil
}

func resolveFileHookPath(ctx *context.Context, filename string) string {
	return resolveDataPath(ctx, filename, defaultHookExtension(filename))
}

func resolveRenamedFilePath(ctx *context.Context, oldPath, newName string) string {
	newName = strings.TrimSpace(newName)
	if !strings.Contains(newName, ".") {
		ext := filepath.Ext(oldPath)
		if ext == "" {
			ext = ".dbf"
		}
		newName += ext
	}
	return resolveDataPath(ctx, newName, "")
}

func defaultHookExtension(filename string) string {
	if strings.Contains(filepath.Base(filename), ".") {
		return ""
	}
	return ".dbf"
}

func assertFileNotOpen(ctx *context.Context, path string) error {
	clean := filepath.Clean(path)
	for _, area := range ctx.WorkAreas {
		if area == nil {
			continue
		}
		if area.Table != nil {
			if tablePath, ok := tableFilePath(area.Table); ok && tablePath == clean {
				return fmt.Errorf("*** File is in use")
			}
		}
		for _, idx := range area.Indexes {
			if idx != nil && filepath.Clean(idx.Path) == clean {
				return fmt.Errorf("*** File is in use")
			}
		}
	}
	return nil
}

func tableFilePath(tbl *dbf.Table) (string, bool) {
	if tbl == nil {
		return "", false
	}
	if f, ok := tbl.Underlying().(*os.File); ok {
		return filepath.Clean(f.Name()), true
	}
	return "", false
}
