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

func TestEDITForWalksMatches(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN"}, {Num: 10}}},
		{Values: []dbf.Value{{Str: "BOB"}, {Num: 20}}},
		{Values: []dbf.Value{{Str: "CYN"}, {Num: 20}}},
	})
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	// On first match (BOB): Ctrl-Y clear NAME, type ZZ, Ctrl-W → next match (CYN): Esc
	ctx.Stdin = strings.NewReader("\x19ZZ\x17" + string(rune(term.KeyEsc)))
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("EDIT FOR AGE = 20")); err != nil {
		t.Fatal(err)
	}
	r1, _ := ctx.Active().Table.Read(2)
	if strings.TrimSpace(r1.Values[0].Str) != "ZZ" {
		t.Fatalf("match 1 NAME: %q", r1.Values[0].Str)
	}
	r2, _ := ctx.Active().Table.Read(3)
	if strings.TrimSpace(r2.Values[0].Str) != "CYN" {
		t.Fatalf("match 2 should be unchanged: %q", r2.Values[0].Str)
	}
}

func TestEDITUpDownFields(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "A", Type: 'C', Length: 2},
		{Name: "B", Type: 'C', Length: 2},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "A1"}, {Str: "B1"}}},
	})
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	// Down to field B, Ctrl-Y, type XX, Ctrl-W
	in := "\x1b[B" + "\x19XX\x17"
	ctx.Stdin = strings.NewReader(in)
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))
	if err := mux.Dispatch(ctx, command.Split("EDIT")); err != nil {
		t.Fatal(err)
	}
	_ = ctx.Active().LoadCurrent()
	if strings.TrimSpace(ctx.Active().Record.Values[0].Str) != "A1" {
		t.Fatalf("A changed: %q", ctx.Active().Record.Values[0].Str)
	}
	if strings.TrimSpace(ctx.Active().Record.Values[1].Str) != "XX" {
		t.Fatalf("B: %q", ctx.Active().Record.Values[1].Str)
	}
}
