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
