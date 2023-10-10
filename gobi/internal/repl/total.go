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

func cmdTOTAL(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	key, target, sumFields, err := parseTotalArgs(cmd)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	ki := fieldIndex(a, key)
	if ki < 0 {
		return fail(ctx, "unknown key field: %s", key)
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

	fields := a.Table.Header.Fields
	sumSet := map[string]bool{}
	if len(sumFields) > 0 {
		for _, n := range sumFields {
			sumSet[n] = true
		}
	} else {
		for _, f := range fields {
			if f.Type == 'N' && f.Name != strings.ToUpper(key) {
				sumSet[f.Name] = true
			}
		}
	}

	now := time.Now()
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gobi-total-*")
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	tmpName := tmp.Name()
	tab, err := dbf.Create(tmp, fields, byte(now.Year()-1900), byte(now.Month()), byte(now.Day()))
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}

	count := a.RecCount()
	var (
		have   bool
		curKey string
		acc    dbf.Record
	)
	flush := func() error {
		if !have {
			return nil
		}
		return tab.Append(acc)
	}
	for n := 1; n <= count; n++ {
		rec, err := a.Table.Read(n)
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
			return fail(ctx, "%s", err.Error())
		}
		if rec.Deleted {
			continue
		}
		k := strings.ToUpper(fieldString(fields[ki], rec.Values[ki]))
		if !have {
			have = true
			curKey = k
			acc = cloneRecord(rec)
			continue
		}
		if k != curKey {
			if err := flush(); err != nil {
				_ = tmp.Close()
				_ = os.Remove(tmpName)
				return fail(ctx, "%s", err.Error())
			}
			curKey = k
			acc = cloneRecord(rec)
			continue
		}
		for i, f := range fields {
			if sumSet[f.Name] && f.Type == 'N' {
				acc.Values[i].Num += rec.Values[i].Num
			}
		}
	}
	if err := flush(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
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
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	return nil
}

func cloneRecord(r dbf.Record) dbf.Record {
	out := dbf.Record{Deleted: r.Deleted, Values: make([]dbf.Value, len(r.Values))}
	copy(out.Values, r.Values)
	return out
}

func parseTotalArgs(cmd command.Command) (key, target string, sumFields []string, err error) {
	to := strings.TrimSpace(cmd.Clauses["TO"])
	if to == "" {
		return "", "", nil, errString("TOTAL requires TO")
	}
	target = strings.Fields(to)[0]
	args := strings.TrimSpace(cmd.Args)
	up := strings.ToUpper(args)
	if !strings.HasPrefix(up, "ON ") {
		return "", "", nil, errString("TOTAL requires ON")
	}
	rest := strings.TrimSpace(args[3:])
	// ON key [FIELDS a, b]
	toks := strings.Fields(rest)
	if len(toks) == 0 {
		return "", "", nil, errString("TOTAL ON requires a key")
	}
	key = toks[0]
	for i := 1; i < len(toks); i++ {
		if strings.EqualFold(toks[i], "FIELDS") {
			joined := strings.Join(toks[i+1:], " ")
			for _, p := range strings.Split(joined, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					sumFields = append(sumFields, strings.ToUpper(p))
				}
			}
			break
		}
	}
	return key, target, sumFields, nil
}
