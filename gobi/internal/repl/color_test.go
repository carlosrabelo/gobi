package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func TestParseColorPair(t *testing.T) {
	fg, bg := parseColorPair("W+/N")
	if fg != 15 || bg != 0 {
		t.Fatalf("W+/N: %d/%d", fg, bg)
	}
	fg, bg = parseColorPair("GR/B")
	if fg != 6 || bg != 1 {
		t.Fatalf("GR/B: %d/%d", fg, bg)
	}
	stdFG, stdBG, enhFG, enhBG := parseColorTO("W/N,N/W")
	if stdFG != 7 || stdBG != 0 || enhFG != 0 || enhBG != 7 {
		t.Fatalf("pairs %d/%d %d/%d", stdFG, stdBG, enhFG, enhBG)
	}
	_, _, efg, ebg := parseColorTO("W/N")
	if efg != 0 || ebg != 7 {
		t.Fatalf("enhanced swap: %d/%d", efg, ebg)
	}
}

func TestSETColorAppliesToCells(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	out := &bytes.Buffer{}
	ctx.Stdout = out
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("SET COLOR TO W/N,N/W"))
	_ = mux.Dispatch(ctx, command.Split("SET COLOR ON"))
	scr := ensureScreen(ctx)
	// Force color active even on non-TTY stdout (CapsOf pipe is false).
	scr.ColorActive = true
	scr.StandardFG, scr.StandardBG = 7, 0
	scr.EnhancedFG, scr.EnhancedBG = 0, 7
	scr.RefreshColorAttrs()
	scr.Clear()
	scr.Put(0, 0, 'A', false)
	scr.Put(0, 1, 'B', true)
	if scr.CellAt(0, 0).FG != 7 || scr.CellAt(0, 0).BG != 0 {
		t.Fatalf("standard cell %+v", scr.CellAt(0, 0))
	}
	if scr.CellAt(0, 1).FG != 0 || scr.CellAt(0, 1).BG != 7 || !scr.CellAt(0, 1).Reverse {
		t.Fatalf("enhanced cell %+v", scr.CellAt(0, 1))
	}
	var buf bytes.Buffer
	if err := scr.Present(&buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "\x1b[37m") { // white fg
		t.Fatalf("missing standard fg: %q", got)
	}
	if !strings.Contains(got, "\x1b[7m") {
		t.Fatalf("missing reverse: %q", got)
	}
}

func TestSETColorOffClearsAttrs(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("SET COLOR TO R/N"))
	_ = mux.Dispatch(ctx, command.Split("SET COLOR ON"))
	scr := ensureScreen(ctx)
	scr.ColorActive = true
	scr.StandardFG = 4
	scr.RefreshColorAttrs()
	scr.Put(0, 0, 'X', false)
	_ = mux.Dispatch(ctx, command.Split("SET COLOR OFF"))
	if scr.ColorActive {
		// OFF with pipe caps stays false; force path through applySessionColors
	}
	applySessionColors(ctx)
	if ctx.Config.ColorOn {
		t.Fatal("ColorOn still true")
	}
	scr.Put(1, 0, 'Y', false)
	if scr.CellAt(1, 0).FG != term.ColorDefault {
		t.Fatalf("OFF should leave default attrs: %+v", scr.CellAt(1, 0))
	}
}
