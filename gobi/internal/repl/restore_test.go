package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDispatchRestoreFrom(t *testing.T) {
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
	if err := ctx.Variables.Set("CHARS", "abcdefghijkl new stuff"); err != nil {
		t.Fatalf("set CHARS: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SAVE", ToClause: "memfile"}); err != nil {
		t.Fatalf("SAVE: %v", err)
	}

	ctx.Variables.Clear()
	if err := ctx.Variables.Set("THREE", float64(3)); err != nil {
		t.Fatalf("set THREE: %v", err)
	}
	if ctx.Variables.Len() != 1 {
		t.Fatalf("expected 1 variable before restore, got %d", ctx.Variables.Len())
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "RESTORE", FromClause: "memfile"}); err != nil {
		t.Fatalf("RESTORE: %v", err)
	}

	if ctx.Variables.Len() != 3 {
		t.Fatalf("expected 3 variables after restore, got %d", ctx.Variables.Len())
	}

	one, ok := ctx.Variables.Get("ONE")
	if !ok || one.(float64) != 1 {
		t.Fatalf("expected ONE=1, got %#v", one)
	}
	alfabet, ok := ctx.Variables.Get("ALFABET")
	if !ok || alfabet.(string) != "abcdefghijkl" {
		t.Fatalf("expected ALFABET restored, got %#v", alfabet)
	}
	chars, ok := ctx.Variables.Get("CHARS")
	if !ok || chars.(string) != "abcdefghijkl new stuff" {
		t.Fatalf("expected CHARS restored, got %#v", chars)
	}
	if _, ok := ctx.Variables.Get("THREE"); ok {
		t.Fatal("THREE should have been replaced by restore")
	}
}

func TestDispatchRestoreFromWithExtension(t *testing.T) {
	tempDir := t.TempDir()
	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir

	if err := ctx.Variables.Set("X", float64(99)); err != nil {
		t.Fatalf("set X: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SAVE", ToClause: "backup.mem"}); err != nil {
		t.Fatalf("SAVE: %v", err)
	}

	ctx.Variables.Clear()
	if err := commandMux.Dispatch(ctx, Command{Verb: "RESTORE", FromClause: "backup.mem"}); err != nil {
		t.Fatalf("RESTORE: %v", err)
	}

	val, ok := ctx.Variables.Get("X")
	if !ok || val.(float64) != 99 {
		t.Fatalf("expected X=99, got %#v", val)
	}
}

func TestDispatchRestoreFromEmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir

	if err := commandMux.Dispatch(ctx, Command{Verb: "SAVE", ToClause: "empty"}); err != nil {
		t.Fatalf("SAVE: %v", err)
	}
	if err := ctx.Variables.Set("OLD", "value"); err != nil {
		t.Fatalf("set OLD: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "RESTORE", FromClause: "empty"}); err != nil {
		t.Fatalf("RESTORE: %v", err)
	}
	if ctx.Variables.Len() != 0 {
		t.Fatalf("expected empty registry after restore from empty file, got %d vars", ctx.Variables.Len())
	}
}

func TestDispatchRestoreFromNoFrom(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "RESTORE"})
	if err == nil {
		t.Fatal("expected error for RESTORE without FROM")
	}
}

func TestDispatchRestoreFromMissingFile(t *testing.T) {
	tempDir := t.TempDir()
	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir

	err := commandMux.Dispatch(ctx, Command{Verb: "RESTORE", FromClause: "missing"})
	if err == nil {
		t.Fatal("expected error for missing memory file")
	}
	if !os.IsNotExist(err) && !contains(err.Error(), "Could not open file") {
		t.Fatalf("expected open error, got %v", err)
	}
}

func contains(s, sub string) bool {
	return bytes.Contains([]byte(s), []byte(sub))
}

func TestDispatchSaveRestoreRoundTripLogical(t *testing.T) {
	tempDir := t.TempDir()
	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir

	if err := ctx.Variables.Set("FLAG", true); err != nil {
		t.Fatalf("set FLAG: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SAVE", ToClause: "flags"}); err != nil {
		t.Fatalf("SAVE: %v", err)
	}

	ctx.Variables.Clear()
	if err := commandMux.Dispatch(ctx, Command{Verb: "RESTORE", FromClause: filepath.Join(tempDir, "flags.mem")}); err != nil {
		t.Fatalf("RESTORE with absolute path: %v", err)
	}

	val, ok := ctx.Variables.Get("FLAG")
	if !ok || val.(bool) != true {
		t.Fatalf("expected FLAG=true, got %#v", val)
	}
}
