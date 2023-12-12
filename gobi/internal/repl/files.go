package repl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func cmdDIR(ctx *context.Context, cmd command.Command) error {
	dir := ctx.Config.Default
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.EqualFold(filepath.Ext(name), ".dbf") {
			continue
		}
		path := filepath.Join(dir, name)
		info, err := e.Info()
		if err != nil {
			continue
		}
		recs := int64(0)
		if f, err := os.Open(path); err == nil {
			if tab, err := dbf.Open(f); err == nil {
				recs = int64(tab.Header.RecordCount)
			}
			f.Close()
		}
		fmt.Fprintf(ctx.Out(), "%s  %d  %d\n", name, info.Size(), recs)
	}
	return nil
}

func cmdRENAME(ctx *context.Context, cmd command.Command) error {
	args := strings.TrimSpace(cmd.Args)
	to := strings.TrimSpace(cmd.Clauses["TO"])
	if to == "" {
		return fail(ctx, "RENAME requires TO")
	}
	oldName := strings.Fields(args)[0]
	newName := strings.Fields(to)[0]
	oldPath, err := resolveAny(ctx, oldName)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	newPath, err := resolveAny(ctx, newName)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return nil
}

func cmdDELETEFILE(ctx *context.Context, cmd command.Command) error {
	args := strings.Fields(cmd.Args)
	if len(args) < 2 || !strings.EqualFold(args[0], "FILE") {
		return fail(ctx, "DELETE FILE requires a name")
	}
	path, err := resolveAny(ctx, args[1])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if err := os.Remove(path); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return nil
}

func cmdCOPYFILE(ctx *context.Context, cmd command.Command) error {
	args := strings.Fields(cmd.Args)
	if len(args) < 2 || !strings.EqualFold(args[0], "FILE") {
		return fail(ctx, "COPY FILE requires a name")
	}
	oldName := args[1]
	to := strings.TrimSpace(cmd.Clauses["TO"])
	if to == "" {
		return fail(ctx, "COPY FILE requires TO")
	}
	newName := strings.Fields(to)[0]
	src, err := resolveAny(ctx, oldName)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	dst, err := resolveAny(ctx, newName)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if !fileExists(src) {
		return fail(ctx, "file not found: %s", src)
	}
	if fileExists(dst) && ctx.Config.Safety {
		if _, err := confirmSafety(ctx, dst+" already exists. Overwrite it? (Y/N)"); err != nil {
			return err
		}
	}
	in, err := os.Open(src)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return nil
}

func resolveAny(ctx *context.Context, name string) (string, error) {
	if filepath.IsAbs(name) {
		return name, nil
	}
	base := ctx.Config.Default
	if base == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		base = wd
	}
	return filepath.Join(base, name), nil
}
