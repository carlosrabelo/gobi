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

func TestASSISTShellEscLeaves(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	ctx.Stderr = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader(string(rune(term.KeyEsc)))
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("ASSIST")); err != nil {
		t.Fatal(err)
	}
	if ctx.AssistOpen {
		t.Fatal("AssistOpen still set")
	}
	if ctx.Screen == nil {
		t.Fatal("expected screen")
	}
	// Menu bar starts with Set Up (possibly reverse).
	row0 := ""
	for c := 0; c < 10 && c < ctx.Screen.Cols; c++ {
		ch := ctx.Screen.CellAt(0, c).Ch
		if ch != 0 && ch != ' ' {
			row0 += string(ch)
		}
	}
	if !strings.HasPrefix(row0, "Set") && !strings.Contains(row0, "S") {
		// After leave, last frame still has the bar in pending cells.
		got := ctx.Screen.CellAt(0, 0).Ch
		if got != 'S' {
			t.Fatalf("menu bar cell0=%q row=%q", got, row0)
		}
	}
	msg := ""
	for c := 0; c < ctx.Screen.Cols; c++ {
		ch := ctx.Screen.CellAt(ctx.Screen.Rows-2, c).Ch
		if ch != ' ' && ch != 0 {
			msg += string(ch)
		}
	}
	if !strings.Contains(msg, "Esc") {
		t.Fatalf("message line %q", msg)
	}
	stat := ""
	for c := 0; c < ctx.Screen.Cols; c++ {
		ch := ctx.Screen.CellAt(ctx.Screen.Rows-1, c).Ch
		if ch != ' ' && ch != 0 {
			stat += string(ch)
		}
	}
	if !strings.Contains(stat, "Rec:") {
		t.Fatalf("status %q", stat)
	}
}

func TestASSISTNavigationSubmenu(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}
	// Right to Create, Down open, letter D jumps? Create items Database/From.
	// Down opens Set Up submenu, Esc back, Esc leave.
	ctx.Stdin = strings.NewReader(
		string(rune(term.KeyRight)) + // Create
			string(rune(term.KeyDown)) + // open
			string(rune(term.KeyEsc)) + // back
			string(rune(term.KeyEsc)), // leave
	)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("ASSIST")); err != nil {
		t.Fatal(err)
	}
	if assistCreateHasReport() {
		t.Fatal("Create submenu must not offer Report")
	}
}

func TestASSISTUseFromSetUp(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "c.dbf"), []dbf.Field{
		{Name: "N", Type: 'C', Length: 3},
	}, []dbf.Record{{Values: []dbf.Value{{Str: "AB "}}}})
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}
	// Set Up is first; Down open; Enter Database file; type c; Enter; Esc Esc leave.
	in := string(rune(term.KeyDown)) + string(rune(term.KeyEnter)) + "c" + string(rune(term.KeyEnter)) +
		string(rune(term.KeyEsc)) + string(rune(term.KeyEsc))
	ctx.Stdin = strings.NewReader(in)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("ASSIST")); err != nil {
		t.Fatal(err)
	}
	if !ctx.Active().HasTable() {
		t.Fatal("expected USE c from Assist")
	}
}

func TestASSISTGotoFromPosition(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "c.dbf"), []dbf.Field{
		{Name: "N", Type: 'C', Length: 3},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "A  "}}},
		{Values: []dbf.Value{{Str: "B  "}}},
	})
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE c"))
	// Position is index 3: Right×3, Down, Down (Skip is 2nd? Goto is first), Enter, "2", Enter, Esc Esc
	in := strings.Repeat(string(rune(term.KeyRight)), 3) +
		string(rune(term.KeyDown)) + string(rune(term.KeyEnter)) + "2" + string(rune(term.KeyEnter)) +
		string(rune(term.KeyEsc)) + string(rune(term.KeyEsc))
	ctx.Stdin = strings.NewReader(in)
	if err := mux.Dispatch(ctx, command.Split("ASSIST")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().Cursor != 2 {
		t.Fatalf("cursor %d want 2", ctx.Active().Cursor)
	}
}

func TestASSISTPromptCancel(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "c.dbf"), []dbf.Field{
		{Name: "N", Type: 'C', Length: 3},
	}, nil)
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}
	// Open Database file prompt then Esc cancel; leave Assist.
	in := string(rune(term.KeyDown)) + string(rune(term.KeyEnter)) + string(rune(term.KeyEsc)) +
		string(rune(term.KeyEsc)) + string(rune(term.KeyEsc))
	ctx.Stdin = strings.NewReader(in)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("ASSIST")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().HasTable() {
		t.Fatal("cancelled USE should leave no table")
	}
}

func TestASSISTRejectsArgs(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}
	mux := NewMux()
	err := mux.Dispatch(ctx, command.Split("ASSIST NOW"))
	if err == nil || !strings.Contains(err.Error(), "no arguments") {
		t.Fatalf("%v", err)
	}
}
