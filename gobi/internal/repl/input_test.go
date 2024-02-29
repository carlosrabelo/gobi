package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func dispatchInput(t *testing.T, ctx *context.Context, typed, toClause, args string) error {
	t.Helper()
	ctx.Stdin = strings.NewReader(typed)
	return commandMux.Dispatch(ctx, Command{Verb: "INPUT", Args: args, ToClause: toClause})
}

func TestDispatchInputStoresNumericValue(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out

	if err := dispatchInput(t, ctx, "42\n", "AGE", "'Enter age'"); err != nil {
		t.Fatalf("INPUT failed: %v", err)
	}

	if !strings.Contains(out.String(), "Enter age: ") {
		t.Fatalf("expected prompt in output, got %q", out.String())
	}
	val, ok := ctx.Variables.Get("AGE")
	if !ok {
		t.Fatal("expected variable AGE to be stored")
	}
	if n, isNum := val.(float64); !isNum || n != 42 {
		t.Fatalf("expected AGE = 42 as float64, got %v (%T)", val, val)
	}
}

func TestDispatchInputEvaluatesArithmeticExpression(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := dispatchInput(t, ctx, "5 * 3 + 1\n", "TOTAL", ""); err != nil {
		t.Fatalf("INPUT failed: %v", err)
	}
	if val, _ := ctx.Variables.Get("TOTAL"); val != float64(16) {
		t.Fatalf("expected TOTAL = 16, got %v", val)
	}
}

func TestDispatchInputStoresLogicalValue(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := dispatchInput(t, ctx, ".T.\n", "FLAG", ""); err != nil {
		t.Fatalf("INPUT failed: %v", err)
	}
	if val, _ := ctx.Variables.Get("FLAG"); val != true {
		t.Fatalf("expected FLAG = true, got %v", val)
	}
}

func TestDispatchInputStoresStringValue(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := dispatchInput(t, ctx, "'hello'\n", "MSG", ""); err != nil {
		t.Fatalf("INPUT failed: %v", err)
	}
	if val, _ := ctx.Variables.Get("MSG"); val != "hello" {
		t.Fatalf("expected MSG = %q, got %v", "hello", val)
	}
}

func TestDispatchInputResolvesMemoryVariables(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	if err := ctx.Variables.Set("BASE", float64(10)); err != nil {
		t.Fatalf("set BASE: %v", err)
	}

	if err := dispatchInput(t, ctx, "BASE + 5\n", "RESULT", ""); err != nil {
		t.Fatalf("INPUT failed: %v", err)
	}
	if val, _ := ctx.Variables.Get("RESULT"); val != float64(15) {
		t.Fatalf("expected RESULT = 15, got %v", val)
	}
}

func TestDispatchInputRequiresToClause(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("42\n")

	err := commandMux.Dispatch(ctx, Command{Verb: "INPUT", Args: "'Age?'"})
	if err == nil || !strings.Contains(err.Error(), "INPUT requires TO") {
		t.Fatalf("expected missing TO error, got %v", err)
	}
}

func TestDispatchInputRejectsEmptyLine(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	err := dispatchInput(t, ctx, "\n", "X", "")
	if err == nil || !strings.Contains(err.Error(), "INPUT requires an expression") {
		t.Fatalf("expected empty expression error, got %v", err)
	}
}

func TestDispatchInputReportsEvaluationError(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	err := dispatchInput(t, ctx, "NOPE + 1\n", "X", "")
	if err == nil || !strings.Contains(err.Error(), "Evaluation error in INPUT") {
		t.Fatalf("expected evaluation error, got %v", err)
	}
	if _, ok := ctx.Variables.Get("X"); ok {
		t.Fatal("expected variable to remain unset on evaluation error")
	}
}

func TestDispatchInputEOFLeavesVariableUnset(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := dispatchInput(t, ctx, "", "NOPE", ""); err != nil {
		t.Fatalf("expected silent EOF, got %v", err)
	}
	if _, ok := ctx.Variables.Get("NOPE"); ok {
		t.Fatal("expected variable to remain unset on EOF")
	}
}
