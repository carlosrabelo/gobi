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
	// Leave blanks the cell buffer and homes the terminal for the dot prompt.
	if ctx.Screen.CellAt(0, 0).Ch != ' ' {
		t.Fatalf("bar not cleared: %q", ctx.Screen.CellAt(0, 0).Ch)
	}
	if !strings.Contains(out.String(), "\x1b[2J\x1b[H") {
		t.Fatalf("terminal not cleared: %q", out.String())
	}
}

// Leftover LF after typing ASSIST+Enter must not open a submenu (one Esc leaves).
func TestASSISTEscLeavesDespiteTrailingLF(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("\n" + string(rune(term.KeyEsc)))
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("ASSIST")); err != nil {
		t.Fatal(err)
	}
	if ctx.AssistOpen {
		t.Fatal("AssistOpen still set after one Esc")
	}
}

func TestASSISTNavigationSubmenu(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}
	// Right to Create, Down open, letter D jumps? Create items Database/From.
	// Down opens Set Up submenu, Esc back, Esc leave.
	ctx.Stdin = strings.NewReader(
		"\x1b[C" + // Create
			"\x1b[B" + // open
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
	in := "\x1b[B" + string(rune(term.KeyEnter)) + "c" + string(rune(term.KeyEnter)) +
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

func TestASSISTDirFromTools(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "c.dbf"), []dbf.Field{
		{Name: "N", Type: 'C', Length: 1},
	}, nil)
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	ctx.Stderr = &bytes.Buffer{}
	mux := NewMux()
	// Tools is index 7: Right×7, Down, Enter Dir, Esc Esc
	in := strings.Repeat("\x1b[C", 7) +
		"\x1b[B" + string(rune(term.KeyEnter)) +
		string(rune(term.KeyEsc)) + string(rune(term.KeyEsc))
	ctx.Stdin = strings.NewReader(in)
	if err := mux.Dispatch(ctx, command.Split("ASSIST")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "c.dbf") {
		t.Fatalf("DIR missing c.dbf: %q", out.String())
	}
}

func TestASSISTListFromRetrieve(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "c.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 8},
	}, []dbf.Record{{Values: []dbf.Value{{Str: "Ada     "}}}})
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	ctx.Stderr = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE c"))
	// Retrieve is index 4: Right×4, Down, Enter List, Esc Esc
	in := strings.Repeat("\x1b[C", 4) +
		"\x1b[B" + string(rune(term.KeyEnter)) +
		string(rune(term.KeyEsc)) + string(rune(term.KeyEsc))
	ctx.Stdin = strings.NewReader(in)
	if err := mux.Dispatch(ctx, command.Split("ASSIST")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Ada") {
		t.Fatalf("LIST output missing Ada: %q", out.String())
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
	in := strings.Repeat("\x1b[C", 3) +
		"\x1b[B" + string(rune(term.KeyEnter)) + "2" + string(rune(term.KeyEnter)) +
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
	in := "\x1b[B" + string(rune(term.KeyEnter)) + string(rune(term.KeyEsc)) +
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

func TestASSISTHelpAndExitQuit(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	ctx.Stderr = &bytes.Buffer{}
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("HELP ASSIST")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "not available") {
		t.Fatalf("HELP still refuses: %q", out.String())
	}
	// Exit category: Right×8, Down, Down (Quit is 2nd item), Enter
	in := strings.Repeat("\x1b[C", 8) +
		"\x1b[B" + "\x1b[B" + string(rune(term.KeyEnter))
	ctx.Stdin = strings.NewReader(in)
	err := mux.Dispatch(ctx, command.Split("ASSIST"))
	if err != ErrQuit {
		t.Fatalf("want ErrQuit got %v", err)
	}
}

func TestASSISTModifyStructureTakesScreen(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "c.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{{Values: []dbf.Value{{Str: "ANN"}, {Num: 10}}}})
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	ctx.Stderr = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE c"))
	// Modify is index 6: Right×6, Down, Enter Structure; keep NAME only; Esc leave Assist.
	in := strings.Repeat("\x1b[C", 6) +
		"\x1b[B" + string(rune(term.KeyEnter)) +
		"NAME\nC\n4\n0\n\n" +
		string(rune(term.KeyEsc)) + string(rune(term.KeyEsc))
	ctx.Stdin = strings.NewReader(in)
	if err := mux.Dispatch(ctx, command.Split("ASSIST")); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	idxClear := strings.Index(got, "\x1b[2J\x1b[H")
	idxStruct := strings.Index(got, "Current structure:")
	if idxClear < 0 || idxStruct < 0 || idxClear > idxStruct {
		t.Fatalf("MODIFY STRUCTURE must clear before listing fields: clear=%d struct=%d out=%q", idxClear, idxStruct, got)
	}
	// Prompt lines must not be glued on one visual row after the status chrome.
	if strings.Contains(got, "Rec:") && strings.Contains(got[strings.Index(got, "Rec:"):], "Current structure:") {
		fragment := got[strings.Index(got, "Rec:"):]
		if i := strings.Index(fragment, "Current structure:"); i > 0 && i < 40 {
			t.Fatalf("structure listing glued to Assist status: %q", fragment[:min(80, len(fragment))])
		}
	}
	a := ctx.Active()
	if len(a.Table.Header.Fields) != 1 || a.Table.Header.Fields[0].Name != "NAME" {
		t.Fatalf("fields %+v", a.Table.Header.Fields)
	}
}

func TestASSISTLeaveClearsTerminal(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	ctx.Stderr = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader(string(rune(term.KeyEsc)))
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("ASSIST")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\x1b[2J\x1b[H") {
		t.Fatalf("leave did not clear terminal: %q", out.String())
	}
}

func TestEXITQuitsSession(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}
	if err := NewMux().Dispatch(ctx, command.Split("EXIT")); err != ErrQuit {
		t.Fatalf("want ErrQuit got %v", err)
	}
}
