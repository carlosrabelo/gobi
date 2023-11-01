package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/ndx"
)

func TestINDEXON(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "peeps.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN"}, {Num: 10}}},
		{Values: []dbf.Value{{Str: "BOB"}, {Num: 20}}},
		{Values: []dbf.Value{{Str: "CYN"}, {Num: 30}}},
	})
	ctx := testCtx(t, dir)
	ctx.Config.Safety = false
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE peeps")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("INDEX ON AGE TO ages")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ages.ndx")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	a := ctx.Active()
	if a.NDX[0] != path {
		t.Fatalf("bound: %q", a.NDX[0])
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	idx, err := ndx.Open(f)
	if err != nil {
		t.Fatal(err)
	}
	if idx.Header().Expr != "age" {
		t.Fatalf("expr %q", idx.Header().Expr)
	}
	rec, ok, err := idx.Search(ndx.EncodeKeyNum(20))
	if err != nil || !ok || rec != 2 {
		t.Fatalf("search: rec=%d ok=%v err=%v", rec, ok, err)
	}
}

func TestSEEKHitAndMiss(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "peeps.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN"}, {Num: 10}}},
		{Values: []dbf.Value{{Str: "BOB"}, {Num: 20}}},
	})
	ctx := testCtx(t, dir)
	ctx.Config.Safety = false
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE peeps")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("INDEX ON AGE TO ages")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("SEEK 20")); err != nil {
		t.Fatal(err)
	}
	a := ctx.Active()
	if !a.Found || a.Cursor != 2 {
		t.Fatalf("SEEK hit: found=%v cursor=%d", a.Found, a.Cursor)
	}
	if err := mux.Dispatch(ctx, command.Split("SEEK 99")); err != nil {
		t.Fatal(err)
	}
	if a.Found || a.Cursor != 3 {
		t.Fatalf("SEEK miss: found=%v cursor=%d", a.Found, a.Cursor)
	}
}

func TestAPPENDUpdatesIndex(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "peeps.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN"}, {Num: 10}}},
	})
	ctx := testCtx(t, dir)
	ctx.Config.Safety = false
	// Full-screen APPEND: NAME then AGE then Ctrl-W then Esc.
	ctx.Stdin = strings.NewReader("DAN\t40\x17\x1b")
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE peeps")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("INDEX ON AGE TO ages")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("APPEND")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("SEEK 40")); err != nil {
		t.Fatal(err)
	}
	if !ctx.Active().Found || ctx.Active().Cursor != 2 {
		t.Fatalf("APPEND not in index: found=%v cursor=%d", ctx.Active().Found, ctx.Active().Cursor)
	}
}

func TestSORTON(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "peeps.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "CYN"}, {Num: 30}}},
		{Values: []dbf.Value{{Str: "ANN"}, {Num: 10}}},
		{Values: []dbf.Value{{Str: "BOB"}, {Num: 20}}, Deleted: true},
	})
	ctx := testCtx(t, dir)
	ctx.Config.Safety = false
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE peeps")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("SORT ON AGE TO ordered")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ordered.dbf")
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tab, err := dbf.Open(f)
	if err != nil {
		t.Fatal(err)
	}
	if tab.Header.RecordCount != 2 {
		t.Fatalf("count %d (starred omitted)", tab.Header.RecordCount)
	}
	r1, _ := tab.Read(1)
	r2, _ := tab.Read(2)
	if r1.Values[1].Num != 10 || r2.Values[1].Num != 30 {
		t.Fatalf("order: %v then %v", r1.Values[1].Num, r2.Values[1].Num)
	}
	// Source unchanged.
	if ctx.Active().RecCount() != 3 {
		t.Fatalf("source count %d", ctx.Active().RecCount())
	}
}
