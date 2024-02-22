package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestSplitInlineCommandsRespectsQuotes(t *testing.T) {
	got := splitInlineCommands(`STORE "a;b" TO x; STORE 1 TO y`)
	if len(got) != 2 {
		t.Fatalf("expected 2 commands, got %d: %v", len(got), got)
	}
	if strings.TrimSpace(got[0]) != `STORE "a;b" TO x` {
		t.Fatalf("unexpected first command: %q", got[0])
	}
	if strings.TrimSpace(got[1]) != "STORE 1 TO y" {
		t.Fatalf("unexpected second command: %q", got[1])
	}
}

func TestRunInlineExecutesCommands(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	err := RunInline(ctx, "STORE 42 TO answer; STORE 7 TO other")
	if err != nil {
		t.Fatalf("RunInline: %v", err)
	}

	val, ok := ctx.Variables.Get("answer")
	if !ok {
		t.Fatal("expected answer variable")
	}
	if f, ok := val.(float64); !ok || f != 42 {
		t.Fatalf("answer = %v (%T), want 42", val, val)
	}
	other, ok := ctx.Variables.Get("other")
	if !ok {
		t.Fatal("expected other variable")
	}
	if f, ok := other.(float64); !ok || f != 7 {
		t.Fatalf("other = %v (%T), want 7", other, other)
	}
}

func TestRunInlineReturnsFirstCommandError(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	err := RunInline(ctx, "USE missing; STORE 1 TO x")
	if err == nil || !strings.Contains(err.Error(), "Could not open file") {
		t.Fatalf("expected USE error, got %v", err)
	}
	if _, ok := ctx.Variables.Get("x"); ok {
		t.Fatal("expected second command to be skipped after error")
	}
}

func TestRunInlineStopsOnQuit(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	err := RunInline(ctx, "QUIT; STORE 1 TO x")
	if err != nil {
		t.Fatalf("RunInline: %v", err)
	}
	if _, ok := ctx.Variables.Get("x"); ok {
		t.Fatal("expected QUIT to stop further commands")
	}
}
