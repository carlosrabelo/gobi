package repl

import (
	"os"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/ndx"
)

func cmdREINDEX(ctx *context.Context, _ command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	saved := a.Cursor
	for i, path := range a.NDX {
		if path == "" {
			continue
		}
		if err := rebuildIndexFile(ctx, path); err != nil {
			return fail(ctx, "%s: %s", path, err.Error())
		}
		a.NDX[i] = path
	}
	if saved >= 1 && saved <= a.RecCount() {
		a.Cursor = saved
	} else if a.RecCount() > 0 {
		a.Cursor = a.RecCount()
	} else {
		a.Cursor = 1
	}
	_ = a.LoadCurrent()
	return nil
}

func rebuildIndexFile(ctx *context.Context, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	idx, err := ndx.Open(f)
	if err != nil {
		f.Close()
		return err
	}
	exprSrc := idx.Header().Expr
	unique := idx.Header().Unique
	f.Close()

	tree, errs := expr.Parse(exprSrc)
	if len(errs) > 0 || tree == nil {
		msg := "invalid stored expression"
		if len(errs) > 0 {
			msg = errs[0]
		}
		return errString(msg)
	}
	keys, typ, keyLen, err := collectIndexKeys(ctx, tree)
	if err != nil {
		return err
	}
	if keyLen < 1 {
		keyLen = 1
	}
	tmp := path + ".tmp"
	_ = os.Remove(tmp)
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	nidx, err := ndx.Create(out, keyLen, typ, unique, exprSrc)
	if err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return err
	}
	for _, k := range keys {
		if err := nidx.Insert(padKey(k.Key, keyLen, typ), k.Rec); err != nil {
			out.Close()
			_ = os.Remove(tmp)
			return err
		}
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func appendIndexKeys(ctx *context.Context, recNo int) error {
	a := ctx.Active()
	for _, path := range a.NDX {
		if path == "" {
			continue
		}
		if err := insertOneIndexKey(ctx, path, recNo); err != nil {
			return fail(ctx, "%s: %s", path, err.Error())
		}
	}
	return nil
}

func insertOneIndexKey(ctx *context.Context, path string, recNo int) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	idx, err := ndx.Open(f)
	if err != nil {
		return err
	}
	tree, errs := expr.Parse(idx.Header().Expr)
	if len(errs) > 0 || tree == nil {
		return errString("invalid index expression")
	}
	a := ctx.Active()
	saved := a.Cursor
	a.Cursor = recNo
	if err := a.LoadCurrent(); err != nil {
		a.Cursor = saved
		_ = a.LoadCurrent()
		return err
	}
	v, err := expr.Eval(tree, newEnv(ctx))
	a.Cursor = saved
	_ = a.LoadCurrent()
	if err != nil {
		return err
	}
	k, _, err := valueToKey(v)
	if err != nil {
		return err
	}
	h := idx.Header()
	return idx.Insert(padKey(k, int(h.KeyLen), h.Type), uint32(recNo))
}

func syncAllIndexes(ctx *context.Context) error {
	a := ctx.Active()
	for _, path := range a.NDX {
		if path == "" {
			continue
		}
		if err := rebuildIndexFile(ctx, path); err != nil {
			return fail(ctx, "%s: %s", path, err.Error())
		}
	}
	return nil
}
