package repl

import (
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestUPDATEReplace(t *testing.T) {
	dir := t.TempDir()
	fields := []dbf.Field{
		{Name: "ID", Type: 'C', Length: 2},
		{Name: "NAME", Type: 'C', Length: 4},
	}
	mustCreateDBF(t, filepath.Join(dir, "tgt.dbf"), fields, []dbf.Record{
		{Values: []dbf.Value{{Str: "01"}, {Str: "OLD1"}}},
		{Values: []dbf.Value{{Str: "02"}, {Str: "OLD2"}}},
	})
	mustCreateDBF(t, filepath.Join(dir, "src.dbf"), fields, []dbf.Record{
		{Values: []dbf.Value{{Str: "01"}, {Str: "NEW1"}}},
		{Values: []dbf.Value{{Str: "03"}, {Str: "XXX"}}},
	})
	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE tgt")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("UPDATE ON ID FROM src REPLACE NAME")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("GOTO 1")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Record.Values[1].Str != "NEW1" {
		t.Fatalf("got %q", ctx.Active().Record.Values[1].Str)
	}
	if err := mux.Dispatch(ctx, command.Split("GOTO 2")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Record.Values[1].Str != "OLD2" {
		t.Fatalf("unchanged got %q", ctx.Active().Record.Values[1].Str)
	}
}
