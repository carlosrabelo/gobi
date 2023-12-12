package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func TestSETAlternateCopiesOutput(t *testing.T) {
	dir := t.TempDir()
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split(`STORE 42 TO N`))
	if err := mux.Dispatch(ctx, command.Split("SET ALTERNATE TO alt")); err != nil {
		t.Fatal(err)
	}
	altPath := filepath.Join(dir, "alt.txt")
	if _, err := os.Stat(altPath); err != nil {
		t.Fatalf("alternate file: %v", err)
	}
	_ = mux.Dispatch(ctx, command.Split("SET ALTERNATE ON"))
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("? N")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "42") {
		t.Fatalf("screen: %q", out.String())
	}
	raw, err := os.ReadFile(altPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "42") {
		t.Fatalf("alternate file: %q", raw)
	}
	_ = mux.Dispatch(ctx, command.Split("SET ALTERNATE OFF"))
	_ = mux.Dispatch(ctx, command.Split("SET ALTERNATE TO"))
}

func TestSETConsoleOffDiscardsOutput(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split(`STORE 1 TO A`))
	_ = mux.Dispatch(ctx, command.Split("SET CONSOLE OFF"))
	if ctx.Config.Console {
		t.Fatal("CONSOLE still on")
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("? A")); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("CONSOLE OFF leaked: %q", out.String())
	}
	_ = mux.Dispatch(ctx, command.Split("SET CONSOLE ON"))
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("? A")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "1") {
		t.Fatalf("CONSOLE ON: %q", out.String())
	}
}

func TestSETTalkAndDefault(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "N", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{{Values: []dbf.Value{{Num: 1}}}, {Values: []dbf.Value{{Num: 2}}}})
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))
	_ = mux.Dispatch(ctx, command.Split("SET TALK OFF"))
	out.Reset()
	_ = mux.Dispatch(ctx, command.Split("DELETE ALL")) // may not print if no talk for delete
	if ctx.Config.Talk {
		t.Fatal("talk still on")
	}
	_ = mux.Dispatch(ctx, command.Split("SET DEFAULT TO "+dir))
	if ctx.Config.Default != dir {
		t.Fatalf("default %q", ctx.Config.Default)
	}
}

func TestDIRAndHELP(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "a.dbf"), []dbf.Field{
		{Name: "N", Type: 'N', Length: 3, Decimals: 0},
	}, nil)
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("DIR")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "a.dbf") {
		t.Fatalf("DIR: %q", out.String())
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("HELP USE")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "USE") {
		t.Fatalf("HELP: %q", out.String())
	}
}

func TestSETExactAndDeleted(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 5},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "Smith"}}},
		{Values: []dbf.Value{{Str: "Sm   "}}, Deleted: true},
	})
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))
	_ = mux.Dispatch(ctx, command.Split("SET EXACT ON"))
	tree, errs := expr.Parse(`"Smith" = "Sm"`)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	v, err := expr.Eval(tree, newEnv(ctx))
	if err != nil {
		t.Fatal(err)
	}
	if v.Log {
		t.Fatal("EXACT ON should make Smith != Sm")
	}
	_ = mux.Dispatch(ctx, command.Split("SET DELETED ON"))
	out.Reset()
	_ = mux.Dispatch(ctx, command.Split("LIST"))
	if strings.Contains(out.String(), "      2") {
		t.Fatalf("DELETED ON should hide starred row: %q", out.String())
	}
	_ = mux.Dispatch(ctx, command.Split(`SET COLOR TO W/N`))
	if !strings.Contains(ctx.Config.Color, "W/N") {
		t.Fatalf("COLOR %q", ctx.Config.Color)
	}
}

func TestRUN(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("RUN echo hi")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "hi") {
		t.Fatalf("RUN: %q", out.String())
	}
	_ = os.Getenv // keep
}
