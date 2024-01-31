package repl

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestDISPLAYSTRUCTURE(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "cust.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 10},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, nil)

	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()

	if err := mux.Dispatch(ctx, command.Split("DISPLAY STRUCTURE")); err == nil {
		t.Fatal("expected error with no table")
	}

	if err := mux.Dispatch(ctx, command.Split("USE cust")); err != nil {
		t.Fatal(err)
	}
	cur := ctx.Active().Cursor
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("DISPLAY STRUCTURE")); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "NAME") || !strings.Contains(got, "Character") {
		t.Fatalf("missing Character field line: %q", got)
	}
	if !strings.Contains(got, "Structure for database:") || !strings.Contains(got, "** Total **") {
		t.Fatalf("missing III Plus STRUCTURE framing: %q", got)
	}
	if !strings.Contains(got, "cust.dbf") {
		t.Fatalf("missing file name: %q", got)
	}
	if ctx.Active().Cursor != cur {
		t.Fatal("STRUCTURE moved the cursor")
	}

	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("LIST STRUCTURE")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "AGE") {
		t.Fatalf("LIST STRUCTURE: %q", out.String())
	}
}
