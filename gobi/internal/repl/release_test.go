package repl

import (
	"testing"
)

func TestDispatchReleaseSingle(t *testing.T) {
	ctx := testCtx()

	if err := ctx.Variables.Set("MESSAGE", "hello"); err != nil {
		t.Fatalf("set MESSAGE: %v", err)
	}
	if err := ctx.Variables.Set("ANOTHER", float64(33)); err != nil {
		t.Fatalf("set ANOTHER: %v", err)
	}
	if err := ctx.Variables.Set("THIRD", float64(3)); err != nil {
		t.Fatalf("set THIRD: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "RELEASE", Args: "ANOTHER"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.Variables.Len() != 2 {
		t.Fatalf("expected 2 variables, got %d", ctx.Variables.Len())
	}
	if _, ok := ctx.Variables.Get("ANOTHER"); ok {
		t.Fatal("ANOTHER should have been released")
	}
	if _, ok := ctx.Variables.Get("MESSAGE"); !ok {
		t.Fatal("MESSAGE should remain")
	}
}

func TestDispatchReleaseMultiple(t *testing.T) {
	ctx := testCtx()

	if err := ctx.Variables.Set("A", float64(1)); err != nil {
		t.Fatalf("set A: %v", err)
	}
	if err := ctx.Variables.Set("B", float64(2)); err != nil {
		t.Fatalf("set B: %v", err)
	}
	if err := ctx.Variables.Set("C", float64(3)); err != nil {
		t.Fatalf("set C: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "RELEASE", Args: "A, C"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.Variables.Len() != 1 {
		t.Fatalf("expected 1 variable, got %d", ctx.Variables.Len())
	}
	if _, ok := ctx.Variables.Get("B"); !ok {
		t.Fatal("B should remain")
	}
}

func TestDispatchReleaseAll(t *testing.T) {
	ctx := testCtx()

	if err := ctx.Variables.Set("ONE", float64(1)); err != nil {
		t.Fatalf("set ONE: %v", err)
	}
	if err := ctx.Variables.Set("TWO", float64(2)); err != nil {
		t.Fatalf("set TWO: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "RELEASE", Args: "ALL"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.Variables.Len() != 0 {
		t.Fatalf("expected empty registry, got %d variables", ctx.Variables.Len())
	}
}

func TestDispatchReleaseMissingVariable(t *testing.T) {
	ctx := testCtx()

	if err := ctx.Variables.Set("KEEP", float64(1)); err != nil {
		t.Fatalf("set KEEP: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "RELEASE", Args: "MISSING"})
	if err != nil {
		t.Fatalf("unexpected error releasing missing variable: %v", err)
	}

	if ctx.Variables.Len() != 1 {
		t.Fatalf("expected KEEP to remain, got %d variables", ctx.Variables.Len())
	}
}

func TestDispatchReleaseNoArgs(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "RELEASE"})
	if err == nil {
		t.Fatal("expected error for RELEASE with no args")
	}
}

func TestDispatchReleaseInvalidName(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "RELEASE", Args: "1BAD"})
	if err == nil {
		t.Fatal("expected error for invalid variable name")
	}
}
