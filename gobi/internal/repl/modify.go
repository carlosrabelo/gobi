package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func cmdModifyStructure(ctx *context.Context, cmd command.Command) error {
	_ = cmd
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	fmt.Fprintln(ctx.Out(), "Current structure:")
	for _, f := range a.Table.Header.Fields {
		fmt.Fprintf(ctx.Out(), "%-10s %c %3d %3d\n", f.Name, f.Type, f.Length, f.Decimals)
	}
	newFields, err := promptFields(ctx)
	if err != nil {
		return err
	}
	if len(newFields) == 0 {
		return fail(ctx, "no fields defined")
	}
	if _, err := dbf.ValidateFields(newFields); err != nil {
		return fail(ctx, "%s", err.Error())
	}

	path := a.Path
	alias := a.Alias
	oldFields := a.Table.Header.Fields
	count := a.RecCount()
	oldRecs := make([]dbf.Record, 0, count)
	for n := 1; n <= count; n++ {
		rec, err := a.Table.Read(n)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		oldRecs = append(oldRecs, rec)
	}

	now := time.Now()
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gobi-mod-*")
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	tmpName := tmp.Name()
	tab, err := dbf.Create(tmp, newFields, byte(now.Year()-1900), byte(now.Month()), byte(now.Day()))
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	oldIdx := map[string]int{}
	for i, f := range oldFields {
		oldIdx[f.Name] = i
	}
	for _, old := range oldRecs {
		rec := dbf.Record{Deleted: old.Deleted, Values: make([]dbf.Value, len(newFields))}
		for i, nf := range newFields {
			if oi, ok := oldIdx[nf.Name]; ok {
				rec.Values[i] = coerceFieldValue(nf, oldFields[oi], old.Values[oi])
			} else {
				rec.Values[i] = blankValue(nf)
			}
		}
		if err := tab.Append(rec); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
			return fail(ctx, "%s", err.Error())
		}
	}
	if err := tab.Close(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
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
	return a.LoadCurrent()
}

func coerceFieldValue(dst, src dbf.Field, v dbf.Value) dbf.Value {
	if dst.Type == src.Type {
		if dst.Type == 'C' && len(v.Str) > int(dst.Length) {
			v.Str = v.Str[:dst.Length]
		}
		return v
	}
	// Best-effort via string form.
	s := fieldString(src, v)
	nv, err := parseFieldInput(dst, s)
	if err != nil {
		return blankValue(dst)
	}
	return nv
}
