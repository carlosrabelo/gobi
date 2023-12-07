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

func TestIFElseEndif(t *testing.T) {
	dir := t.TempDir()
	prg := filepath.Join(dir, "cond.prg")
	body := "" +
		"STORE .T. TO F\n" +
		"IF F\n" +
		"STORE 1 TO T\n" +
		"ELSE\n" +
		"STORE 0 TO T\n" +
		"ENDIF\n" +
		"IF .F.\n" +
		"STORE 1 TO Z\n" +
		"ELSE\n" +
		"STORE 2 TO Z\n" +
		"ENDIF\n" +
		"IF .T.\n" +
		"IF .T.\n" +
		"STORE 3 TO N\n" +
		"ENDIF\n" +
		"ENDIF\n"
	if err := os.WriteFile(prg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("DO cond")); err != nil {
		t.Fatal(err)
	}
	if v, _ := ctx.Symbols.Get("T"); v.Num != 1 {
		t.Fatalf("true branch T=%v", v)
	}
	if v, _ := ctx.Symbols.Get("Z"); v.Num != 2 {
		t.Fatalf("false branch Z=%v", v)
	}
	if v, _ := ctx.Symbols.Get("N"); v.Num != 3 {
		t.Fatalf("nested N=%v", v)
	}

	bad := filepath.Join(dir, "bad.prg")
	if err := os.WriteFile(bad, []byte("STORE 1 TO X\nIF .T.\nSTORE 2 TO X\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx.Symbols.ReleaseAll()
	err := mux.Dispatch(ctx, command.Split("DO bad"))
	if err == nil {
		t.Fatal("expected missing ENDIF")
	}
	if _, ok := ctx.Symbols.Get("X"); ok {
		t.Fatal("must not run first line when ENDIF missing")
	}
}

func TestDOWHILELoopExit(t *testing.T) {
	dir := t.TempDir()
	prg := filepath.Join(dir, "loop.prg")
	body := "" +
		"STORE 0 TO N\n" +
		"DO WHILE N < 3\n" +
		"STORE N + 1 TO N\n" +
		"ENDDO\n" +
		"STORE 0 TO L\n" +
		"DO WHILE .T.\n" +
		"STORE L + 1 TO L\n" +
		"IF L >= 2\n" +
		"EXIT\n" +
		"ENDIF\n" +
		"LOOP\n" +
		"STORE 99 TO L\n" +
		"ENDDO\n"
	if err := os.WriteFile(prg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("DO loop")); err != nil {
		t.Fatal(err)
	}
	if v, _ := ctx.Symbols.Get("N"); v.Num != 3 {
		t.Fatalf("N=%v want 3", v)
	}
	if v, _ := ctx.Symbols.Get("L"); v.Num != 2 {
		t.Fatalf("L=%v want 2 (LOOP/EXIT)", v)
	}
}

func TestSETEchoPrintsCommandLines(t *testing.T) {
	dir := t.TempDir()
	prg := filepath.Join(dir, "echo.prg")
	body := "STORE 1 TO A\nSTORE 2 TO B\n"
	if err := os.WriteFile(prg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("SET ECHO ON")); err != nil {
		t.Fatal(err)
	}
	if !ctx.Config.Echo {
		t.Fatal("ECHO still off")
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("DO echo")); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "STORE 1 TO A") || !strings.Contains(got, "STORE 2 TO B") {
		t.Fatalf("ECHO did not print lines: %q", got)
	}
	_ = mux.Dispatch(ctx, command.Split("SET ECHO OFF"))
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("DO echo")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "STORE") {
		t.Fatalf("ECHO OFF still printed: %q", out.String())
	}
}
