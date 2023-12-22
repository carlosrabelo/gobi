package repl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/script"
)

func TestRunProgramCancelHaltsNestedScripts(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "child.prg"), []byte("STORE 42 TO answer\nCANCEL\nSTORE 999 TO skipped\n"), 0644); err != nil {
		t.Fatalf("write child: %v", err)
	}

	parent, err := script.ParseSource(filepath.Join(tempDir, "parent.prg"), "DO child\nSTORE 1 TO done\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	if err := RunProgram(ctx, parent); err != nil {
		t.Fatalf("RunProgram: %v", err)
	}

	answer, ok := ctx.Variables.Get("ANSWER")
	if !ok || answer.(float64) != 42 {
		t.Fatalf("expected ANSWER=42, got %#v", answer)
	}
	if _, ok := ctx.Variables.Get("DONE"); ok {
		t.Fatal("expected parent script not to resume after CANCEL")
	}
	if _, ok := ctx.Variables.Get("SKIPPED"); ok {
		t.Fatal("expected child commands after CANCEL to be skipped")
	}
	if ctx.Script != nil {
		t.Fatal("expected script controller to be cleared")
	}
	if len(ctx.ExecutionStack) != 0 {
		t.Fatalf("expected execution stack cleared, got %#v", ctx.ExecutionStack)
	}
}

func TestRunProgramCancelAtTopLevel(t *testing.T) {
	prog, err := script.ParseSource("test.prg", "STORE 1 TO kept\nCANCEL\nSTORE 2 TO skipped\n")
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
		t.Fatal("expected SKIPPED to be ignored after CANCEL")
	}
}

func TestRunProgramThreeLevelCancel(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "grandchild.prg"), []byte("CANCEL\n"), 0644); err != nil {
		t.Fatalf("write grandchild: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "child.prg"), []byte("DO grandchild\nSTORE 2 TO level\n"), 0644); err != nil {
		t.Fatalf("write child: %v", err)
	}

	parent, err := script.ParseSource(filepath.Join(tempDir, "parent.prg"), "DO child\nSTORE 1 TO level\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	if err := RunProgram(ctx, parent); err != nil {
		t.Fatalf("RunProgram: %v", err)
	}

	if _, ok := ctx.Variables.Get("LEVEL"); ok {
		t.Fatal("expected no LEVEL assignment after nested CANCEL")
	}
}

func TestDispatchCancelClearsScriptState(t *testing.T) {
	ctx := testCtx()
	ctx.ExecutionStack = append(ctx.ExecutionStack, "/tmp/sample.prg")

	if err := commandMux.Dispatch(ctx, Command{Verb: "CANCEL"}); err != nil {
		t.Fatalf("CANCEL: %v", err)
	}
	if len(ctx.ExecutionStack) != 0 {
		t.Fatalf("expected execution stack cleared, got %#v", ctx.ExecutionStack)
	}
	if ctx.Script != nil {
		t.Fatal("expected script controller cleared")
	}
}
