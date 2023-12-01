package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func TestOnErrorRetryAndHelp(t *testing.T) {
	dir := t.TempDir()
	prg := "" +
		"ON ERROR X = ERROR()\n" +
		"NOSUCH\n" +
		"STORE MESSAGE() TO M\n" +
		"RETURN\n"
	if err := os.WriteFile(filepath.Join(dir, "e.prg"), []byte(prg), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("DO e")); err != nil {
		t.Fatal(err)
	}
	v, ok := ctx.Symbols.Get("X")
	if !ok || v.Num != 36 {
		t.Fatalf("ERROR code %v ok=%v", v, ok)
	}
	m, ok := ctx.Symbols.Get("M")
	if !ok || !strings.Contains(m.Str, "Unrecognized") {
		t.Fatalf("MESSAGE %v", m)
	}

	// RETRY once then clear ON ERROR inside handler
	prg2 := "" +
		"N = 0\n" +
		"ON ERROR DO FIX\n" +
		"NOSUCH\n" +
		"RETURN\n" +
		"PROCEDURE FIX\n" +
		"N = N+1\n" +
		"ON ERROR\n" +
		"RETRY\n" +
		"RETURN\n"
	if err := os.WriteFile(filepath.Join(dir, "r.prg"), []byte(prg2), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx = testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}
	mux = NewMux()
	if err := mux.Dispatch(ctx, command.Split("DO r")); err != nil {
		// after RETRY, NOSUCH fails again without handler — expected
		if !strings.Contains(err.Error(), "Unrecognized") {
			t.Fatal(err)
		}
	}
	v, _ = ctx.Symbols.Get("N")
	if v.Num != 1 {
		t.Fatalf("N %v want 1", v)
	}

	// SET HELP OFF skips question
	ctx = testCtx(t, dir)
	out := &bytes.Buffer{}
	ctx.Stdout = out
	ctx.Stderr = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("N\n")
	mux = NewMux()
	_ = mux.Dispatch(ctx, command.Split("SET HELP OFF"))
	_ = mux.Dispatch(ctx, command.Split("NOSUCH"))
	if strings.Contains(out.String(), "help") {
		t.Fatal("HELP question should be skipped")
	}
}
