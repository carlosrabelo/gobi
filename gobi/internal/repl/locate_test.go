package repl

import (
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestLOCATEAndCONTINUE(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "peeps.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN"}, {Num: 10}}},
		{Values: []dbf.Value{{Str: "BOB"}, {Num: 20}}},
		{Values: []dbf.Value{{Str: "CYN"}, {Num: 20}}},
		{Values: []dbf.Value{{Str: "DAN"}, {Num: 40}}},
	})

	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE peeps")); err != nil {
		t.Fatal(err)
	}

	if err := mux.Dispatch(ctx, command.Split("LOCATE FOR AGE = 20")); err != nil {
		t.Fatal(err)
	}
	a := ctx.Active()
	if !a.Found || a.Cursor != 2 {
		t.Fatalf("first LOCATE: found=%v cursor=%d", a.Found, a.Cursor)
	}

	if err := mux.Dispatch(ctx, command.Split("CONTINUE")); err != nil {
		t.Fatal(err)
	}
	if !a.Found || a.Cursor != 3 {
		t.Fatalf("CONTINUE: found=%v cursor=%d", a.Found, a.Cursor)
	}

	if err := mux.Dispatch(ctx, command.Split("CONTINUE")); err != nil {
		t.Fatal(err)
	}
	if a.Found || a.Cursor != 5 {
		t.Fatalf("CONTINUE EOF: found=%v cursor=%d", a.Found, a.Cursor)
	}

	if err := mux.Dispatch(ctx, command.Split("LOCATE FOR AGE = 40")); err != nil {
		t.Fatal(err)
	}
	if !a.Found || a.Cursor != 4 {
		t.Fatalf("second LOCATE replaces: found=%v cursor=%d", a.Found, a.Cursor)
	}
	if err := mux.Dispatch(ctx, command.Split("CONTINUE")); err != nil {
		t.Fatal(err)
	}
	if a.Found || a.Cursor != 5 {
		t.Fatalf("CONTINUE after second LOCATE: found=%v cursor=%d", a.Found, a.Cursor)
	}
}

func TestCONTINUEWithoutLOCATE(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "N", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{{Values: []dbf.Value{{Num: 1}}}})

	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("CONTINUE")); err == nil {
		t.Fatal("expected error")
	}
}
