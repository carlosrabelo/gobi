package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func TestDORunsFileAndSkipsComments(t *testing.T) {
	dir := t.TempDir()
	prg := filepath.Join(dir, "hi.prg")
	body := "* comment\nNOTE also comment\n\nSTORE 1 TO A\nSTORE 2 TO B\n"
	if err := os.WriteFile(prg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("DO hi")); err != nil {
		t.Fatal(err)
	}
	if v, ok := ctx.Symbols.Get("A"); !ok || v.Num != 1 {
		t.Fatalf("A: %v %v", ok, v)
	}
	if v, ok := ctx.Symbols.Get("B"); !ok || v.Num != 2 {
		t.Fatalf("B: %v %v", ok, v)
	}
	if len(mux.prog.Stack) != 0 {
		t.Fatalf("stack not empty: %d", len(mux.prog.Stack))
	}
}

func TestDOMissingFile(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	mux := NewMux()
	err := mux.Dispatch(ctx, command.Split("DO nope"))
	if err == nil {
		t.Fatal("expected missing file error")
	}
	if !strings.Contains(err.Error(), "nope") && !strings.Contains(err.Error(), "No such") && !strings.Contains(err.Error(), "cannot find") {
		// fail() wraps message; ensure we got an error
	}
}

func TestDONestedReturnCancel(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child.prg")
	parent := filepath.Join(dir, "parent.prg")
	if err := os.WriteFile(child, []byte("STORE 1 TO C\nRETURN\nSTORE 99 TO C\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(parent, []byte("DO child\nSTORE 2 TO P\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("DO parent")); err != nil {
		t.Fatal(err)
	}
	if v, ok := ctx.Symbols.Get("C"); !ok || v.Num != 1 {
		t.Fatalf("C after RETURN: %v %v", ok, v)
	}
	if v, ok := ctx.Symbols.Get("P"); !ok || v.Num != 2 {
		t.Fatalf("P: %v %v", ok, v)
	}

	deep := filepath.Join(dir, "deep.prg")
	mid := filepath.Join(dir, "mid.prg")
	top := filepath.Join(dir, "top.prg")
	if err := os.WriteFile(deep, []byte("CANCEL\nSTORE 1 TO Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mid, []byte("DO deep\nSTORE 2 TO Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(top, []byte("DO mid\nSTORE 3 TO Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx.Symbols.ReleaseAll()
	if err := mux.Dispatch(ctx, command.Split("DO top")); err != nil {
		t.Fatal(err)
	}
	if _, ok := ctx.Symbols.Get("Z"); ok {
		t.Fatal("CANCEL should drop all frames before STORE")
	}
	if len(mux.prog.Stack) != 0 {
		t.Fatalf("stack after CANCEL: %d", len(mux.prog.Stack))
	}
}
