package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func TestREPORTFormAndEJECT(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN"}, {Num: 10}}},
		{Values: []dbf.Value{{Str: "BOB"}, {Num: 20}}},
	})
	frmPath := filepath.Join(dir, "r.frm")
	if err := os.WriteFile(frmPath, []byte("People\nNAME 4\nAGE 3\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	ctx := testCtx(t, dir)
	ctx.Stdout = &out
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("REPORT FORM r PLAIN")); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "People") || !strings.Contains(got, "ANN") || !strings.Contains(got, "BOB") {
		t.Fatalf("report %q", got)
	}
	if !strings.Contains(got, "NAME") || !strings.Contains(got, "AGE") {
		t.Fatalf("headers %q", got)
	}

	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("EJECT")); err != nil {
		t.Fatal(err)
	}
	if out.String() != "\x0c" {
		t.Fatalf("EJECT got %q", out.String())
	}
}

func TestMODIFYCommandSaveAndQuit(t *testing.T) {
	dir := t.TempDir()
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	// type "X", Ctrl-W save
	ctx.Stdin = strings.NewReader("X\x17")
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("MODIFY COMMAND p")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "p.prg"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "X\n" {
		t.Fatalf("saved %q", data)
	}

	// Ctrl-Q discard: start with existing, type Z, quit without save
	if err := os.WriteFile(filepath.Join(dir, "q.prg"), []byte("OLD\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx.Stdin = strings.NewReader("Z\x11") // Z then Ctrl-Q
	ctx.Keys = term.NewReader(ctx.Stdin)
	if err := mux.Dispatch(ctx, command.Split("MODIFY COMMAND q")); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(dir, "q.prg"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "OLD\n" {
		t.Fatalf("discarded got %q", data)
	}
}
