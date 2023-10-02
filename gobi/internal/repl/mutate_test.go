package repl

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestDELETERecallPackZap(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "A1"}}},
		{Values: []dbf.Value{{Str: "A2"}}},
		{Values: []dbf.Value{{Str: "A3"}}},
	})

	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("GOTO 2")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("DELETE")); err != nil {
		t.Fatal(err)
	}
	if !ctx.Active().Record.Deleted {
		t.Fatal("not deleted")
	}
	if err := mux.Dispatch(ctx, command.Split("RECALL")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Record.Deleted {
		t.Fatal("still deleted")
	}
	if err := mux.Dispatch(ctx, command.Split("DELETE")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("PACK")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().RecCount() != 2 {
		t.Fatalf("after pack count %d", ctx.Active().RecCount())
	}
	if ctx.Active().Cursor != 1 || ctx.Active().Record.Values[0].Str != "A1" {
		t.Fatalf("pack cursor %+v", ctx.Active().Record)
	}

	ctx.Config.Safety = true
	ctx.Stdin = strings.NewReader("Y\n")
	if err := mux.Dispatch(ctx, command.Split("ZAP")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().RecCount() != 0 {
		t.Fatalf("zap count %d", ctx.Active().RecCount())
	}
}
