package repl

import (
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestREPLACECurrentAndFor(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN"}, {Num: 10}}},
		{Values: []dbf.Value{{Str: "BOB"}, {Num: 20}}},
	})

	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split(`REPLACE NAME WITH "ZZ"`)); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Record.Values[0].Str != "ZZ" {
		t.Fatalf("got %q", ctx.Active().Record.Values[0].Str)
	}

	if err := mux.Dispatch(ctx, command.Split("REPLACE AGE WITH 99 FOR AGE > 15")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("GOTO 2")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Record.Values[1].Num != 99 {
		t.Fatalf("age %v", ctx.Active().Record.Values[1].Num)
	}
	if err := mux.Dispatch(ctx, command.Split("GOTO 1")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Record.Values[1].Num != 10 {
		t.Fatalf("rec1 age changed: %v", ctx.Active().Record.Values[1].Num)
	}
}
