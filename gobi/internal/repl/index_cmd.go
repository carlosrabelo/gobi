package repl

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/ndx"
)

func cmdINDEX(ctx *context.Context, cmd command.Command) error {
	args := strings.TrimSpace(cmd.Args)
	up := strings.ToUpper(args)
	if !strings.HasPrefix(up, "ON ") {
		return fail(ctx, "INDEX requires ON")
	}
	rest := strings.TrimSpace(args[3:])
	to := strings.TrimSpace(cmd.Clauses["TO"])
	unique := false
	// UNIQUE may be in Args after ON expr, or as trailing token in TO clause.
	if i := strings.LastIndex(strings.ToUpper(rest), " UNIQUE"); i >= 0 && strings.TrimSpace(rest[i+7:]) == "" {
		unique = true
		rest = strings.TrimSpace(rest[:i])
	}
	if strings.Contains(strings.ToUpper(to), " UNIQUE") {
		unique = true
		to = strings.TrimSpace(strings.Split(to, " ")[0])
	}
	// Parse ON expr TO name — TO may have been taken as clause; expr is rest before TO was stripped.
	// When Split sees TO, expr is in Args as "ON <expr>" and TO is clause.
	exprSrc := rest
	if to == "" {
		return fail(ctx, "INDEX ON requires TO")
	}
	toFields := strings.Fields(to)
	name := toFields[0]
	for _, f := range toFields[1:] {
		if strings.EqualFold(f, "UNIQUE") {
			unique = true
		}
	}

	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	path, err := ndxPath(ctx, name)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if fileExists(path) && ctx.Config.Safety {
		if _, err := confirmSafety(ctx, path+" already exists. Overwrite it? (Y/N)"); err != nil {
			return err
		}
	}

	tree, errs := expr.Parse(exprSrc)
	if len(errs) > 0 || tree == nil {
		msg := "invalid expression"
		if len(errs) > 0 {
			msg = errs[0]
		}
		return fail(ctx, "%s", msg)
	}

	keys, typ, keyLen, err := collectIndexKeys(ctx, tree)
	if err != nil {
		return err
	}
	if keyLen < 1 {
		keyLen = 1
	}
	if keyLen > ndx.MaxKeyLen {
		keyLen = ndx.MaxKeyLen
	}

	tmp := path + ".tmp"
	_ = os.Remove(tmp)
	f, err := os.Create(tmp)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	idx, err := ndx.Create(f, keyLen, typ, unique, exprSrc)
	if err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return fail(ctx, "%s", err.Error())
	}
	for _, k := range keys {
		key := padKey(k.Key, keyLen, typ)
		if err := idx.Insert(key, k.Rec); err != nil {
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
	bindIndex(a, path)
	return nil
}

type indexKey struct {
	Key []byte
	Rec uint32
}

func collectIndexKeys(ctx *context.Context, tree expr.Expression) ([]indexKey, ndx.KeyType, int, error) {
	a := ctx.Active()
	saved := a.Cursor
	defer func() {
		a.Cursor = saved
		_ = a.LoadCurrent()
	}()
	env := newEnv(ctx)
	var keys []indexKey
	var typ ndx.KeyType = ndx.KeyChar
	keyLen := 1
	first := true
	count := a.RecCount()
	for n := 1; n <= count; n++ {
		a.Cursor = n
		if err := a.LoadCurrent(); err != nil {
			return nil, 0, 0, fail(ctx, "%s", err.Error())
		}
		if ctx.Config.Deleted && a.Record.Deleted {
			continue
		}
		v, err := expr.Eval(tree, env)
		if err != nil {
			return nil, 0, 0, fail(ctx, "%s", err.Error())
		}
		k, t, err := valueToKey(v)
		if err != nil {
			return nil, 0, 0, fail(ctx, "%s", err.Error())
		}
		if first {
			typ = t
			first = false
		} else if t != typ {
			return nil, 0, 0, fail(ctx, "mixed key types in INDEX")
		}
		if len(k) > keyLen {
			keyLen = len(k)
		}
		keys = append(keys, indexKey{Key: k, Rec: uint32(n)})
	}
	return keys, typ, keyLen, nil
}

func valueToKey(v expr.Value) ([]byte, ndx.KeyType, error) {
	switch v.Kind {
	case expr.KindNumber:
		return ndx.EncodeKeyNum(v.Num), ndx.KeyNum, nil
	case expr.KindLogical:
		if v.Log {
			return ndx.EncodeKeyChar("T", 1), ndx.KeyChar, nil
		}
		return ndx.EncodeKeyChar("F", 1), ndx.KeyChar, nil
	default:
		s := v.Str
		if len(s) > ndx.MaxKeyLen {
			s = s[:ndx.MaxKeyLen]
		}
		if len(s) == 0 {
			s = " "
		}
		return []byte(s), ndx.KeyChar, nil
	}
}

func padKey(k []byte, keyLen int, typ ndx.KeyType) []byte {
	if typ == ndx.KeyNum {
		return k
	}
	return ndx.EncodeKeyChar(string(k), keyLen)
}

func ndxPath(ctx *context.Context, name string) (string, error) {
	name = strings.TrimSpace(name)
	if filepath.Ext(name) == "" {
		name += ".ndx"
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

func bindIndex(a *context.Area, path string) {
	for i := 0; i < len(a.NDX); i++ {
		if a.NDX[i] == "" {
			a.NDX[i] = path
			return
		}
	}
	// Replace last slot when full.
	a.NDX[len(a.NDX)-1] = path
}
