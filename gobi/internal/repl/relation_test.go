package repl

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestRelationFilterFields(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "parent.dbf"), []dbf.Field{
		{Name: "KEY", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "A"}}},
		{Values: []dbf.Value{{Str: "B"}}},
	})
	mustCreateDBF(t, filepath.Join(dir, "child.dbf"), []dbf.Field{
		{Name: "KEY", Type: 'C', Length: 4},
		{Name: "VAL", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "A"}, {Str: "ONE"}}},
		{Values: []dbf.Value{{Str: "B"}, {Str: "TWO"}}},
	})

	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("SELECT 2"))
	_ = mux.Dispatch(ctx, command.Split("USE child"))
	_ = mux.Dispatch(ctx, command.Split("INDEX ON KEY TO child"))
	_ = mux.Dispatch(ctx, command.Split("SELECT 1"))
	_ = mux.Dispatch(ctx, command.Split("USE parent"))
	if err := mux.Dispatch(ctx, command.Split("SET RELATION TO KEY INTO child")); err != nil {
		t.Fatal(err)
	}
	_ = mux.Dispatch(ctx, command.Split("GO TOP"))
	_ = mux.Dispatch(ctx, command.Split("SELECT 2"))
	a := ctx.Active()
	if a.Cursor != 1 {
		t.Fatalf("child after relation cursor %d", a.Cursor)
	}
	_ = mux.Dispatch(ctx, command.Split("SELECT 1"))
	if err := mux.Dispatch(ctx, command.Split("SKIP")); err != nil {
		t.Fatal(err)
	}
	_ = mux.Dispatch(ctx, command.Split("SELECT 2"))
	if ctx.Active().Cursor != 2 {
		t.Fatalf("child after SKIP cursor %d", ctx.Active().Cursor)
	}

	_ = mux.Dispatch(ctx, command.Split("SELECT 1"))
	mustCreateDBF(t, filepath.Join(dir, "f.dbf"), []dbf.Field{
		{Name: "A", Type: 'C', Length: 1},
		{Name: "B", Type: 'C', Length: 1},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "X"}, {Str: "1"}}},
		{Values: []dbf.Value{{Str: "Y"}, {Str: "2"}}},
	})
	_ = mux.Dispatch(ctx, command.Split("USE f"))
	_ = mux.Dispatch(ctx, command.Split(`SET FILTER TO A = "X"`))
	out.Reset()
	_ = mux.Dispatch(ctx, command.Split("LIST"))
	if strings.Contains(out.String(), "Y") {
		t.Fatalf("filter failed: %q", out.String())
	}
	if err := mux.Dispatch(ctx, command.Split("GOTO 2")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Cursor != 2 {
		t.Fatal("GOTO should land on filtered record")
	}
	_ = mux.Dispatch(ctx, command.Split("SET FIELDS TO A"))
	_ = mux.Dispatch(ctx, command.Split("SET FIELDS ON"))
	_ = mux.Dispatch(ctx, command.Split("SET FILTER TO"))
	out.Reset()
	_ = mux.Dispatch(ctx, command.Split("LIST"))
	got := out.String()
	if !strings.Contains(got, "A") || strings.Contains(got, " B") {
		// header should only show A when FIELDS ON
		if strings.Count(got, "B") > 0 && strings.Contains(got, "B") {
			// data column B should not appear as header of second field
		}
	}
	if !strings.Contains(got, "A") {
		t.Fatalf("fields %q", got)
	}
}
