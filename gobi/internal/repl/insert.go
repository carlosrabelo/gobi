package repl

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func cmdINSERT(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	args := strings.ToUpper(cmd.Args)
	before := strings.Contains(args, "BEFORE")
	blank := strings.Contains(args, "BLANK")
	if uint64(a.RecCount()) >= maxRecords {
		return fail(ctx, "record count limit exceeded")
	}
	pos := a.Cursor
	if pos < 1 {
		pos = 1
	}
	if !before {
		pos = a.Cursor + 1
		if a.Cursor > a.RecCount() {
			pos = a.RecCount() + 1
		}
	}
	fields := a.Table.Header.Fields
	var rec dbf.Record
	if blank {
		rec = dbf.Record{Values: make([]dbf.Value, len(fields))}
	} else {
		// one blank for tests / non-interactive: treat like BLANK when no tty prompts
		rec = dbf.Record{Values: make([]dbf.Value, len(fields))}
	}
	if err := insertAt(ctx, pos, rec); err != nil {
		return err
	}
	a.Cursor = pos
	a.BOF = false
	return a.LoadCurrent()
}

func insertAt(ctx *context.Context, pos int, rec dbf.Record) error {
	a := ctx.Active()
	path := a.Path
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gobi-ins-*")
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	tmpName := tmp.Name()
	hdr := a.Table.Header
	tab, err := dbf.Create(tmp, hdr.Fields, hdr.Year, hdr.Month, hdr.Day)
	if err != nil {
		tmp.Close()
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	count := a.RecCount()
	for n := 1; n <= count+1; n++ {
		var r dbf.Record
		if n == pos {
			r = rec
		} else {
			src := n
			if n > pos {
				src = n - 1
			}
			r, err = a.Table.Read(src)
			if err != nil {
				tmp.Close()
				_ = os.Remove(tmpName)
				return fail(ctx, "%s", err.Error())
			}
		}
		if err := tab.Append(r); err != nil {
			tmp.Close()
			_ = os.Remove(tmpName)
			return fail(ctx, "%s", err.Error())
		}
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	alias := a.Alias
	_ = ctx.CloseArea(ctx.ActiveArea)
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	if err := ctx.Use(path, alias); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return nil
}
