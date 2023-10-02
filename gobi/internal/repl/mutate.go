package repl

import (
	"os"
	"path/filepath"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func cmdDELETE(ctx *context.Context, cmd command.Command) error {
	return markDeleted(ctx, cmd, true)
}

func cmdRECALL(ctx *context.Context, cmd command.Command) error {
	return markDeleted(ctx, cmd, false)
}

func markDeleted(ctx *context.Context, cmd command.Command, deleted bool) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	forExpr, err := parseOptionalLogic(cmd.Clauses["FOR"])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	whileExpr, err := parseOptionalLogic(cmd.Clauses["WHILE"])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	env := newEnv(ctx)

	if forExpr == nil && whileExpr == nil {
		if !a.Landed {
			return fail(ctx, "no current record")
		}
		return writeMark(ctx, deleted)
	}

	count := a.RecCount()
	start := 1
	if whileExpr != nil {
		start = a.Cursor
		if start < 1 {
			start = 1
		}
	}
	last := 0
	for n := start; n <= count; n++ {
		a.Cursor = n
		a.BOF = false
		if err := a.LoadCurrent(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if whileExpr != nil {
			ok, err := evalLogic(whileExpr, env)
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			if !ok {
				break
			}
		}
		if forExpr != nil {
			ok, err := evalLogic(forExpr, env)
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			if !ok {
				continue
			}
		}
		if err := writeMark(ctx, deleted); err != nil {
			return err
		}
		last = n
	}
	if last > 0 {
		a.Cursor = last
		_ = a.LoadCurrent()
	}
	return nil
}

func writeMark(ctx *context.Context, deleted bool) error {
	a := ctx.Active()
	rec := a.Record
	rec.Deleted = deleted
	if err := a.Table.Write(a.Cursor, rec); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	a.Record = rec
	return nil
}

func cmdPACK(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	path := a.Path
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gobi-pack-*")
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	tmpName := tmp.Name()
	kept, err := dbf.PackNonDeleted(a.Table, tmp)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	alias := a.Alias
	if err := ctx.CloseArea(ctx.ActiveArea); err != nil {
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	if err := ctx.Use(path, alias); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	a = ctx.Active()
	a.Cursor = 1
	a.BOF = false
	_ = a.LoadCurrent()
	_ = kept
	return nil
}

func cmdZAP(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	if ctx.Config.Safety {
		_, err := confirmSafety(ctx, "Zap "+a.Path+"? (Y/N)")
		if err != nil {
			return err
		}
	}
	if err := a.Table.Zap(); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	a.Cursor = 1
	a.BOF = false
	return a.LoadCurrent()
}
