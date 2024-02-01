package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func TestSyncScreenSizeKeepsDefaultWithoutTerminal(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = &bytes.Buffer{}

	syncScreenSize(ctx)

	if ctx.Screen.Cols() != term.DefaultCols || ctx.Screen.Rows() != term.DefaultRows {
		t.Fatalf("expected default 80x24 without terminal, got %dx%d",
			ctx.Screen.Cols(), ctx.Screen.Rows())
	}
}

func TestSetScreenDefaultPinsClassicGeometry(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Screen.Resize(120, 40)

	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "SCREEN DEFAULT"}); err != nil {
		t.Fatalf("SET SCREEN DEFAULT failed: %v", err)
	}

	if ctx.Config.ScreenAuto {
		t.Fatal("expected ScreenAuto disabled after SET SCREEN DEFAULT")
	}
	if ctx.Screen.Cols() != 80 || ctx.Screen.Rows() != 24 {
		t.Fatalf("expected 80x24 after SET SCREEN DEFAULT, got %dx%d",
			ctx.Screen.Cols(), ctx.Screen.Rows())
	}

	// Adaptive sync must not override the pinned geometry.
	syncScreenSize(ctx)
	if ctx.Screen.Cols() != 80 || ctx.Screen.Rows() != 24 {
		t.Fatalf("expected pinned 80x24 after sync, got %dx%d",
			ctx.Screen.Cols(), ctx.Screen.Rows())
	}
}

func TestSetScreenAutoRestoresAdaptiveMode(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Config.ScreenAuto = false

	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "SCREEN AUTO"}); err != nil {
		t.Fatalf("SET SCREEN AUTO failed: %v", err)
	}
	if !ctx.Config.ScreenAuto {
		t.Fatal("expected ScreenAuto enabled after SET SCREEN AUTO")
	}
}

func TestSetScreenRejectsInvalidArgument(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "SCREEN WIDE"})
	if err == nil || !strings.Contains(err.Error(), "SET SCREEN requires AUTO or DEFAULT") {
		t.Fatalf("expected invalid argument error, got %v", err)
	}

	err = commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "SCREEN"})
	if err == nil || !strings.Contains(err.Error(), "SET SCREEN requires AUTO or DEFAULT") {
		t.Fatalf("expected missing argument error, got %v", err)
	}
}

func TestSetScreenTalkReportsGeometry(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out

	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "SCREEN DEFAULT"}); err != nil {
		t.Fatalf("SET SCREEN DEFAULT failed: %v", err)
	}
	if !strings.Contains(out.String(), "SCREEN DEFAULT (80x24)") {
		t.Fatalf("expected geometry feedback, got %q", out.String())
	}
}

func TestPaintScreenTextPreservesCursor(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out

	if err := paintScreenText(ctx, 5, 10, "HELLO"); err != nil {
		t.Fatalf("paintScreenText failed: %v", err)
	}

	got := out.String()
	want := "\0337\033[6;11HHELLO\0338"
	if got != want {
		t.Fatalf("paint sequence = %q, want %q", got, want)
	}
}

func TestPaintScreenTextClipsToScreenBounds(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out

	if err := paintScreenText(ctx, 0, 78, "TOOLONG"); err != nil {
		t.Fatalf("paintScreenText failed: %v", err)
	}
	if !strings.Contains(out.String(), "TO") || strings.Contains(out.String(), "TOOL") {
		t.Fatalf("expected text clipped at column 80, got %q", out.String())
	}

	out.Reset()
	if err := paintScreenText(ctx, 99, 0, "OFF"); err != nil {
		t.Fatalf("paintScreenText failed: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no output for off-screen row, got %q", out.String())
	}
}

func TestReturnToConsoleIsNoOpWithoutTerminal(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out
	ctx.Stdin = &bytes.Buffer{}

	if err := returnToConsole(ctx); err != nil {
		t.Fatalf("returnToConsole failed: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no output without terminal, got %q", out.String())
	}
}

func TestDisplayPageSizeFollowsScreenRows(t *testing.T) {
	ctx := testCtx()

	if got := displayPageSize(ctx); got != 20 {
		t.Fatalf("expected classic page size 20 on 80x24, got %d", got)
	}

	ctx.Screen.Resize(80, 40)
	if got := displayPageSize(ctx); got != 36 {
		t.Fatalf("expected page size 36 on 80x40, got %d", got)
	}

	ctx.Screen = nil
	if got := displayPageSize(ctx); got != 20 {
		t.Fatalf("expected fallback page size 20 without screen, got %d", got)
	}
}

func TestClampScreenSizeEnforcesMinimums(t *testing.T) {
	cols, rows := clampScreenSize(20, 5)
	if cols != minScreenCols || rows != minScreenRows {
		t.Fatalf("expected %dx%d, got %dx%d", minScreenCols, minScreenRows, cols, rows)
	}

	cols, rows = clampScreenSize(132, 50)
	if cols != 132 || rows != 50 {
		t.Fatalf("expected 132x50 preserved, got %dx%d", cols, rows)
	}
}
