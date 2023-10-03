package repl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestCOPYTOFor(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "src.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN"}, {Num: 10}}},
		{Deleted: true, Values: []dbf.Value{{Str: "XX"}, {Num: 99}}},
		{Values: []dbf.Value{{Str: "BOB"}, {Num: 20}}},
	})

	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE src")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("GOTO 3")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("COPY TO out FOR AGE > 15")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Cursor != 3 {
		t.Fatalf("cursor restored? got %d", ctx.Active().Cursor)
	}

	f, err := os.OpenFile(filepath.Join(dir, "out.dbf"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tab, err := dbf.Open(f)
	if err != nil {
		t.Fatal(err)
	}
	if tab.Header.RecordCount != 1 {
		t.Fatalf("count %d", tab.Header.RecordCount)
	}
	rec, err := tab.Read(1)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Values[0].Str != "BOB" {
		t.Fatalf("got %+v", rec)
	}
}
