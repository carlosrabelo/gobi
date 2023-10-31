package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func TestAtSAY(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split(`@ 0, 0 SAY "Hi"`)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Hi") {
		t.Fatalf("SAY output: %q", out.String())
	}
	if ctx.Screen.CellAt(0, 0).Ch != 'H' {
		t.Fatalf("cell %q", ctx.Screen.CellAt(0, 0).Ch)
	}
}

func TestAtGETLimit(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split(`STORE "x" TO V`))
	for i := 0; i < 128; i++ {
		if err := mux.Dispatch(ctx, command.Split("@ 0, 0 GET V")); err != nil {
			t.Fatalf("get %d: %v", i, err)
		}
	}
	if err := mux.Dispatch(ctx, command.Split("@ 0, 0 GET V")); err == nil {
		t.Fatal("expected 129th GET error")
	}
	if len(mux.gets) != 128 {
		t.Fatalf("pending %d", len(mux.gets))
	}
}
