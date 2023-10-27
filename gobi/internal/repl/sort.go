package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func cmdSORT(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	args := strings.TrimSpace(cmd.Args)
	up := strings.ToUpper(args)
	if !strings.HasPrefix(up, "ON ") {
		return fail(ctx, "SORT requires ON")
	}
	rest := strings.TrimSpace(args[3:])
	to := strings.TrimSpace(cmd.Clauses["TO"])
	if to == "" {
		return fail(ctx, "SORT ON requires TO")
	}
	toName := strings.Fields(to)[0]
	descending := strings.Contains(up, "DESCENDING") || strings.Contains(strings.ToUpper(to), "DESCENDING")
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return fail(ctx, "SORT ON requires a field")
	}
	fieldName := fields[0]
	fi := fieldIndex(a, fieldName)
	if fi < 0 {
		return fail(ctx, "unknown field: %s", fieldName)
	}

	path, err := resolveDBFOut(ctx, toName)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if fileExists(path) && ctx.Config.Safety {
		if _, err := confirmSafety(ctx, path+" already exists. Overwrite it? (Y/N)"); err != nil {
			return err
		}
	}

	type row struct {
		key string
		rec dbf.Record
	}
	var rows []row
	saved := a.Cursor
	count := a.RecCount()
	for n := 1; n <= count; n++ {
		a.Cursor = n
		if err := a.LoadCurrent(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if a.Record.Deleted {
			continue
		}
		rows = append(rows, row{
			key: fieldSortKey(a.Record.Values[fi], a.Table.Header.Fields[fi]),
			rec: cloneRecord(a.Record),
		})
	}
	a.Cursor = saved
	_ = a.LoadCurrent()

	sort.SliceStable(rows, func(i, j int) bool {
		if descending {
			return rows[i].key > rows[j].key
		}
		return rows[i].key < rows[j].key
	})

	tmp := path + ".tmp"
	_ = os.Remove(tmp)
	f, err := os.Create(tmp)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	hdr := a.Table.Header
	tab, err := dbf.Create(f, hdr.Fields, hdr.Year, hdr.Month, hdr.Day)
	if err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return fail(ctx, "%s", err.Error())
	}
	for _, r := range rows {
		if err := tab.Append(r.rec); err != nil {
			f.Close()
			_ = os.Remove(tmp)
			return fail(ctx, "%s", err.Error())
		}
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fail(ctx, "%s", err.Error())
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fail(ctx, "%s", err.Error())
	}
	return nil
}

func resolveDBFOut(ctx *context.Context, name string) (string, error) {
	name = strings.TrimSpace(name)
	if filepath.Ext(name) == "" {
		name += ".dbf"
	}
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

func fieldSortKey(v dbf.Value, f dbf.Field) string {
	switch f.Type {
	case 'N':
		return fmt.Sprintf("%*.*f", f.Length, f.Decimals, v.Num)
	case 'L':
		if v.Log {
			return "T"
		}
		return "F"
	default:
		return v.Str
	}
}
