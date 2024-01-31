package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestLABELForm(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "A", Type: 'C', Length: 4},
		{Name: "B", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ONE"}, {Str: "AAA"}}},
		{Values: []dbf.Value{{Str: "TWO"}, {Str: "BBB"}}},
	})
	lbl := "2 20 1 1 0\nA\nB\n"
	if err := os.WriteFile(filepath.Join(dir, "l.lbl"), []byte(lbl), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))
	if err := mux.Dispatch(ctx, command.Split("LABEL FORM l")); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "ONE") || !strings.Contains(got, "TWO") {
		t.Fatalf("%q", got)
	}
	cur := ctx.Active().Cursor
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("LABEL FORM l SAMPLE")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Cursor != cur {
		t.Fatal("SAMPLE moved cursor")
	}
	if !strings.Contains(out.String(), "XXXX") {
		t.Fatalf("sample %q", out.String())
	}
	if err := mux.Dispatch(ctx, command.Split("LABEL FORM l TO FILE out")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "out.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestLOADMissingModuleErrors(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var errBuf bytes.Buffer
	ctx.Stderr = &errBuf
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	err := mux.Dispatch(ctx, command.Split("LOAD x"))
	if err == nil || !strings.Contains(err.Error(), "cannot load") {
		t.Fatalf("%v", err)
	}
}
