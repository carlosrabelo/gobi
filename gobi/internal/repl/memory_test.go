package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func TestSTOREAssignmentDisplayRelease(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()

	if err := mux.Dispatch(ctx, command.Split("STORE 2 TO X")); err != nil {
		t.Fatal(err)
	}
	v, ok := ctx.Symbols.Get("X")
	if !ok || v.Num != 2 {
		t.Fatalf("STORE: %v %v", ok, v)
	}

	if err := mux.Dispatch(ctx, command.Split("X = 3")); err != nil {
		t.Fatal(err)
	}
	v, ok = ctx.Symbols.Get("X")
	if !ok || v.Num != 3 {
		t.Fatalf("assign: %v %v", ok, v)
	}

	if err := mux.Dispatch(ctx, command.Split(`STORE "hi" TO MSG`)); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("DISPLAY MEMORY")); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "MSG") || !strings.Contains(got, "C") || !strings.Contains(got, "hi") {
		t.Fatalf("DISPLAY MEMORY: %q", got)
	}
	if !strings.Contains(got, "X") || !strings.Contains(got, "N") {
		t.Fatalf("DISPLAY MEMORY X: %q", got)
	}
	// Sorted: MSG before X
	mi := strings.Index(got, "MSG")
	xi := strings.Index(got, "X")
	if mi < 0 || xi < 0 || mi > xi {
		t.Fatalf("sort order: %q", got)
	}

	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("LIST MEMORY")); err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 {
		t.Fatal("LIST MEMORY empty")
	}

	if err := mux.Dispatch(ctx, command.Split("RELEASE X")); err != nil {
		t.Fatal(err)
	}
	if _, ok := ctx.Symbols.Get("X"); ok {
		t.Fatal("X still present")
	}
	if err := mux.Dispatch(ctx, command.Split("RELEASE ALL")); err != nil {
		t.Fatal(err)
	}
	if ctx.Symbols.Len() != 0 {
		t.Fatal("RELEASE ALL")
	}
}

func TestRELEASEAllLikeExcept(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("STORE 1 TO N1"))
	_ = mux.Dispatch(ctx, command.Split("STORE 10 TO N10"))
	_ = mux.Dispatch(ctx, command.Split("STORE 0 TO X"))
	if err := mux.Dispatch(ctx, command.Split("RELEASE ALL LIKE N?")); err != nil {
		t.Fatal(err)
	}
	if _, ok := ctx.Symbols.Get("N1"); ok {
		t.Fatal("N1")
	}
	if _, ok := ctx.Symbols.Get("N10"); !ok {
		t.Fatal("N10 should remain")
	}
	if err := mux.Dispatch(ctx, command.Split("RELEASE ALL EXCEPT X")); err != nil {
		t.Fatal(err)
	}
	if _, ok := ctx.Symbols.Get("N10"); ok {
		t.Fatal("N10 released by EXCEPT X")
	}
	if _, ok := ctx.Symbols.Get("X"); !ok {
		t.Fatal("X kept")
	}
}
