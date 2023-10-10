package repl

import (
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"path/filepath"
)

func TestMODIFYSTRUCTUREDropField(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN"}, {Num: 10}}},
		{Deleted: true, Values: []dbf.Value{{Str: "XX"}, {Num: 99}}},
	})
	ctx := testCtx(t, dir)
	// Keep only NAME
	ctx.Stdin = strings.NewReader("NAME\nC\n4\n0\n\n")
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("MODIFY STRUCTURE")); err != nil {
		t.Fatal(err)
	}
	a := ctx.Active()
	if len(a.Table.Header.Fields) != 1 || a.Table.Header.Fields[0].Name != "NAME" {
		t.Fatalf("fields %+v", a.Table.Header.Fields)
	}
	if a.RecCount() != 2 {
		t.Fatalf("count %d", a.RecCount())
	}
	if err := mux.Dispatch(ctx, command.Split("GOTO 2")); err != nil {
		t.Fatal(err)
	}
	if !a.Record.Deleted || a.Record.Values[0].Str != "XX" {
		t.Fatalf("deleted mark/value %+v", a.Record)
	}
	if a.Cursor != 2 {
		// after modify cursor should be 1; we GOTO 2
	}
	if ctx.Active().Cursor != 2 {
		t.Fatalf("cursor %d", ctx.Active().Cursor)
	}
}
