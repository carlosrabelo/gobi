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

func cmdCOPY(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	// Split treats TO as a clause: COPY TO name … → Clauses["TO"], Args = trailing modes/fields.
	to := strings.TrimSpace(cmd.Clauses["TO"])
	if to == "" {
		return fail(ctx, "COPY requires TO")
	}
	tokens := strings.Fields(to)
	target := tokens[0]
	rest := append([]string{}, tokens[1:]...)
	rest = append(rest, strings.Fields(cmd.Args)...)
	mode, fieldList := parseCopyModeAndFields(rest)
	switch mode {
	case "STRUCTURE":
		return copyStructure(ctx, target, false)
	case "STRUCTURE EXTENDED":
		return copyStructure(ctx, target, true)
	case "SDF", "DELIMITED":
		return fail(ctx, "COPY %s arrives in 012.2", mode)
	default:
		return copyDBF(ctx, cmd, target, fieldList, mode)
	}
}

func parseCopyModeAndFields(tokens []string) (mode string, fields []string) {
	i := 0
	var fieldParts []string
	for i < len(tokens) {
		t := strings.ToUpper(tokens[i])
		if t == "SDF" || t == "DELIMITED" || t == "STRUCTURE" {
			break
		}
		fieldParts = append(fieldParts, tokens[i])
		i++
	}
	if len(fieldParts) > 0 {
		joined := strings.Join(fieldParts, " ")
		for _, p := range strings.Split(joined, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				fields = append(fields, strings.ToUpper(p))
			}
		}
	}
	if i < len(tokens) {
		t := strings.ToUpper(tokens[i])
		switch t {
		case "SDF":
			mode = "SDF"
		case "DELIMITED":
			mode = "DELIMITED"
		case "STRUCTURE":
			mode = "STRUCTURE"
			if i+1 < len(tokens) && strings.EqualFold(tokens[i+1], "EXTENDED") {
				mode = "STRUCTURE EXTENDED"
			}
		}
	}
	return mode, fields
}

func copyDBF(ctx *context.Context, cmd command.Command, target string, fieldNames []string, _ string) error {
	a := ctx.Active()
	savedCursor := a.Cursor
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

	srcFields := a.Table.Header.Fields
	outFields := srcFields
	var idxs []int
	if len(fieldNames) > 0 {
		outFields = nil
		for _, name := range fieldNames {
			fi := -1
			for i, f := range srcFields {
				if f.Name == name {
					fi = i
					break
				}
			}
			if fi < 0 {
				return fail(ctx, "unknown field: %s", name)
			}
			outFields = append(outFields, srcFields[fi])
			idxs = append(idxs, fi)
		}
	} else {
		for i := range srcFields {
			idxs = append(idxs, i)
		}
	}

	forExpr, err := parseOptionalLogic(cmd.Clauses["FOR"])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	env := newEnv(ctx)

	now := time.Now()
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gobi-copy-*")
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

	count := a.RecCount()
	for n := 1; n <= count; n++ {
		a.Cursor = n
		if err := a.LoadCurrent(); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
			return fail(ctx, "%s", err.Error())
		}
		if a.Record.Deleted {
			continue
		}
		if forExpr != nil {
			ok, err := evalLogic(forExpr, env)
			if err != nil {
				_ = tmp.Close()
				_ = os.Remove(tmpName)
				return fail(ctx, "%s", err.Error())
			}
			if !ok {
				continue
			}
		}
		rec := dbf.Record{Values: make([]dbf.Value, len(idxs))}
		for i, fi := range idxs {
			rec.Values[i] = a.Record.Values[fi]
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
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}

	a.Cursor = savedCursor
	_ = a.LoadCurrent()
	return nil
}

func copyStructure(ctx *context.Context, target string, extended bool) error {
	_ = extended
	return fail(ctx, "COPY STRUCTURE arrives in 012.3")
}
