package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/mem"
)

func TestDispatchSaveTo(t *testing.T) {
	tempDir := t.TempDir()
	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Stdout = &bytes.Buffer{}

	if err := ctx.Variables.Set("ONE", float64(1)); err != nil {
		t.Fatalf("set ONE: %v", err)
	}
	if err := ctx.Variables.Set("ALFABET", "abcdefghijkl"); err != nil {
		t.Fatalf("set ALFABET: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "SAVE", ToClause: "memfile"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	memPath := filepath.Join(tempDir, "memfile.mem")
	data, err := os.ReadFile(memPath)
	if err != nil {
		t.Fatalf("reading mem file: %v", err)
	}
	if len(data) == 0 || data[len(data)-1] != 0x1A {
		t.Fatalf("expected EOF marker, got % x", data)
	}

	f, err := os.Open(memPath)
	if err != nil {
		t.Fatalf("open mem file: %v", err)
	}
	defer f.Close()

	vars, err := mem.Read(f)
	if err != nil {
		t.Fatalf("mem.Read: %v", err)
	}
	if len(vars) != 2 {
		t.Fatalf("expected 2 variables, got %d", len(vars))
	}
}

func TestDispatchSaveToWithExtension(t *testing.T) {
	tempDir := t.TempDir()
	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir

	if err := ctx.Variables.Set("X", float64(99)); err != nil {
		t.Fatalf("set X: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "SAVE", ToClause: "backup.mem"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "backup.mem")); err != nil {
		t.Fatalf("expected backup.mem to exist: %v", err)
	}
}

func TestDispatchSaveToNoTo(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "SAVE"})
	if err == nil {
		t.Fatal("expected error for SAVE without TO")
	}
}

func TestDispatchSaveToEmptyVariables(t *testing.T) {
	tempDir := t.TempDir()
	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir

	err := commandMux.Dispatch(ctx, Command{Verb: "SAVE", ToClause: "empty"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tempDir, "empty.mem"))
	if err != nil {
		t.Fatalf("reading empty mem file: %v", err)
	}
	if len(data) != 1 || data[0] != 0x1A {
		t.Fatalf("expected single EOF byte, got % x", data)
	}
}
