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
	ctx.Stdin = strings.NewReader("DAN\n40\n\n")
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
