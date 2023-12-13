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

func TestAPPENDLineMode(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, nil)

	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	// ANN, Tab, 10, Ctrl-W, BOB, Tab, 20, Ctrl-W, Esc
	ctx.Stdin = strings.NewReader("ANN\t10\x17BOB\t20\x17\x1b")
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("APPEND")); err != nil {
		t.Fatal(err)
	}
	a := ctx.Active()
	if a.RecCount() != 2 {
		t.Fatalf("count %d", a.RecCount())
	}
}

func TestSETFormatTODrivesEDIT(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN "}}},
	})
	fmtPath := filepath.Join(dir, "f.fmt")
	if err := os.WriteFile(fmtPath, []byte("@ 0,0 SAY \"N\"\n@ 0,2 GET NAME\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))
	if err := mux.Dispatch(ctx, command.Split("SET FORMAT TO f")); err != nil {
		t.Fatal(err)
	}
	if len(ctx.FormatLines) == 0 || !ctx.Active().Format {
		t.Fatal("format not loaded")
	}
	ctx.Stdin = strings.NewReader("\x7f\x7f\x7f\x7fBOB\r")
	if err := mux.Dispatch(ctx, command.Split("EDIT")); err != nil {
		t.Fatal(err)
	}
	_ = ctx.Active().LoadCurrent()
	if strings.TrimSpace(ctx.Active().Record.Values[0].Str) != "BOB" {
		t.Fatalf("format EDIT: %q", ctx.Active().Record.Values[0].Str)
	}
	_ = mux.Dispatch(ctx, command.Split("SET FORMAT TO"))
	if ctx.Config.Format != "" || len(ctx.FormatLines) != 0 {
		t.Fatal("format not closed")
	}
}

func TestSETCarrySeedsAppendForm(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN "}, {Num: 10}}},
	})
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))
	_ = mux.Dispatch(ctx, command.Split("SET CARRY ON"))
	if !ctx.Config.Carry {
		t.Fatal("CARRY still off")
	}
	// First APPEND form is seeded from ANN/10; Tab to AGE, Ctrl-W saves carried values; Esc exits.
	ctx.Stdin = strings.NewReader("\t\x17\x1b")
	if err := mux.Dispatch(ctx, command.Split("APPEND")); err != nil {
		t.Fatal(err)
	}
	a := ctx.Active()
	if a.RecCount() != 2 {
		t.Fatalf("count %d", a.RecCount())
	}
	rec, err := a.Table.Read(2)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(rec.Values[0].Str) != "ANN" || rec.Values[1].Num != 10 {
		t.Fatalf("carried record: %+v", rec.Values)
	}

	// APPEND BLANK stays blank even with CARRY on.
	if err := mux.Dispatch(ctx, command.Split("APPEND BLANK")); err != nil {
		t.Fatal(err)
	}
	blank, err := a.Table.Read(a.RecCount())
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(blank.Values[0].Str) != "" || blank.Values[1].Num != 0 {
		t.Fatalf("APPEND BLANK not blank: %+v", blank.Values)
	}
}

func TestEDITCtrlW(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN "}}},
	})
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("\x7f\x7f\x7f\x7fBOB\x17")
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("EDIT")); err != nil {
		t.Fatal(err)
	}
	_ = ctx.Active().LoadCurrent()
	if strings.TrimSpace(ctx.Active().Record.Values[0].Str) != "BOB" {
		t.Fatalf("got %q", ctx.Active().Record.Values[0].Str)
	}
}
