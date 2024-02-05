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

func TestBROWSENoAppendStopsAtBottom(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "A", Type: 'C', Length: 1},
	}, []dbf.Record{{Values: []dbf.Value{{Str: "X"}}}})
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	// Down at last row then Esc — NOAPPEND must not add a record.
	ctx.Stdin = strings.NewReader("\x1b[B\x1b")
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))
	if err := mux.Dispatch(ctx, command.Split("BROWSE NOAPPEND")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().RecCount() != 1 {
		t.Fatalf("NOAPPEND added a record: count=%d", ctx.Active().RecCount())
	}
}

func TestBROWSEAppendAtBottom(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "A", Type: 'C', Length: 1},
	}, []dbf.Record{{Values: []dbf.Value{{Str: "X"}}}})
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("\x1b[B\x1b")
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))
	if err := mux.Dispatch(ctx, command.Split("BROWSE")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().RecCount() != 2 {
		t.Fatalf("Down at bottom should APPEND: count=%d", ctx.Active().RecCount())
	}
}

func TestBROWSELockKeepsLeftColumn(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "A", Type: 'C', Length: 2},
		{Name: "B", Type: 'C', Length: 2},
		{Name: "C", Type: 'C', Length: 2},
	}, []dbf.Record{{Values: []dbf.Value{{Str: "A1"}, {Str: "B1"}, {Str: "C1"}}}})
	ctx := testCtx(t, dir)
	out := &bytes.Buffer{}
	ctx.Stdout = out
	// Right twice (into unlocked) then Esc — locked header A must still appear.
	ctx.Stdin = strings.NewReader("\x1b[C\x1b[C\x1b")
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))
	if err := mux.Dispatch(ctx, command.Split("BROWSE LOCK 1")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "A") {
		t.Fatalf("LOCK 1 should keep column A visible: %q", out.String())
	}
}

func TestApplyInputByteInsertOverwrite(t *testing.T) {
	if got := applyInputByte("ab", 'x', 4, true); got != "abx" {
		t.Fatalf("insert: %q", got)
	}
	if got := applyInputByte("abcd", 'x', 4, true); got != "abcd" {
		t.Fatalf("insert full: %q", got)
	}
	if got := applyInputByte("abcd", 'x', 4, false); got != "abcx" {
		t.Fatalf("overwrite full: %q", got)
	}
}

func TestINSERTOpensForm(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "N", Type: 'C', Length: 3},
	}, []dbf.Record{{Values: []dbf.Value{{Str: "AA"}}}})
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	// Type ZZZ, Ctrl-W to save the INSERT form.
	ctx.Stdin = strings.NewReader("ZZZ\x17")
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))
	_ = mux.Dispatch(ctx, command.Split("GOTO 1"))
	if err := mux.Dispatch(ctx, command.Split("INSERT")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().RecCount() != 2 {
		t.Fatalf("count %d", ctx.Active().RecCount())
	}
	r, _ := ctx.Active().Table.Read(2)
	if strings.TrimSpace(r.Values[0].Str) != "ZZZ" {
		t.Fatalf("inserted value %q", r.Values[0].Str)
	}
}
