package repl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDispatchDoExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "deptlist.prg")
	if err := os.WriteFile(path, []byte("STORE 1 TO x\r\n"), 0644); err != nil {
		t.Fatalf("write prg: %v", err)
	}

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir

	err := commandMux.Dispatch(ctx, Command{Verb: "DO", Args: "deptlist"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := ctx.Variables.Get("X")
	if !ok || val.(float64) != 1 {
		t.Fatalf("expected X=1 after DO, got %#v", val)
	}
}

func TestDispatchDoWithExtension(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "run.prg")
	if err := os.WriteFile(path, []byte("\r\n"), 0644); err != nil {
		t.Fatalf("write prg: %v", err)
	}

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir

	err := commandMux.Dispatch(ctx, Command{Verb: "DO", Args: "run.prg"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatchDoMissingFile(t *testing.T) {
	ctx := testCtx()
	ctx.Config.DefaultDir = t.TempDir()

	err := commandMux.Dispatch(ctx, Command{Verb: "DO", Args: "missing"})
	if err == nil {
		t.Fatal("expected error for missing command file")
	}
	if err.Error() != "*** Command file not found" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatchDoNoArgs(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "DO"})
	if err == nil {
		t.Fatal("expected error for DO with no args")
	}
}

func TestDispatchDoWhileInteractiveFails(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "DO", Args: "WHILE .T."})
	if err == nil {
		t.Fatal("expected error for DO WHILE")
	}
	if err.Error() != "*** DO WHILE is only valid in command files" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatchDoCaseInteractiveFails(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "DO", Args: "CASE"})
	if err == nil || err.Error() != "*** DO CASE is only valid in command files" {
		t.Fatalf("unexpected error: %v", err)
	}
}
