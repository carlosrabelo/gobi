package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func TestACCEPTINPUTWAIT(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	ctx.Stdin = strings.NewReader("hello\n2+3\nX")
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split(`ACCEPT "p" TO A`)); err != nil {
		t.Fatal(err)
	}
	if v, _ := ctx.Symbols.Get("A"); v.Str != "hello" {
		t.Fatalf("ACCEPT %q", v.Str)
	}
	if err := mux.Dispatch(ctx, command.Split(`INPUT "n" TO N`)); err != nil {
		t.Fatal(err)
	}
	if v, _ := ctx.Symbols.Get("N"); v.Num != 5 {
		t.Fatalf("INPUT %v", v)
	}
	if err := mux.Dispatch(ctx, command.Split("WAIT TO K")); err != nil {
		t.Fatal(err)
	}
	if v, _ := ctx.Symbols.Get("K"); v.Str != "X" {
		t.Fatalf("WAIT %q", v.Str)
	}
}

func TestTEXTRemarkCase(t *testing.T) {
	dir := t.TempDir()
	prg := filepath.Join(dir, "c.prg")
	body := "" +
		"REMARK hi\n" +
		"NOTE silent\n" +
		"* also silent\n" +
		"TEXT\n" +
		"line1\n" +
		"line2\n" +
		"ENDTEXT\n" +
		"STORE 1 TO X\n" +
		"DO CASE\n" +
		"CASE X = 0\n" +
		"STORE 9 TO Y\n" +
		"CASE X = 1\n" +
		"STORE 2 TO Y\n" +
		"OTHERWISE\n" +
		"STORE 3 TO Y\n" +
		"ENDCASE\n"
	if err := os.WriteFile(prg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("DO c")); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "hi") || !strings.Contains(got, "line1") || !strings.Contains(got, "line2") {
		t.Fatalf("out %q", got)
	}
	if strings.Contains(got, "silent") {
		t.Fatal("NOTE printed")
	}
	if v, _ := ctx.Symbols.Get("Y"); v.Num != 2 {
		t.Fatalf("CASE Y=%v", v)
	}

	bad := filepath.Join(dir, "bad.prg")
	_ = os.WriteFile(bad, []byte("DO CASE\nCASE .T.\nSTORE 1 TO Z\n"), 0o644)
	ctx.Symbols.ReleaseAll()
	if err := mux.Dispatch(ctx, command.Split("DO bad")); err == nil {
		t.Fatal("expected missing ENDCASE")
	}
	if _, ok := ctx.Symbols.Get("Z"); ok {
		t.Fatal("must not run")
	}
}
