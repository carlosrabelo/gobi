package repl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/script"
)

func TestRunProgramExecutesCommands(t *testing.T) {
	prog, err := script.ParseSource("test.prg", "STORE 10 TO counter\nSTORE 20 TO total\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctx := testCtx()
	if err := RunProgram(ctx, prog); err != nil {
		t.Fatalf("RunProgram: %v", err)
	}

	val, ok := ctx.Variables.Get("COUNTER")
	if !ok || val.(float64) != 10 {
		t.Fatalf("expected COUNTER=10, got %#v", val)
	}
	total, ok := ctx.Variables.Get("TOTAL")
	if !ok || total.(float64) != 20 {
		t.Fatalf("expected TOTAL=20, got %#v", total)
	}
	if ctx.Script != nil {
		t.Fatal("expected script controller to be cleared")
	}
}

func TestRunProgramStopsAtReturn(t *testing.T) {
	prog, err := script.ParseSource("test.prg", "STORE 1 TO kept\nRETURN\nSTORE 2 TO skipped\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctx := testCtx()
	if err := RunProgram(ctx, prog); err != nil {
		t.Fatalf("RunProgram: %v", err)
	}

	if _, ok := ctx.Variables.Get("KEPT"); !ok {
		t.Fatal("expected KEPT to be set")
	}
	if _, ok := ctx.Variables.Get("SKIPPED"); ok {
		t.Fatal("expected SKIPPED to be ignored after RETURN")
	}
}

func TestRunProgramSetsInstructionPointer(t *testing.T) {
	prog, err := script.ParseSource("test.prg", "STORE 1 TO x\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctx := testCtx()
	ctrlStarted := false
	originalDispatch := commandMux.handlers["STORE"]
	commandMux.handlers["STORE"] = func(ctx *context.Context, cmd Command) error {
		if ctx.Script == nil {
			t.Fatal("expected active script controller during execution")
		}
		if ctx.Script.Index() != 0 {
			t.Fatalf("expected instruction index 0 during first command, got %d", ctx.Script.Index())
		}
		ctrlStarted = true
		return originalDispatch(ctx, cmd)
	}
	defer func() {
		commandMux.handlers["STORE"] = originalDispatch
	}()

	if err := RunProgram(ctx, prog); err != nil {
		t.Fatalf("RunProgram: %v", err)
	}
	if !ctrlStarted {
		t.Fatal("expected STORE handler to run")
	}
}

func TestDispatchDoExecutesScript(t *testing.T) {
	tempDir := t.TempDir()
	content := "STORE 42 TO answer\nRETURN\n"
	if err := os.WriteFile(filepath.Join(tempDir, "run.prg"), []byte(content), 0644); err != nil {
		t.Fatalf("write prg: %v", err)
	}

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir

	if err := commandMux.Dispatch(ctx, Command{Verb: "DO", Args: "run"}); err != nil {
		t.Fatalf("DO: %v", err)
	}

	val, ok := ctx.Variables.Get("ANSWER")
	if !ok || val.(float64) != 42 {
		t.Fatalf("expected ANSWER=42, got %#v", val)
	}
}

func TestRunProgramTracksExecutionStack(t *testing.T) {
	prog, err := script.ParseSource("/tmp/sample.prg", "STORE 1 TO x\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctx := testCtx()
	stackSeen := false
	originalStore := commandMux.handlers["STORE"]
	commandMux.handlers["STORE"] = func(ctx *context.Context, cmd Command) error {
		if len(ctx.ExecutionStack) != 1 || ctx.ExecutionStack[0] != "/tmp/sample.prg" {
			t.Fatalf("unexpected execution stack: %#v", ctx.ExecutionStack)
		}
		stackSeen = true
		return originalStore(ctx, cmd)
	}
	defer func() {
		commandMux.handlers["STORE"] = originalStore
	}()

	if err := RunProgram(ctx, prog); err != nil {
		t.Fatalf("RunProgram: %v", err)
	}
	if !stackSeen {
		t.Fatal("expected STORE handler during script run")
	}
	if len(ctx.ExecutionStack) != 0 {
		t.Fatalf("expected execution stack cleared, got %#v", ctx.ExecutionStack)
	}
}

func TestRunProgramIfTrueBranch(t *testing.T) {
	source := "IF .T.\nSTORE 1 TO taken\nENDIF\nSTORE 2 TO always\n"
	prog, err := script.ParseSource("test.prg", source)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctx := testCtx()
	if err := RunProgram(ctx, prog); err != nil {
		t.Fatalf("RunProgram: %v", err)
	}

	taken, ok := ctx.Variables.Get("TAKEN")
	if !ok || taken.(float64) != 1 {
		t.Fatalf("expected TAKEN=1, got %#v", taken)
	}
	always, ok := ctx.Variables.Get("ALWAYS")
	if !ok || always.(float64) != 2 {
		t.Fatalf("expected ALWAYS=2, got %#v", always)
	}
}

func TestRunProgramIfFalseBranchSkipped(t *testing.T) {
	source := "IF .F.\nSTORE 1 TO skipped\nENDIF\nSTORE 2 TO kept\n"
	prog, err := script.ParseSource("test.prg", source)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctx := testCtx()
	if err := RunProgram(ctx, prog); err != nil {
		t.Fatalf("RunProgram: %v", err)
	}

	if _, ok := ctx.Variables.Get("SKIPPED"); ok {
		t.Fatal("expected SKIPPED branch not to run")
	}
	kept, ok := ctx.Variables.Get("KEPT")
	if !ok || kept.(float64) != 2 {
		t.Fatalf("expected KEPT=2, got %#v", kept)
	}
}

func TestRunProgramIfElseBranch(t *testing.T) {
	source := "IF .F.\nSTORE 1 TO false_branch\nELSE\nSTORE 2 TO else_branch\nENDIF\n"
	prog, err := script.ParseSource("test.prg", source)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctx := testCtx()
	if err := RunProgram(ctx, prog); err != nil {
		t.Fatalf("RunProgram: %v", err)
	}

	if _, ok := ctx.Variables.Get("FALSE_BRANCH"); ok {
		t.Fatal("expected false branch not to run")
	}
	elseBranch, ok := ctx.Variables.Get("ELSE_BRANCH")
	if !ok || elseBranch.(float64) != 2 {
		t.Fatalf("expected ELSE_BRANCH=2, got %#v", elseBranch)
	}
}

func TestRunProgramNestedIf(t *testing.T) {
	source := "STORE 0 TO result\nIF .T.\nIF .T.\nSTORE 1 TO result\nELSE\nSTORE 2 TO result\nENDIF\nELSE\nSTORE 3 TO result\nENDIF\n"
	prog, err := script.ParseSource("test.prg", source)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctx := testCtx()
	if err := RunProgram(ctx, prog); err != nil {
		t.Fatalf("RunProgram: %v", err)
	}

	result, ok := ctx.Variables.Get("RESULT")
	if !ok || result.(float64) != 1 {
		t.Fatalf("expected RESULT=1, got %#v", result)
	}
}
