package repl

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func TestWalkPRG(t *testing.T) {
	dir := t.TempDir()
	copyFile(t, filepath.Join("testdata", "walk.prg"), filepath.Join(dir, "walk.prg"))
	copyFile(t, filepath.Join("..", "..", "pkg", "dbf", "testdata", "sample.dbf"), filepath.Join(dir, "sample.dbf"))

	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("DO walk")); err != nil {
		t.Fatal(err)
	}
	v, ok := ctx.Symbols.Get("N")
	if !ok || v.Kind != expr.KindNumber || v.Num != 1 {
		t.Fatalf("count N=%v ok=%v", v, ok)
	}
	// back at prompt: stack empty
	if mux.prog != nil && len(mux.prog.Stack) != 0 {
		t.Fatalf("stack len %d", len(mux.prog.Stack))
	}
}

func TestSampleNDXSeekViaMux(t *testing.T) {
	dir := t.TempDir()
	copyFile(t, filepath.Join("..", "..", "pkg", "dbf", "testdata", "sample.dbf"), filepath.Join(dir, "sample.dbf"))
	copyFile(t, filepath.Join("..", "..", "pkg", "ndx", "testdata", "sample.ndx"), filepath.Join(dir, "sample.ndx"))
	ctx := testCtx(t, dir)
	ctx.Stdout = io.Discard
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE sample INDEX sample")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split(`SEEK "ALPHA"`)); err != nil {
		t.Fatal(err)
	}
	a := ctx.Active()
	if a.Cursor != 1 || !a.Found {
		t.Fatalf("cursor=%d found=%v", a.Cursor, a.Found)
	}
}

func TestSampleMEMRestore(t *testing.T) {
	dir := t.TempDir()
	copyFile(t, filepath.Join("..", "..", "pkg", "mem", "testdata", "sample.mem"), filepath.Join(dir, "sample.mem"))
	ctx := testCtx(t, dir)
	ctx.Stdout = io.Discard
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("RESTORE FROM sample")); err != nil {
		t.Fatal(err)
	}
	v, ok := ctx.Symbols.Get("MSG")
	if !ok || v.Str != "hello" {
		t.Fatalf("MSG %v %v", v, ok)
	}
	v, ok = ctx.Symbols.Get("N")
	if !ok || v.Num != 42 {
		t.Fatalf("N %v %v", v, ok)
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	in, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, in, 0o644); err != nil {
		t.Fatal(err)
	}
}
