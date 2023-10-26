package repl

import (
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/ndx"
)

func cmdFIND(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	path := controllingIndex(a)
	if path == "" {
		return fail(ctx, "no controlling index")
	}
	text := strings.TrimSpace(cmd.Args)
	return seekKey(ctx, path, []byte(text), true)
}

func cmdSEEK(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	path := controllingIndex(a)
	if path == "" {
		return fail(ctx, "no controlling index")
	}
	src := strings.TrimSpace(cmd.Args)
	tree, errs := expr.Parse(src)
	if len(errs) > 0 || tree == nil {
		msg := "invalid expression"
		if len(errs) > 0 {
			msg = errs[0]
		}
		return fail(ctx, "%s", msg)
	}
	v, err := expr.Eval(tree, newEnv(ctx))
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	k, _, err := valueToKey(v)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return seekKey(ctx, path, k, false)
}

func controllingIndex(a *context.Area) string {
	return a.NDX[0]
}

func seekKey(ctx *context.Context, path string, key []byte, asChar bool) error {
	a := ctx.Active()
	f, err := os.Open(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	defer f.Close()
	idx, err := ndx.Open(f)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	h := idx.Header()
	padded := key
	if h.Type == ndx.KeyNum && !asChar {
		if len(key) != 8 {
			// valueToKey already encoded numeric
			padded = key
		}
	} else if h.Type == ndx.KeyChar || asChar {
		padded = ndx.EncodeKeyChar(string(key), int(h.KeyLen))
	} else if len(key) != int(h.KeyLen) {
		padded = padKey(key, int(h.KeyLen), h.Type)
	}
	if h.Type == ndx.KeyNum && asChar {
		// FIND on numeric index: coerce text to number key
		tree, errs := expr.Parse(string(key))
		if len(errs) == 0 && tree != nil {
			v, err := expr.Eval(tree, newEnv(ctx))
			if err == nil && v.Kind == expr.KindNumber {
				padded = ndx.EncodeKeyNum(v.Num)
			}
		}
	}
	rec, ok, err := idx.Search(padded)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if !ok {
		a.Found = false
		a.Cursor = a.RecCount() + 1
		a.BOF = false
		_ = a.LoadCurrent()
		return nil
	}
	a.Found = true
	a.Cursor = int(rec)
	a.BOF = false
	return a.LoadCurrent()
}
