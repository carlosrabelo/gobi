package repl

import (
	"os"
	"path/filepath"
	"strings"
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

func TestCOPYStructureAndCreateFrom(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "src.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 10},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{{Values: []dbf.Value{{Str: "X"}, {Num: 1}}}})
	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE src")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("COPY TO empty STRUCTURE")); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(dir, "empty.dbf"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	tab, err := dbf.Open(f)
	if err != nil {
		t.Fatal(err)
	}
	if tab.Header.RecordCount != 0 || len(tab.Header.Fields) != 2 {
		t.Fatalf("structure %+v count %d", tab.Header.Fields, tab.Header.RecordCount)
	}
	f.Close()

	if err := mux.Dispatch(ctx, command.Split("COPY TO ext STRUCTURE EXTENDED")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("CREATE neu FROM ext")); err != nil {
		t.Fatal(err)
	}
	f2, err := os.OpenFile(filepath.Join(dir, "neu.dbf"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()
	tab2, err := dbf.Open(f2)
	if err != nil {
		t.Fatal(err)
	}
	if tab2.Header.RecordCount != 0 || len(tab2.Header.Fields) != 2 || tab2.Header.Fields[0].Name != "NAME" {
		t.Fatalf("from extended %+v", tab2.Header)
	}
}

func TestCOPYSDFAndDelimited(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "src.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN"}, {Num: 10}}},
	})
	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE src")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("COPY TO out.txt SDF")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "ANN") || !strings.Contains(string(raw), "10") {
		t.Fatalf("SDF %q", raw)
	}
	if err := mux.Dispatch(ctx, command.Split("COPY TO d.txt DELIMITED")); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(filepath.Join(dir, "d.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"ANN"`) {
		t.Fatalf("DELIMITED %q", raw)
	}
}
