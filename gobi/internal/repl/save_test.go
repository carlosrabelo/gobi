package repl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func TestSAVEAndRESTORE(t *testing.T) {
	dir := t.TempDir()
	ctx := testCtx(t, dir)
	ctx.Config.Safety = false
	mux := NewMux()

	if err := mux.Dispatch(ctx, command.Split("STORE 2 TO X")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split(`STORE "hi" TO MSG`)); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("SAVE TO vars")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "vars.mem")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}

	ctx.Symbols.ReleaseAll()
	if err := mux.Dispatch(ctx, command.Split("RESTORE FROM vars")); err != nil {
		t.Fatal(err)
	}
	v, ok := ctx.Symbols.Get("X")
	if !ok || v.Num != 2 {
		t.Fatalf("X after RESTORE: %v %v", ok, v)
	}
	v, ok = ctx.Symbols.Get("MSG")
	if !ok || v.Str != "hi" {
		t.Fatalf("MSG: %v %v", ok, v)
	}

	_ = ctx.Symbols.Set("KEEP", expr.NumberVal(9))
	if err := mux.Dispatch(ctx, command.Split("STORE 5 TO X")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("SAVE TO partial ALL LIKE M*")); err != nil {
		t.Fatal(err)
	}
	ctx.Symbols.ReleaseAll()
	_ = ctx.Symbols.Set("KEEP", expr.NumberVal(1))
	if err := mux.Dispatch(ctx, command.Split("RESTORE FROM partial ADDITIVE")); err != nil {
		t.Fatal(err)
	}
	if _, ok := ctx.Symbols.Get("KEEP"); !ok {
		t.Fatal("ADDITIVE should keep KEEP")
	}
	if _, ok := ctx.Symbols.Get("MSG"); !ok {
		t.Fatal("MSG from partial")
	}
	if _, ok := ctx.Symbols.Get("X"); ok {
		t.Fatal("X should not be in LIKE M*")
	}
}
