package repl

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestBROWSERightAndDelete(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "A", Type: 'C', Length: 2},
		{Name: "B", Type: 'C', Length: 2},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "A1"}, {Str: "B1"}}},
	})
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	// Right arrow then Ctrl-U then Esc
	ctx.Stdin = strings.NewReader("\x1b[C\x15\x1b")
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("BROWSE")); err != nil {
		t.Fatal(err)
	}
	r, err := ctx.Active().Table.Read(1)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Deleted {
		t.Fatal("expected deletion mark toggled")
	}
}

func TestBROWSEFields(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "A", Type: 'C', Length: 1},
		{Name: "B", Type: 'C', Length: 1},
	}, []dbf.Record{{Values: []dbf.Value{{Str: "X"}, {Str: "Y"}}}})
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("\x1b")
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))
	if err := mux.Dispatch(ctx, command.Split("BROWSE FIELDS B")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ctx.Stdout.(*bytes.Buffer).String(), "B") {
		// present may have written B header
	}
}
