package repl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestJOINOneMatch(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "a.dbf"), []dbf.Field{
		{Name: "ID", Type: 'C', Length: 2},
		{Name: "ANAME", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "01"}, {Str: "AAA"}}},
	})
	mustCreateDBF(t, filepath.Join(dir, "b.dbf"), []dbf.Field{
		{Name: "ID", Type: 'C', Length: 2},
		{Name: "BNAME", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "01"}, {Str: "BBB"}}},
		{Values: []dbf.Value{{Str: "02"}, {Str: "XXX"}}},
	})

	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE a")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("SELECT 2")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("USE b ALIAS B")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("SELECT 1")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("JOIN WITH B TO out FOR ID = B->ID")); err != nil {
		t.Fatal(err)
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
	// fields: ID, ANAME, BNAME
	if len(tab.Header.Fields) != 3 || rec.Values[1].Str != "AAA" || rec.Values[2].Str != "BBB" {
		t.Fatalf("fields %+v rec %+v", tab.Header.Fields, rec)
	}
}
