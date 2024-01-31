package repl

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
	"github.com/carlosrabelo/gobi/gobi/pkg/hostmod"
)

func registerHostExchange(m *Mux) {
	m.Register("EXPORT", cmdEXPORT)
	m.Register("IMPORT", cmdIMPORT)
	m.Register("LOAD", cmdLOAD)
	m.Register("CALL", cmdCALL)
	m.Register("UNLOCK", cmdUNLOCK)
	m.Register("LOGOUT", cmdLOGOUT)
	help.Register("EXPORT", "EXPORT TO <file> [DELIMITED|SDF] — write the active table as host text.")
	help.Register("IMPORT", "IMPORT FROM <file> [DELIMITED|SDF] — build a .dbf from host text.")
	help.Register("LOAD", "LOAD <name> — load a host module.")
	help.Register("CALL", "CALL <name> [args…] — call a loaded module entry.")
	help.Register("RELEASE MODULE", "RELEASE MODULE <name> — unload a host module.")
	help.Register("UNLOCK", "UNLOCK — release file locks on the active table.")
	help.Register("LOGOUT", "LOGOUT — no-op on a local session (tables stay open).")
	help.Register("FLOCK", "FLOCK() — advisory lock on the active table.")
	help.Register("RLOCK", "RLOCK() — advisory lock on the current record's table.")
	help.Register("LOCK", "LOCK() — same as RLOCK().")
	help.Register("SET EXCLUSIVE", "SET EXCLUSIVE ON|OFF — exclusive USE locking.")
	help.Register("SET ENCRYPTION", "SET ENCRYPTION ON|OFF — accepted; local no-op.")
	help.Register("DISPLAY USERS", "DISPLAY USERS — local session; prints nothing.")
	help.Register("LIST USERS", "LIST USERS — local session; prints nothing.")
}

func cmdEXPORT(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return failCode(ctx, 26, "no table is open")
	}
	to := strings.TrimSpace(cmd.Clauses["TO"])
	rest := strings.ToUpper(strings.TrimSpace(cmd.Args))
	if to == "" {
		fields := strings.Fields(cmd.Args)
		if len(fields) == 0 {
			return fail(ctx, "EXPORT requires TO <file>")
		}
		to = fields[0]
		rest = strings.ToUpper(strings.Join(fields[1:], " "))
	} else {
		// Splitter may leave mode words in the TO clause: "out.txt DELIMITED".
		tf := strings.Fields(to)
		if len(tf) > 1 {
			to = tf[0]
			rest = strings.ToUpper(strings.Join(append(tf[1:], strings.Fields(rest)...), " "))
		}
	}
	sdf := strings.Contains(rest, "SDF")
	delim := byte('"')
	path, err := resolveOutPath(ctx, to, ".txt")
	if err != nil {
		return err
	}
	if fileExists(path) && ctx.Config.Safety {
		if _, err := confirmSafety(ctx, path+" already exists. Overwrite it? (Y/N)"); err != nil {
			return err
		}
	}
	idxs, outFields, err := selectFields(a.Table.Header.Fields, nil)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	f, err := os.Create(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	defer f.Close()
	saved := a.Cursor
	for n := 1; n <= a.RecCount(); n++ {
		a.Cursor = n
		if err := a.LoadCurrent(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if a.Record.Deleted {
			continue
		}
		if sdf {
			line := encodeSDF(outFields, a.Record, idxs)
			if _, err := f.Write(append(line, '\n')); err != nil {
				return fail(ctx, "%s", err.Error())
			}
		} else {
			line := encodeDelimited(outFields, a.Record, idxs, delim)
			if _, err := fmt.Fprintln(f, line); err != nil {
				return fail(ctx, "%s", err.Error())
			}
		}
	}
	a.Cursor = saved
	_ = a.LoadCurrent()
	return nil
}

func cmdIMPORT(ctx *context.Context, cmd command.Command) error {
	from := strings.TrimSpace(cmd.Clauses["FROM"])
	args := strings.Fields(cmd.Args)
	rest := strings.ToUpper(strings.TrimSpace(cmd.Args))
	if from == "" {
		if len(args) == 0 {
			return fail(ctx, "IMPORT requires FROM <file>")
		}
		from = args[0]
		rest = strings.ToUpper(strings.Join(args[1:], " "))
	}
	sdf := strings.Contains(rest, "SDF")
	src, err := ctx.Resolve(from)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	f, err := os.Open(src)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return fail(ctx, "IMPORT: empty file")
	}
	first := sc.Text()
	var fields []dbf.Field
	var rows [][]string
	if sdf {
		width := len(first)
		if width < 1 {
			return fail(ctx, "IMPORT: bad SDF")
		}
		w := width
		if w > 254 {
			w = 254
		}
		fields = []dbf.Field{{Name: "FIELD1", Type: 'C', Length: byte(w)}}
		rows = append(rows, []string{first})
		for sc.Scan() {
			rows = append(rows, []string{sc.Text()})
		}
	} else {
		parts := splitCSVLine(first)
		if len(parts) == 0 {
			return fail(ctx, "IMPORT: bad DELIMITED")
		}
		for i := range parts {
			name := fmt.Sprintf("FIELD%d", i+1)
			if len(name) > 10 {
				name = name[:10]
			}
			fields = append(fields, dbf.Field{Name: name, Type: 'C', Length: 40})
		}
		rows = append(rows, parts)
		for sc.Scan() {
			rows = append(rows, splitCSVLine(sc.Text()))
		}
	}
	if err := sc.Err(); err != nil {
		return fail(ctx, "%s", err.Error())
	}

	outName := strings.TrimSpace(cmd.Clauses["TO"])
	if outName == "" {
		base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
		outName = base + ".dbf"
	}
	path, err := resolveOutPath(ctx, outName, ".dbf")
	if err != nil {
		return err
	}
	if fileExists(path) && ctx.Config.Safety {
		if _, err := confirmSafety(ctx, path+" already exists. Overwrite it? (Y/N)"); err != nil {
			return err
		}
	}
	now := time.Now()
	year := byte(now.Year() - 1900)
	month := byte(now.Month())
	day := byte(now.Day())
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gobi-import-*")
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	tmpName := tmp.Name()
	tbl, err := dbf.Create(tmp, fields, year, month, day)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	for _, row := range rows {
		rec := dbf.Record{Values: make([]dbf.Value, len(fields))}
		for i, fld := range fields {
			s := ""
			if i < len(row) {
				s = row[i]
			}
			v, err := parseFieldInput(fld, s)
			if err != nil {
				_ = tbl.Close()
				_ = tmp.Close()
				_ = os.Remove(tmpName)
				return fail(ctx, "%s", err.Error())
			}
			rec.Values[i] = v
		}
		if err := tbl.Append(rec); err != nil {
			_ = tbl.Close()
			_ = tmp.Close()
			_ = os.Remove(tmpName)
			return fail(ctx, "%s", err.Error())
		}
	}
	if err := tbl.Close(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	_ = tmp.Close()
	_ = os.Remove(path)
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	return nil
}

func cmdLOAD(ctx *context.Context, cmd command.Command) error {
	name := strings.TrimSpace(cmd.Args)
	if name == "" {
		return fail(ctx, "LOAD requires a module name")
	}
	key := strings.ToUpper(filepath.Base(name))
	key = strings.TrimSuffix(key, strings.ToUpper(filepath.Ext(key)))
	if _, ok := hostmod.LookupBuiltin(key); ok {
		if ctx.Modules == nil {
			ctx.Modules = map[string]string{}
		}
		ctx.Modules[key] = name
		return nil
	}
	if _, ok := hostmod.LookupBuiltin(strings.ToUpper(name)); ok {
		if ctx.Modules == nil {
			ctx.Modules = map[string]string{}
		}
		ctx.Modules[strings.ToUpper(name)] = name
		return nil
	}
	path, err := ctx.Resolve(name)
	if err != nil {
		path = name
	}
	if _, err := os.Stat(path); err != nil {
		return fail(ctx, "cannot load module %s", name)
	}
	if ctx.Modules == nil {
		ctx.Modules = map[string]string{}
	}
	ctx.Modules[key] = path
	return nil
}

func cmdCALL(ctx *context.Context, cmd command.Command) error {
	fields := strings.Fields(cmd.Args)
	if len(fields) == 0 {
		return fail(ctx, "CALL requires a module name")
	}
	key := strings.ToUpper(fields[0])
	args := fields[1:]
	if ctx.Modules == nil {
		return fail(ctx, "module %s not loaded", key)
	}
	if _, ok := ctx.Modules[key]; !ok {
		return fail(ctx, "module %s not loaded", key)
	}
	fn, ok := hostmod.LookupBuiltin(key)
	if !ok {
		return fail(ctx, "module entry %s not found", key)
	}
	if err := fn(args); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return nil
}

func releaseModule(ctx *context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fail(ctx, "RELEASE MODULE requires a name")
	}
	key := strings.ToUpper(filepath.Base(name))
	key = strings.TrimSuffix(key, strings.ToUpper(filepath.Ext(key)))
	if ctx.Modules != nil {
		delete(ctx.Modules, key)
		delete(ctx.Modules, strings.ToUpper(name))
	}
	return nil
}

func cmdUNLOCK(ctx *context.Context, _ command.Command) error {
	releaseAreaLock(ctx.Active())
	return nil
}

func cmdLOGOUT(ctx *context.Context, _ command.Command) error {
	return nil
}
