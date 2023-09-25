package repl

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestGOTOandSKIP(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "nav.dbf"), []dbf.Field{{Name: "NAME", Type: 'C', Length: 4}}, []dbf.Record{
		{Values: []dbf.Value{{Str: "R1"}}},
		{Values: []dbf.Value{{Str: "R2"}}},
		{Values: []dbf.Value{{Str: "R3"}}},
	})

	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE nav")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Cursor != 1 || !ctx.Active().Landed {
		t.Fatalf("after USE: cursor=%d landed=%v", ctx.Active().Cursor, ctx.Active().Landed)
	}

	if err := mux.Dispatch(ctx, command.Split("GOTO 2")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Cursor != 2 || ctx.Active().Record.Values[0].Str != "R2" {
		t.Fatalf("GOTO 2: %+v", ctx.Active().Record)
	}

	prev := ctx.Active().Cursor
	var errBuf bytes.Buffer
	ctx.Stderr = &errBuf
	if err := mux.Dispatch(ctx, command.Split("GO TO 99")); err == nil {
		t.Fatal("expected out of range")
	}
	if !strings.Contains(errBuf.String(), "*** ") {
		t.Fatalf("stderr %q", errBuf.String())
	}
	if ctx.Active().Cursor != prev {
		t.Fatalf("cursor moved on error to %d", ctx.Active().Cursor)
	}

	if err := mux.Dispatch(ctx, command.Split("GO TOP")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("SKIP -1")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Cursor != 1 || !ctx.Active().BOF {
		t.Fatalf("SKIP -1 clamp: cursor=%d bof=%v", ctx.Active().Cursor, ctx.Active().BOF)
	}

	if err := mux.Dispatch(ctx, command.Split("GO BOTTOM")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Cursor != 3 || ctx.Active().BOF {
		t.Fatalf("BOTTOM: cursor=%d bof=%v", ctx.Active().Cursor, ctx.Active().BOF)
	}
	if err := mux.Dispatch(ctx, command.Split("SKIP")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Cursor != 4 || ctx.Active().Landed {
		t.Fatalf("SKIP past end: cursor=%d landed=%v", ctx.Active().Cursor, ctx.Active().Landed)
	}
}
