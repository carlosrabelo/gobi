package repl

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
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

func TestDISPLAYPageFollowsScreenRows(t *testing.T) {
	dir := t.TempDir()
	fields := []dbf.Field{{Name: "N", Type: 'N', Length: 3, Decimals: 0}}
	recs := make([]dbf.Record, 50)
	for i := range recs {
		recs[i] = dbf.Record{Values: []dbf.Value{{Num: float64(i + 1)}}}
	}
	mustCreateDBF(t, filepath.Join(dir, "many.dbf"), fields, recs)
	ctx := testCtx(t, dir)
	ctx.Screen = term.NewSize(80, 40)
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE many"))
	out.Reset()
	_ = mux.Dispatch(ctx, command.Split("DISPLAY"))
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	// header + (40-4)=36 data lines
	if len(lines) != 37 {
		t.Fatalf("want 37 lines, got %d", len(lines))
	}
}

func TestDISPLAYPageAndRejectTO(t *testing.T) {
	dir := t.TempDir()
	fields := []dbf.Field{{Name: "N", Type: 'N', Length: 3, Decimals: 0}}
	recs := make([]dbf.Record, 25)
	for i := range recs {
		recs[i] = dbf.Record{Values: []dbf.Value{{Num: float64(i + 1)}}}
	}
	mustCreateDBF(t, filepath.Join(dir, "many.dbf"), fields, recs)

	ctx := testCtx(t, dir)
	var out, errBuf bytes.Buffer
	ctx.Stdout = &out
	ctx.Stderr = &errBuf
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE many")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("DISPLAY")); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	// header + 20 data lines
	if len(lines) != 21 {
		t.Fatalf("want 21 lines, got %d: %q", len(lines), out.String())
	}
	if ctx.Active().Cursor != 20 {
		t.Fatalf("cursor %d, want 20", ctx.Active().Cursor)
	}

	err := mux.Dispatch(ctx, command.Split("DISPLAY TO FILE x"))
	if err == nil {
		t.Fatal("expected TO rejection")
	}
	if !strings.Contains(errBuf.String(), "*** ") {
		t.Fatalf("stderr %q", errBuf.String())
	}
}

func TestLISTForAndWhile(t *testing.T) {
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
	if err := mux.Dispatch(ctx, command.Split("LIST FOR AGE > 15")); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "ANN") || !strings.Contains(got, "BOB") || !strings.Contains(got, "CYN") {
		t.Fatalf("FOR: %q", got)
	}

	if err := mux.Dispatch(ctx, command.Split("GO TOP")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("LIST WHILE AGE < 25")); err != nil {
		t.Fatal(err)
	}
	got = out.String()
	if !strings.Contains(got, "ANN") || !strings.Contains(got, "BOB") || strings.Contains(got, "CYN") {
		t.Fatalf("WHILE: %q", got)
	}
}
