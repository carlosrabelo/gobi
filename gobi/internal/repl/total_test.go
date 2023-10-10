package repl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestTOTALTwoRowGroup(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "src.dbf"), []dbf.Field{
		{Name: "GRP", Type: 'C', Length: 1},
		{Name: "AMT", Type: 'N', Length: 5, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "A"}, {Num: 10}}},
		{Values: []dbf.Value{{Str: "A"}, {Num: 15}}},
		{Values: []dbf.Value{{Str: "B"}, {Num: 3}}},
	})
	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE src")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("TOTAL ON GRP TO out")); err != nil {
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
	if tab.Header.RecordCount != 2 {
		t.Fatalf("count %d", tab.Header.RecordCount)
	}
	r1, _ := tab.Read(1)
	if r1.Values[0].Str != "A" || r1.Values[1].Num != 25 {
		t.Fatalf("group A %+v", r1)
	}
}
