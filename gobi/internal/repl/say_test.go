package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/internal/symbols"
)

func TestParseSayArgs(t *testing.T) {
	row, col, say, err := parseSayArgs(`2, 10 SAY "Hello"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row != "2" || col != "10" || say != `"Hello"` {
		t.Fatalf("unexpected parse result: row=%q col=%q say=%q", row, col, say)
	}
}

func TestParseSayArgsExpressionWithComma(t *testing.T) {
	_, _, say, err := parseSayArgs(`1, 5 SAY "Hi, " + NAME`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if say != `"Hi, " + NAME` {
		t.Fatalf("unexpected say expression: %q", say)
	}
}

func TestParseSayArgsMissingSAY(t *testing.T) {
	_, _, _, err := parseSayArgs(`2, 10`)
	if err == nil || !strings.Contains(err.Error(), "SAY") {
		t.Fatalf("expected SAY error, got %v", err)
	}
}

func TestParseSayArgsBadCoordinates(t *testing.T) {
	_, _, _, err := parseSayArgs(`2 SAY "Hello"`)
	if err == nil || !strings.Contains(err.Error(), "coordinates") {
		t.Fatalf("expected coordinates error, got %v", err)
	}
}

func TestSplitAtKeywordInsideStringIgnored(t *testing.T) {
	before, after, ok := splitAtKeyword(`"SAY hi", 1`, "SAY")
	if ok {
		t.Fatalf("expected no keyword match inside string, got before=%q after=%q", before, after)
	}
}

func TestDispatchAtSayWritesToScreenBuffer(t *testing.T) {
	var stdout bytes.Buffer
	ctx := testCtx()
	ctx.Stdout = &stdout

	if err := ctx.Variables.Set("TITLE", "Gobi"); err != nil {
		t.Fatalf("set variable: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{
		Verb: "@",
		Args: `3, 5 SAY TITLE`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := screenTextAt(ctx, 3, 5, 4)
	if got != "Gobi" {
		t.Fatalf("expected screen text %q, got %q", "Gobi", got)
	}
	if stdout.Len() == 0 {
		t.Fatal("expected screen presentation output")
	}
}

func TestDispatchAtSayEvaluatesNumericCoordinates(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := symbols.ValidateName("R"); err != nil {
		t.Fatal(err)
	}
	if err := ctx.Variables.Set("R", 1); err != nil {
		t.Fatalf("set variable: %v", err)
	}
	if err := ctx.Variables.Set("C", 4); err != nil {
		t.Fatalf("set variable: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{
		Verb: "@",
		Args: `R, C SAY "OK"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := screenTextAt(ctx, 1, 4, 2); got != "OK" {
		t.Fatalf("expected OK at (1,4), got %q", got)
	}
}

func TestDispatchAtSayNegativeCoordinates(t *testing.T) {
	ctx := testCtx()
	if err := ctx.Variables.Set("N", -1); err != nil {
		t.Fatalf("set variable: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{
		Verb: "@",
		Args: `N, 0 SAY "X"`,
	})
	if err == nil || !strings.Contains(err.Error(), "non-negative") {
		t.Fatalf("expected non-negative coordinate error, got %v", err)
	}
}

func screenTextAt(ctx *context.Context, row, col, length int) string {
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteByte(ctx.Screen.At(row, col+i))
	}
	return b.String()
}
