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
