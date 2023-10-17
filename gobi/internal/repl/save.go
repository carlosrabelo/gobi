package repl

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/internal/symbols"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/mem"
)

func cmdSAVE(ctx *context.Context, cmd command.Command) error {
	to := strings.TrimSpace(cmd.Clauses["TO"])
	if to == "" {
		return fail(ctx, "SAVE requires TO")
	}
	name, pattern, like, err := parseSaveTO(to)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	path, err := memPath(ctx, name)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if fileExists(path) && ctx.Config.Safety {
		if _, err := confirmSafety(ctx, path+" already exists. Overwrite it? (Y/N)"); err != nil {
			return err
		}
	}
	reg := ctx.Symbols
	if pattern != "" {
		reg = ctx.Symbols.Filter(pattern, like)
	}
	entries := make([]mem.Entry, 0, reg.Len())
	for _, n := range reg.Names() {
		v, _ := reg.Get(n)
		entries = append(entries, mem.Entry{Name: n, Value: v})
	}
	if err := mem.WriteFile(path, entries); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return nil
}

func cmdRESTORE(ctx *context.Context, cmd command.Command) error {
	from := strings.TrimSpace(cmd.Clauses["FROM"])
	if from == "" {
		return fail(ctx, "RESTORE requires FROM")
	}
	path, err := memPath(ctx, from)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	additive := strings.Contains(strings.ToUpper(cmd.Args), "ADDITIVE")
	entries, err := mem.ReadFile(path)
	if err != nil {
		// Leave registry unchanged on read/format errors (including type D).
		return fail(ctx, "%s", err.Error())
	}
	loaded := symbols.New()
	for _, e := range entries {
		if err := loaded.Set(e.Name, e.Value); err != nil {
			return fail(ctx, "%s", err.Error())
		}
	}
	if additive {
		if err := ctx.Symbols.Merge(loaded); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		return nil
	}
	ctx.Symbols.Replace(loaded)
	return nil
}

func parseSaveTO(to string) (name, pattern string, like bool, err error) {
	fields := strings.Fields(to)
	if len(fields) == 0 {
		return "", "", false, errString("SAVE requires TO name")
	}
	name = fields[0]
	if len(fields) == 1 {
		return name, "", false, nil
	}
	if len(fields) >= 3 && strings.EqualFold(fields[1], "ALL") {
		switch strings.ToUpper(fields[2]) {
		case "LIKE":
			if len(fields) < 4 {
				return "", "", false, errString("SAVE ALL LIKE requires a pattern")
			}
			return name, fields[3], true, nil
		case "EXCEPT":
			if len(fields) < 4 {
				return "", "", false, errString("SAVE ALL EXCEPT requires a pattern")
			}
			return name, fields[3], false, nil
		}
	}
	return "", "", false, errString("invalid SAVE TO")
}

func memPath(ctx *context.Context, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errString("missing .mem name")
	}
	if filepath.Ext(name) == "" {
		name += ".mem"
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
