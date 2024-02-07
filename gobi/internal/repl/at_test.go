package repl

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func TestAtSAY(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split(`@ 0, 0 SAY "Hi"`)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Hi") {
		t.Fatalf("SAY output: %q", out.String())
	}
	if ctx.Screen.CellAt(0, 0).Ch != 'H' {
		t.Fatalf("cell %q", ctx.Screen.CellAt(0, 0).Ch)
	}
}

func TestSETDevicePrintSAY(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("SET DEVICE TO PRINT"))
	if !strings.EqualFold(ctx.Config.Device, "PRINT") {
		t.Fatalf("device %q", ctx.Config.Device)
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split(`@ 0, 0 SAY "Hi"`)); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Hi" {
		t.Fatalf("print SAY: %q", out.String())
	}
	if ctx.Screen != nil && ctx.Screen.CellAt(0, 0).Ch == 'H' {
		t.Fatal("SAY wrote screen while DEVICE PRINT")
	}
}

func TestREADStoresField(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN "}}},
	})
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	// Type "BOB" then Enter on last (only) field.
	ctx.Stdin = strings.NewReader("\x7f\x7f\x7f\x7fBOB\r")
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("@ 0, 0 GET NAME")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("READ")); err != nil {
		t.Fatal(err)
	}
	a := ctx.Active()
	if err := a.LoadCurrent(); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(a.Record.Values[0].Str)
	if got != "BOB" {
		t.Fatalf("field after READ: %q", a.Record.Values[0].Str)
	}
	if len(mux.gets) != 0 {
		t.Fatal("gets not cleared")
	}
}

func TestSETEscapeOffIgnoresEscOnREAD(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN "}}},
	})
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))
	_ = mux.Dispatch(ctx, command.Split("SET ESCAPE OFF"))
	if ctx.Config.Escape {
		t.Fatal("ESCAPE still on")
	}
	// Esc ignored, then replace with BOB and Enter.
	ctx.Stdin = strings.NewReader("\x1b\x7f\x7f\x7f\x7fBOB\r")
	_ = mux.Dispatch(ctx, command.Split("@ 0, 0 GET NAME"))
	if err := mux.Dispatch(ctx, command.Split("READ")); err != nil {
		t.Fatal(err)
	}
	_ = ctx.Active().LoadCurrent()
	if strings.TrimSpace(ctx.Active().Record.Values[0].Str) != "BOB" {
		t.Fatalf("got %q", ctx.Active().Record.Values[0].Str)
	}
}

func TestSETDelimitersTOChars(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split(`SET DELIMITERS TO "[]"`)); err != nil {
		t.Fatal(err)
	}
	if ctx.Config.DelimChars != "[]" {
		t.Fatalf("DelimChars %q", ctx.Config.DelimChars)
	}
	_ = mux.Dispatch(ctx, command.Split("SET DELIMITERS ON"))
	_ = mux.Dispatch(ctx, command.Split(`STORE "AB" TO V`))
	if err := mux.Dispatch(ctx, command.Split("@ 0, 0 GET V")); err != nil {
		t.Fatal(err)
	}
	if ctx.Screen.CellAt(0, 0).Ch != '[' || ctx.Screen.CellAt(0, 11).Ch != ']' {
		t.Fatalf("pair delims %q %q", ctx.Screen.CellAt(0, 0).Ch, ctx.Screen.CellAt(0, 11).Ch)
	}
	err := mux.Dispatch(ctx, command.Split(`SET DELIMITERS TO "abc"`))
	if err == nil {
		t.Fatal("expected error for three characters")
	}
	if ctx.Config.DelimChars != "[]" {
		t.Fatalf("chars changed after error: %q", ctx.Config.DelimChars)
	}
	if err := mux.Dispatch(ctx, command.Split("SET DELIMITERS TO")); err != nil {
		t.Fatal(err)
	}
	if ctx.Config.DelimChars != ":" {
		t.Fatalf("empty TO should reset to colon: %q", ctx.Config.DelimChars)
	}
}

func TestSETDelimitersDrawAroundGET(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split(`STORE "AB" TO V`))
	_ = mux.Dispatch(ctx, command.Split("SET DELIMITERS ON"))
	if !ctx.Config.Delimiters {
		t.Fatal("DELIMITERS still off")
	}
	if err := mux.Dispatch(ctx, command.Split("@ 0, 0 GET V")); err != nil {
		t.Fatal(err)
	}
	if ctx.Screen.CellAt(0, 0).Ch != ':' {
		t.Fatalf("left delim %q", ctx.Screen.CellAt(0, 0).Ch)
	}
	if ctx.Screen.CellAt(0, 1).Ch != 'A' {
		t.Fatalf("field start %q", ctx.Screen.CellAt(0, 1).Ch)
	}
	// default width 10 for memory var → right delim at col 11
	if ctx.Screen.CellAt(0, 11).Ch != ':' {
		t.Fatalf("right delim %q", ctx.Screen.CellAt(0, 11).Ch)
	}
	_ = mux.Dispatch(ctx, command.Split("SET DELIMITERS OFF"))
	mux.gets = nil
	if err := mux.Dispatch(ctx, command.Split("@ 1, 0 GET V")); err != nil {
		t.Fatal(err)
	}
	if ctx.Screen.CellAt(1, 0).Ch == ':' {
		t.Fatal("DELIMITERS OFF still drew colon")
	}
}

func TestSETConfirmWaitsOnFullField(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 3},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "   "}}},
	})
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))

	// CONFIRM OFF (default): filling the only GET ends READ.
	ctx.Stdin = strings.NewReader("ABC")
	_ = mux.Dispatch(ctx, command.Split("@ 0, 0 GET NAME"))
	if err := mux.Dispatch(ctx, command.Split("READ")); err != nil {
		t.Fatal(err)
	}
	_ = ctx.Active().LoadCurrent()
	if strings.TrimSpace(ctx.Active().Record.Values[0].Str) != "ABC" {
		t.Fatalf("CONFIRM OFF: %q", ctx.Active().Record.Values[0].Str)
	}

	// CONFIRM ON: full field stays until Enter.
	_ = mux.Dispatch(ctx, command.Split("SET CONFIRM ON"))
	if !ctx.Config.Confirm {
		t.Fatal("CONFIRM still off")
	}
	_ = mux.Dispatch(ctx, command.Split("REPLACE NAME WITH [   ]"))
	ctx.Stdin = strings.NewReader("XYZ\r")
	_ = mux.Dispatch(ctx, command.Split("@ 0, 0 GET NAME"))
	if err := mux.Dispatch(ctx, command.Split("READ")); err != nil {
		t.Fatal(err)
	}
	_ = ctx.Active().LoadCurrent()
	if strings.TrimSpace(ctx.Active().Record.Values[0].Str) != "XYZ" {
		t.Fatalf("CONFIRM ON: %q", ctx.Active().Record.Values[0].Str)
	}
}

func TestREADSAVEKeepsGets(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split(`STORE "A" TO V`))
	if err := mux.Dispatch(ctx, command.Split(`@ 0, 0 GET V`)); err != nil {
		t.Fatal(err)
	}
	ctx.Stdin = strings.NewReader("\x7fB\r")
	if err := mux.Dispatch(ctx, command.Split("READ SAVE")); err != nil {
		t.Fatal(err)
	}
	if len(mux.gets) != 1 {
		t.Fatalf("READ SAVE cleared gets: %d", len(mux.gets))
	}
	v, _ := ctx.Symbols.Get("V")
	if v.String() != "B" {
		t.Fatalf("after SAVE read V=%q", v.String())
	}
	// Second READ without a new @ GET.
	ctx.Stdin = strings.NewReader("\x7fC\r")
	if err := mux.Dispatch(ctx, command.Split("READ")); err != nil {
		t.Fatal(err)
	}
	if len(mux.gets) != 0 {
		t.Fatal("plain READ should clear gets")
	}
	v, _ = ctx.Symbols.Get("V")
	if v.String() != "C" {
		t.Fatalf("second READ V=%q", v.String())
	}
}

func TestAtClearAndBox(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split(`@ 0, 0 SAY "ABC"`))
	_ = mux.Dispatch(ctx, command.Split(`@ 1, 0 SAY "XYZ"`))
	if err := mux.Dispatch(ctx, command.Split(`@ 0, 0 CLEAR TO 0, 1`)); err != nil {
		t.Fatal(err)
	}
	if ctx.Screen.CellAt(0, 0).Ch != ' ' || ctx.Screen.CellAt(0, 1).Ch != ' ' {
		t.Fatalf("CLEAR TO left %q %q", ctx.Screen.CellAt(0, 0).Ch, ctx.Screen.CellAt(0, 1).Ch)
	}
	if ctx.Screen.CellAt(0, 2).Ch != 'C' {
		t.Fatalf("CLEAR TO spilled: %q", ctx.Screen.CellAt(0, 2).Ch)
	}
	if err := mux.Dispatch(ctx, command.Split(`@ 0, 0 TO 2, 2`)); err != nil {
		t.Fatal(err)
	}
	for _, p := range [][2]int{{0, 0}, {0, 2}, {2, 0}, {2, 2}} {
		if ctx.Screen.CellAt(p[0], p[1]).Ch != '+' {
			t.Fatalf("corner %v = %q", p, ctx.Screen.CellAt(p[0], p[1]).Ch)
		}
	}
	if ctx.Screen.CellAt(0, 1).Ch != '-' || ctx.Screen.CellAt(1, 0).Ch != '|' {
		t.Fatalf("edges %q %q", ctx.Screen.CellAt(0, 1).Ch, ctx.Screen.CellAt(1, 0).Ch)
	}
	if err := mux.Dispatch(ctx, command.Split(`@ 3, 0 TO 3, 3 DOUBLE`)); err != nil {
		t.Fatal(err)
	}
	if ctx.Screen.CellAt(3, 1).Ch != '=' {
		t.Fatalf("DOUBLE h %q", ctx.Screen.CellAt(3, 1).Ch)
	}
}

func TestGETRangeRejectsKeepsOld(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split(`STORE 5 TO N`))
	// Out of RANGE 1,10: type 99 then Enter → restore 5; replace with 7.
	ctx.Stdin = strings.NewReader("\x7f99\r\x7f7\r")
	if err := mux.Dispatch(ctx, command.Split(`@ 0, 0 GET N RANGE 1, 10`)); err != nil {
		t.Fatal(err)
	}
	if !mux.gets[0].HasRange || !mux.gets[0].HasLo || !mux.gets[0].HasHi {
		t.Fatalf("range %+v", mux.gets[0])
	}
	if err := mux.Dispatch(ctx, command.Split("READ")); err != nil {
		t.Fatal(err)
	}
	v, _ := ctx.Symbols.Get("N")
	if v.Kind != expr.KindNumber || v.Num != 7 {
		t.Fatalf("N=%v", v)
	}
}

func TestPictureFunctionCharsOnGET(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split(`STORE "x" TO V`))
	// @A rejects digits; @! uppercases; typed "ab" (digit ignored) then Enter.
	ctx.Stdin = strings.NewReader("\x7fab1\r")
	if err := mux.Dispatch(ctx, command.Split(`@ 0, 0 GET V PICTURE "@A!"`)); err != nil {
		t.Fatal(err)
	}
	if !mux.gets[0].PicAlpha || !mux.gets[0].PicUpper {
		t.Fatalf("flags %+v", mux.gets[0])
	}
	if err := mux.Dispatch(ctx, command.Split("READ")); err != nil {
		t.Fatal(err)
	}
	v, ok := ctx.Symbols.Get("V")
	if !ok || v.String() != "AB" {
		t.Fatalf("V=%v ok=%v", v, ok)
	}

	_ = mux.Dispatch(ctx, command.Split(`STORE "" TO W`))
	ctx.Stdin = strings.NewReader("1234\r")
	if err := mux.Dispatch(ctx, command.Split(`@ 0, 0 GET W PICTURE "@R 99-99"`)); err != nil {
		t.Fatal(err)
	}
	if !mux.gets[0].PicR {
		t.Fatal("expected @R")
	}
	// Initial GET paint must show template literals.
	if cell := ctx.Screen.CellAt(0, 2); cell.Ch != '-' {
		t.Fatalf("@R GET display missing literal '-', got %q", cell.Ch)
	}
	if err := mux.Dispatch(ctx, command.Split("READ")); err != nil {
		t.Fatal(err)
	}
	w, _ := ctx.Symbols.Get("W")
	if w.String() != "1234" {
		t.Fatalf("@R stored %q", w.String())
	}

	_ = mux.Dispatch(ctx, command.Split(`STORE "" TO S`))
	if err := mux.Dispatch(ctx, command.Split(`@ 0, 0 GET S PICTURE "@S4"`)); err != nil {
		t.Fatal(err)
	}
	if mux.gets[0].Scroll != 4 || mux.gets[0].Width != 4 {
		t.Fatalf("scroll %+v", mux.gets[0])
	}
}

func TestAtGETLimit(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split(`STORE "x" TO V`))
	for i := 0; i < 128; i++ {
		if err := mux.Dispatch(ctx, command.Split("@ 0, 0 GET V")); err != nil {
			t.Fatalf("get %d: %v", i, err)
		}
	}
	if err := mux.Dispatch(ctx, command.Split("@ 0, 0 GET V")); err == nil {
		t.Fatal("expected 129th GET error")
	}
	if len(mux.gets) != 128 {
		t.Fatalf("pending %d", len(mux.gets))
	}
}
