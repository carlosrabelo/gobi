package repl

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestLISTAllAndFieldList(t *testing.T) {
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
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()

	if err := mux.Dispatch(ctx, command.Split("USE peeps")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("LIST")); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Record#") || !strings.Contains(got, "NAME") || !strings.Contains(got, "AGE") {
		t.Fatalf("header: %q", got)
	}
	if !strings.Contains(got, "ANN") || !strings.Contains(got, "BOB") || !strings.Contains(got, "CYN") {
		t.Fatalf("rows: %q", got)
	}
	a := ctx.Active()
	if a.Cursor != 4 || a.Landed {
		t.Fatalf("cursor after LIST: %d landed=%v", a.Cursor, a.Landed)
	}

	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("LIST NAME")); err != nil {
		t.Fatal(err)
	}
	got = out.String()
	if !strings.Contains(got, "NAME") || strings.Contains(got, " AGE") {
		// header should name NAME; AGE column title absent as separate field list
	}
	if !strings.Contains(got, "ANN") {
		t.Fatalf("LIST NAME: %q", got)
	}
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) < 2 {
		t.Fatalf("lines: %q", got)
	}
	if strings.Contains(lines[0], "AGE") {
		t.Fatalf("field list header still has AGE: %q", lines[0])
	}
}
