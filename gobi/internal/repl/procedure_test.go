package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func TestProceduresAcceptance(t *testing.T) {
	dir := t.TempDir()
	prg := "" +
		"DO ADD WITH 3\n" +
		"DO NEST\n" +
		"RETURN\n" +
		"PROCEDURE ADD\n" +
		"PARAMETERS X\n" +
		"PUBLIC G\n" +
		"G = X\n" +
		"PRIVATE P\n" +
		"P = 9\n" +
		"RETURN\n" +
		"PROCEDURE NEST\n" +
		"DO INNER\n" +
		"RETURN\n" +
		"PROCEDURE INNER\n" +
		"RETURN TO MASTER\n"
	if err := os.WriteFile(filepath.Join(dir, "t.prg"), []byte(prg), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("DO t")); err != nil {
		t.Fatal(err)
	}
	g, ok := ctx.Symbols.Get("G")
	if !ok || g.Num != 3 {
		t.Fatalf("PUBLIC G=%v ok=%v", g, ok)
	}
	if _, ok := ctx.Symbols.Get("P"); ok {
		t.Fatal("PRIVATE P should not survive")
	}
	if mux.prog != nil && len(mux.prog.Stack) != 0 {
		t.Fatalf("stack after TO MASTER: %d", len(mux.prog.Stack))
	}
}

func TestSETPROCEDURE(t *testing.T) {
	dir := t.TempDir()
	lib := "PROCEDURE HI\nX = 1\nRETURN\n"
	if err := os.WriteFile(filepath.Join(dir, "lib.prg"), []byte(lib), 0o644); err != nil {
		t.Fatal(err)
	}
	main := "DO HI\n"
	if err := os.WriteFile(filepath.Join(dir, "main.prg"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("SET PROCEDURE TO lib")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("DO main")); err != nil {
		t.Fatal(err)
	}
	v, ok := ctx.Symbols.Get("X")
	if !ok || v.Kind != expr.KindNumber || v.Num != 1 {
		t.Fatalf("X=%v", v)
	}
	if err := mux.Dispatch(ctx, command.Split("CLOSE PROCEDURE")); err != nil {
		t.Fatal(err)
	}
}
