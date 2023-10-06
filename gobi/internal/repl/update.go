package repl

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func cmdUPDATE(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	key, from, mode, fields, err := parseUpdateArgs(cmd)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	pathName := from
	if filepath.Ext(pathName) == "" {
		pathName += ".dbf"
	}
	path, err := ctx.Resolve(pathName)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	defer f.Close()
	src, err := dbf.Open(f)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	tKey := fieldIndex(a, key)
	if tKey < 0 {
		return fail(ctx, "unknown key field: %s", key)
	}
	sKey := -1
	for i, f := range src.Header.Fields {
		if f.Name == strings.ToUpper(key) {
			sKey = i
			break
		}
	}
	if sKey < 0 {
		return fail(ctx, "unknown key field in source: %s", key)
	}
	fieldPairs := make([][2]int, 0, len(fields))
	for _, name := range fields {
		ti := fieldIndex(a, name)
		if ti < 0 {
			return fail(ctx, "unknown field: %s", name)
		}
		si := -1
		for i, f := range src.Header.Fields {
			if f.Name == strings.ToUpper(name) {
				si = i
				break
			}
		}
		if si < 0 {
			return fail(ctx, "unknown source field: %s", name)
		}
		fieldPairs = append(fieldPairs, [2]int{ti, si})
	}

	// Merge walk assuming both sorted by key (string compare of fieldString).
	ti, si := 1, 1
	tCount, sCount := a.RecCount(), int(src.Header.RecordCount)
	for ti <= tCount && si <= sCount {
		trec, err := a.Table.Read(ti)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		srec, err := src.Read(si)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		tk := fieldString(a.Table.Header.Fields[tKey], trec.Values[tKey])
		sk := fieldString(src.Header.Fields[sKey], srec.Values[sKey])
		cmp := strings.Compare(strings.ToUpper(tk), strings.ToUpper(sk))
		switch {
		case cmp < 0:
			ti++
		case cmp > 0:
			si++
		default:
			for _, p := range fieldPairs {
				if mode == "ADD" {
					trec.Values[p[0]].Num += srec.Values[p[1]].Num
				} else {
					trec.Values[p[0]] = srec.Values[p[1]]
				}
			}
			if err := a.Table.Write(ti, trec); err != nil {
				return fail(ctx, "%s", err.Error())
			}
			ti++
			si++
		}
	}
	return a.LoadCurrent()
}

func parseUpdateArgs(cmd command.Command) (key, from, mode string, fields []string, err error) {
	from = strings.TrimSpace(cmd.Clauses["FROM"])
	if from == "" {
		return "", "", "", nil, errString("UPDATE requires FROM")
	}
	args := strings.TrimSpace(cmd.Args)
	up := strings.ToUpper(args)
	if !strings.HasPrefix(up, "ON ") {
		return "", "", "", nil, errString("UPDATE requires ON")
	}
	rest := strings.TrimSpace(args[3:])
	tokens := strings.Fields(rest)
	if len(tokens) < 2 {
		return "", "", "", nil, errString("UPDATE ON key REPLACE|ADD fields")
	}
	key = tokens[0]
	modeTok := strings.ToUpper(tokens[1])
	if modeTok != "REPLACE" && modeTok != "ADD" {
		return "", "", "", nil, errString("UPDATE requires REPLACE or ADD")
	}
	mode = modeTok
	joined := strings.Join(tokens[2:], " ")
	for _, p := range strings.Split(joined, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			fields = append(fields, strings.ToUpper(p))
		}
	}
	if len(fields) == 0 {
		return "", "", "", nil, errString("UPDATE requires at least one field")
	}
	return key, from, mode, fields, nil
}
