package repl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	// III Plus–style columns: name, records, bytes, last update.
	fmt.Fprintf(ctx.Out(), "%-16s %8s %10s  %s\n", "File", "Records", "Bytes", "Last Update")
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
		updated := info.ModTime()
		if f, err := os.Open(path); err == nil {
			if tab, err := dbf.Open(f); err == nil {
				recs = int64(tab.Header.RecordCount)
				if t, ok := dbfHeaderDate(tab.Header); ok {
					updated = t
				}
			}
			f.Close()
		}
		fmt.Fprintf(ctx.Out(), "%-16s %8d %10d  %s\n",
			name, recs, info.Size(), updated.Format("01/02/06"))
	}
	return nil
}

func dbfHeaderDate(h dbf.Header) (time.Time, bool) {
	if h.Year == 0 && h.Month == 0 && h.Day == 0 {
		return time.Time{}, false
	}
	y := 1900 + int(h.Year)
	t := time.Date(y, time.Month(h.Month), int(h.Day), 0, 0, 0, 0, time.Local)
	return t, true
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
