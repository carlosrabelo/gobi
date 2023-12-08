package repl

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
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

func TestREADStoresField(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN "}}},
	})
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	// Type "BOB" then Enter on last (only) field.
	ctx.Stdin = strings.NewReader("\x7f\x7f\x7f\x7fBOB\r")
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("@ 0, 0 GET NAME")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("READ")); err != nil {
		t.Fatal(err)
	}
	a := ctx.Active()
	if err := a.LoadCurrent(); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(a.Record.Values[0].Str)
	if got != "BOB" {
		t.Fatalf("field after READ: %q", a.Record.Values[0].Str)
	}
	if len(mux.gets) != 0 {
		t.Fatal("gets not cleared")
	}
}

func TestSETConfirmWaitsOnFullField(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 3},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "   "}}},
	})
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))

	// CONFIRM OFF (default): filling the only GET ends READ.
	ctx.Stdin = strings.NewReader("ABC")
	_ = mux.Dispatch(ctx, command.Split("@ 0, 0 GET NAME"))
	if err := mux.Dispatch(ctx, command.Split("READ")); err != nil {
		t.Fatal(err)
	}
	_ = ctx.Active().LoadCurrent()
	if strings.TrimSpace(ctx.Active().Record.Values[0].Str) != "ABC" {
		t.Fatalf("CONFIRM OFF: %q", ctx.Active().Record.Values[0].Str)
	}

	// CONFIRM ON: full field stays until Enter.
	_ = mux.Dispatch(ctx, command.Split("SET CONFIRM ON"))
	if !ctx.Config.Confirm {
		t.Fatal("CONFIRM still off")
	}
	_ = mux.Dispatch(ctx, command.Split("REPLACE NAME WITH [   ]"))
	ctx.Stdin = strings.NewReader("XYZ\r")
	_ = mux.Dispatch(ctx, command.Split("@ 0, 0 GET NAME"))
	if err := mux.Dispatch(ctx, command.Split("READ")); err != nil {
		t.Fatal(err)
	}
	_ = ctx.Active().LoadCurrent()
	if strings.TrimSpace(ctx.Active().Record.Values[0].Str) != "XYZ" {
		t.Fatalf("CONFIRM ON: %q", ctx.Active().Record.Values[0].Str)
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
