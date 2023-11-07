package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func cmdCOPY(ctx *context.Context, cmd command.Command) error {
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(cmd.Args)), "FILE") {
		return cmdCOPYFILE(ctx, cmd)
	}
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
	mode, delim, fieldList := parseCopyModeAndFields(rest)
	switch mode {
	case "STRUCTURE":
		return copyStructure(ctx, target, false)
	case "STRUCTURE EXTENDED":
		return copyStructure(ctx, target, true)
	case "SDF":
		return copyText(ctx, cmd, target, fieldList, true, 0)
	case "DELIMITED":
		return copyText(ctx, cmd, target, fieldList, false, delim)
	default:
		return copyDBF(ctx, cmd, target, fieldList, mode)
	}
}

func parseCopyModeAndFields(tokens []string) (mode string, delim byte, fields []string) {
	delim = '"'
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
			if i+2 < len(tokens) && strings.EqualFold(tokens[i+1], "WITH") {
				w := tokens[i+2]
				if len(w) > 0 {
					delim = w[0]
				}
			}
		case "STRUCTURE":
			mode = "STRUCTURE"
			if i+1 < len(tokens) && strings.EqualFold(tokens[i+1], "EXTENDED") {
				mode = "STRUCTURE EXTENDED"
			}
		}
	}
	return mode, delim, fields
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

func copyText(ctx *context.Context, cmd command.Command, target string, fieldNames []string, sdf bool, delim byte) error {
	a := ctx.Active()
	savedCursor := a.Cursor
	path, err := resolveOutPath(ctx, target, "")
	if err != nil {
		return err
	}
	if fileExists(path) && ctx.Config.Safety {
		if _, err := confirmSafety(ctx, path+" already exists. Overwrite it? (Y/N)"); err != nil {
			return err
		}
	}
	srcFields := a.Table.Header.Fields
	idxs, outFields, err := selectFields(srcFields, fieldNames)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	forExpr, err := parseOptionalLogic(cmd.Clauses["FOR"])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	env := newEnv(ctx)
	f, err := os.Create(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	defer f.Close()

	count := a.RecCount()
	for n := 1; n <= count; n++ {
		a.Cursor = n
		if err := a.LoadCurrent(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if a.Record.Deleted {
			continue
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
	a.Cursor = savedCursor
	_ = a.LoadCurrent()
	return nil
}

func resolveOutPath(ctx *context.Context, name, forceExt string) (string, error) {
	pathName := name
	if forceExt != "" {
		if filepath.Ext(pathName) == "" {
			pathName += forceExt
		}
	}
	path, err := ctx.Resolve(pathName)
	if err != nil {
		return "", fail(ctx, "%s", err.Error())
	}
	return path, nil
}

func selectFields(src []dbf.Field, names []string) (idxs []int, out []dbf.Field, err error) {
	if len(names) == 0 {
		out = src
		for i := range src {
			idxs = append(idxs, i)
		}
		return idxs, out, nil
	}
	for _, name := range names {
		fi := -1
		for i, f := range src {
			if f.Name == name {
				fi = i
				break
			}
		}
		if fi < 0 {
			return nil, nil, errString("unknown field: " + name)
		}
		out = append(out, src[fi])
		idxs = append(idxs, fi)
	}
	return idxs, out, nil
}

func encodeSDF(fields []dbf.Field, rec dbf.Record, idxs []int) []byte {
	var b []byte
	for i, fi := range idxs {
		f := fields[i]
		raw, _ := dbf.EncodeRecord([]dbf.Field{f}, dbf.Record{Values: []dbf.Value{rec.Values[fi]}})
		// EncodeRecord includes deletion mark; skip it.
		b = append(b, raw[1:]...)
	}
	return b
}

func encodeDelimited(fields []dbf.Field, rec dbf.Record, idxs []int, delim byte) string {
	parts := make([]string, len(idxs))
	for i, fi := range idxs {
		f := fields[i]
		v := rec.Values[fi]
		s := fieldString(f, v)
		if f.Type == 'C' || f.Type == 'D' || f.Type == 'M' {
			parts[i] = string(delim) + s + string(delim)
		} else {
			parts[i] = s
		}
	}
	return strings.Join(parts, ",")
}

func copyStructure(ctx *context.Context, target string, extended bool) error {
	a := ctx.Active()
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
	now := time.Now()
	y, m, d := byte(now.Year()-1900), byte(now.Month()), byte(now.Day())
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gobi-copy-*")
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	tmpName := tmp.Name()
	var fields []dbf.Field
	if extended {
		fields = []dbf.Field{
			{Name: "FIELD_NAME", Type: 'C', Length: 10},
			{Name: "FIELD_TYPE", Type: 'C', Length: 1},
			{Name: "FIELD_LEN", Type: 'N', Length: 3, Decimals: 0},
			{Name: "FIELD_DEC", Type: 'N', Length: 3, Decimals: 0},
		}
	} else {
		fields = a.Table.Header.Fields
	}
	tab, err := dbf.Create(tmp, fields, y, m, d)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	if extended {
		for _, f := range a.Table.Header.Fields {
			rec := dbf.Record{Values: []dbf.Value{
				{Str: f.Name},
				{Str: string(f.Type)},
				{Num: float64(f.Length)},
				{Num: float64(f.Decimals)},
			}}
			if err := tab.Append(rec); err != nil {
				_ = tmp.Close()
				_ = os.Remove(tmpName)
				return fail(ctx, "%s", err.Error())
			}
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
	return nil
}

func cmdCREATEFROM(ctx *context.Context, name, from string) error {
	fromPath := from
	if filepath.Ext(fromPath) == "" {
		fromPath += ".dbf"
	}
	fp, err := ctx.Resolve(fromPath)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	f, err := os.OpenFile(fp, os.O_RDONLY, 0)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	defer f.Close()
	src, err := dbf.Open(f)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	var fields []dbf.Field
	for n := 1; n <= int(src.Header.RecordCount); n++ {
		rec, err := src.Read(n)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if len(rec.Values) < 4 {
			return fail(ctx, "invalid STRUCTURE EXTENDED")
		}
		typ := byte('C')
		if len(rec.Values[1].Str) > 0 {
			typ = rec.Values[1].Str[0]
		}
		fields = append(fields, dbf.Field{
			Name:     strings.ToUpper(strings.TrimSpace(rec.Values[0].Str)),
			Type:     typ,
			Length:   byte(rec.Values[2].Num),
			Decimals: byte(rec.Values[3].Num),
		})
	}
	// Reuse CREATE write path without prompts.
	pathName := name
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
	now := time.Now()
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gobi-create-*")
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
	_ = tab.Close()
	_ = tmp.Close()
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	return nil
}
