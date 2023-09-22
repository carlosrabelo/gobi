package repl

import (
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestSELECTKeepsOtherCursor(t *testing.T) {
	dir := t.TempDir()
	fields := []dbf.Field{{Name: "NAME", Type: 'C', Length: 4}}
	mustCreateDBF(t, filepath.Join(dir, "one.dbf"), fields, []dbf.Record{
		{Values: []dbf.Value{{Str: "A1"}}},
		{Values: []dbf.Value{{Str: "A2"}}},
	})
	mustCreateDBF(t, filepath.Join(dir, "two.dbf"), fields, []dbf.Record{
		{Values: []dbf.Value{{Str: "B1"}}},
		{Values: []dbf.Value{{Str: "B2"}}},
		{Values: []dbf.Value{{Str: "B3"}}},
	})

	ctx := testCtx(t, dir)
	mux := NewMux()

	if err := mux.Dispatch(ctx, command.Split("USE one")); err != nil {
		t.Fatal(err)
	}
	ctx.Active().Cursor = 2
	if err := ctx.Active().LoadCurrent(); err != nil {
		t.Fatal(err)
	}

	if err := mux.Dispatch(ctx, command.Split("SELECT 2")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("USE two ALIAS TW")); err != nil {
		t.Fatal(err)
	}
	ctx.Active().Cursor = 3
	if err := ctx.Active().LoadCurrent(); err != nil {
		t.Fatal(err)
	}

	if err := mux.Dispatch(ctx, command.Split("SELECT B")); err != nil {
		t.Fatal(err)
	}
	if ctx.ActiveArea != 2 {
		t.Fatalf("letter B: area %d", ctx.ActiveArea)
	}
	if ctx.Active().Cursor != 3 {
		t.Fatalf("area 2 cursor %d", ctx.Active().Cursor)
	}

	if err := mux.Dispatch(ctx, command.Split("SELECT ONE")); err != nil {
		t.Fatal(err)
	}
	if ctx.ActiveArea != 1 {
		t.Fatalf("alias ONE: area %d", ctx.ActiveArea)
	}
	if ctx.Active().Cursor != 2 || ctx.Active().Record.Values[0].Str != "A2" {
		t.Fatalf("area 1 cursor lost: %d %q", ctx.Active().Cursor, ctx.Active().Record.Values[0].Str)
	}

	if err := mux.Dispatch(ctx, command.Split("SELECT TW")); err != nil {
		t.Fatal(err)
	}
	if ctx.ActiveArea != 2 || ctx.Active().Cursor != 3 {
		t.Fatalf("alias TW: area=%d cursor=%d", ctx.ActiveArea, ctx.Active().Cursor)
	}
}
