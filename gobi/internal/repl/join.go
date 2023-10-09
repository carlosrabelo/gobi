package repl

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func cmdJOIN(ctx *context.Context, cmd command.Command) error {
	active := ctx.Active()
	if !active.HasTable() {
		return fail(ctx, "no database in use")
	}
	alias, target, fieldNames, err := parseJoinArgs(cmd)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	forSrc := strings.TrimSpace(cmd.Clauses["FOR"])
	if forSrc == "" {
		return fail(ctx, "JOIN requires FOR")
	}
	forExpr, err := parseOptionalLogic(forSrc)
	if err != nil || forExpr == nil {
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		return fail(ctx, "JOIN requires FOR")
	}

	other := ctx.AreaByAlias(alias)
	if other == nil || !other.HasTable() {
		return fail(ctx, "alias not open: %s", alias)
	}

	pathName := target
	if filepath.Ext(pathName) == "" {
		pathName += ".dbf"
	}
	path, err := ctx.Resolve(pathName)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if fileExists(path) && ctx.Config.Safety {
		if _, err := confirmSafety(ctx, path+" already exists. Overwrite it? (Y/N)"); err != nil {
			return err
		}
	}

	outFields := joinFields(active.Table.Header.Fields, other.Table.Header.Fields, fieldNames)
	now := time.Now()
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gobi-join-*")
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	tmpName := tmp.Name()
	tab, err := dbf.Create(tmp, outFields, byte(now.Year()-1900), byte(now.Month()), byte(now.Day()))
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}

	env := newEnv(ctx)
	savedAct, savedOth := active.Cursor, other.Cursor
	var kept uint64
	aCount, oCount := active.RecCount(), other.RecCount()
	for ai := 1; ai <= aCount; ai++ {
		active.Cursor = ai
		if err := active.LoadCurrent(); err != nil {
			cleanupJoin(tmp, tmpName)
			return fail(ctx, "%s", err.Error())
		}
		if active.Record.Deleted {
			continue
		}
		for oi := 1; oi <= oCount; oi++ {
			other.Cursor = oi
			if err := other.LoadCurrent(); err != nil {
				cleanupJoin(tmp, tmpName)
				return fail(ctx, "%s", err.Error())
			}
			if other.Record.Deleted {
				continue
			}
			ok, err := evalLogic(forExpr, env)
			if err != nil {
				cleanupJoin(tmp, tmpName)
				return fail(ctx, "%s", err.Error())
			}
			if !ok {
				continue
			}
			if kept >= maxRecords {
				cleanupJoin(tmp, tmpName)
				return fail(ctx, "record count limit exceeded")
			}
			rec := buildJoinRecord(outFields, active, other)
			if err := tab.Append(rec); err != nil {
				cleanupJoin(tmp, tmpName)
				return fail(ctx, "%s", err.Error())
			}
			kept++
		}
	}
	active.Cursor, other.Cursor = savedAct, savedOth
	_ = active.LoadCurrent()
	_ = other.LoadCurrent()

	if err := tab.Close(); err != nil {
		cleanupJoin(tmp, tmpName)
		return fail(ctx, "%s", err.Error())
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	return nil
}

func cleanupJoin(tmp *os.File, name string) {
	_ = tmp.Close()
	_ = os.Remove(name)
}

func parseJoinArgs(cmd command.Command) (alias, target string, fields []string, err error) {
	to := strings.TrimSpace(cmd.Clauses["TO"])
	if to == "" {
		return "", "", nil, errString("JOIN requires TO")
	}
	toks := strings.Fields(to)
	target = toks[0]
	if len(toks) > 1 {
		joined := strings.Join(toks[1:], " ")
		for _, p := range strings.Split(joined, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				fields = append(fields, strings.ToUpper(p))
			}
		}
	}
	args := strings.Fields(cmd.Args)
	if len(args) < 2 || !strings.EqualFold(args[0], "WITH") {
		return "", "", nil, errString("JOIN requires WITH alias")
	}
	alias = strings.ToUpper(args[1])
	return alias, target, fields, nil
}

func joinFields(aFields, bFields []dbf.Field, limit []string) []dbf.Field {
	seen := map[string]bool{}
	var out []dbf.Field
	for _, f := range aFields {
		if len(limit) > 0 && !nameIn(limit, f.Name) {
			continue
		}
		out = append(out, f)
		seen[f.Name] = true
	}
	for _, f := range bFields {
		if seen[f.Name] {
			continue
		}
		if len(limit) > 0 && !nameIn(limit, f.Name) {
			continue
		}
		out = append(out, f)
	}
	if len(limit) == 0 {
		return out
	}
	// When limit given, order by limit list using first occurrence from a then b.
	var ordered []dbf.Field
	byName := map[string]dbf.Field{}
	for _, f := range out {
		byName[f.Name] = f
	}
	for _, n := range limit {
		if f, ok := byName[n]; ok {
			ordered = append(ordered, f)
		}
	}
	return ordered
}

func nameIn(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

func buildJoinRecord(outFields []dbf.Field, active, other *context.Area) dbf.Record {
	rec := dbf.Record{Values: make([]dbf.Value, len(outFields))}
	aMap := map[string]dbf.Value{}
	for i, f := range active.Table.Header.Fields {
		aMap[f.Name] = active.Record.Values[i]
	}
	bMap := map[string]dbf.Value{}
	for i, f := range other.Table.Header.Fields {
		bMap[f.Name] = other.Record.Values[i]
	}
	for i, f := range outFields {
		if v, ok := aMap[f.Name]; ok {
			rec.Values[i] = v
		} else if v, ok := bMap[f.Name]; ok {
			rec.Values[i] = v
		} else {
			rec.Values[i] = blankValue(f)
		}
	}
	return rec
}
