package repl

import (
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestAPPENDFromByName(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "tgt.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "OLD"}, {Num: 1}}},
	})
	mustCreateDBF(t, filepath.Join(dir, "src.dbf"), []dbf.Field{
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "XTRA", Type: 'C', Length: 1},
	}, []dbf.Record{
		{Values: []dbf.Value{{Num: 20}, {Str: "NEW"}, {Str: "Z"}}},
		{Deleted: true, Values: []dbf.Value{{Num: 99}, {Str: "XX"}, {Str: "Q"}}},
	})

	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE tgt")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("APPEND FROM src")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().RecCount() != 2 {
		t.Fatalf("count %d", ctx.Active().RecCount())
	}
	if ctx.Active().Cursor != 2 || ctx.Active().Record.Values[0].Str != "NEW" || ctx.Active().Record.Values[1].Num != 20 {
		t.Fatalf("last %+v", ctx.Active().Record)
	}
}
