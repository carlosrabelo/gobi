package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestDispatchWaitPausesAndShowsWaiting(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out
	ctx.Stdin = strings.NewReader("x")

	if err := commandMux.Dispatch(ctx, Command{Verb: "WAIT"}); err != nil {
		t.Fatalf("WAIT failed: %v", err)
	}
	if !strings.Contains(out.String(), "WAITING") {
		t.Fatalf("expected WAITING marker, got %q", out.String())
	}
}

func TestDispatchWaitStoresPressedKey(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("y")

	if err := commandMux.Dispatch(ctx, Command{Verb: "WAIT", ToClause: "KEY"}); err != nil {
		t.Fatalf("WAIT TO failed: %v", err)
	}

	val, ok := ctx.Variables.Get("KEY")
	if !ok {
		t.Fatal("expected variable KEY to be stored")
	}
	if val != "y" {
		t.Fatalf("expected KEY = %q, got %v", "y", val)
	}
}

func TestDispatchWaitStoresEmptyStringForControlKey(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("\n")

	if err := commandMux.Dispatch(ctx, Command{Verb: "WAIT", ToClause: "KEY"}); err != nil {
		t.Fatalf("WAIT TO failed: %v", err)
	}
	if val, _ := ctx.Variables.Get("KEY"); val != "" {
		t.Fatalf("expected empty string for control key, got %v", val)
	}
}

func TestDispatchWaitConsumesSingleByteOnly(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("ab")

	if err := commandMux.Dispatch(ctx, Command{Verb: "WAIT", ToClause: "K1"}); err != nil {
		t.Fatalf("first WAIT failed: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "WAIT", ToClause: "K2"}); err != nil {
		t.Fatalf("second WAIT failed: %v", err)
	}

	if val, _ := ctx.Variables.Get("K1"); val != "a" {
		t.Fatalf("expected K1 = %q, got %v", "a", val)
	}
	if val, _ := ctx.Variables.Get("K2"); val != "b" {
		t.Fatalf("expected K2 = %q, got %v", "b", val)
	}
}

func TestDispatchWaitEOFLeavesVariableUnset(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("")

	if err := commandMux.Dispatch(ctx, Command{Verb: "WAIT", ToClause: "NOPE"}); err != nil {
		t.Fatalf("expected silent EOF, got %v", err)
	}
	if _, ok := ctx.Variables.Get("NOPE"); ok {
		t.Fatal("expected variable to remain unset on EOF")
	}
}

func TestDispatchWaitRejectsInvalidVariableName(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("x")

	err := commandMux.Dispatch(ctx, Command{Verb: "WAIT", ToClause: "9BAD"})
	if err == nil || !strings.Contains(err.Error(), "Invalid WAIT target") {
		t.Fatalf("expected invalid target error, got %v", err)
	}
}

func TestDispatchWaitRejectsExtraArguments(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("x")

	err := commandMux.Dispatch(ctx, Command{Verb: "WAIT", Args: "FOREVER"})
	if err == nil || !strings.Contains(err.Error(), "WAIT accepts only") {
		t.Fatalf("expected extra arguments error, got %v", err)
	}
}
