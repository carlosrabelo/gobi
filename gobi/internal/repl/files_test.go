package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRenamedFilePathKeepsExtension(t *testing.T) {
	ctx := testCtx()
	ctx.Config.DefaultDir = "/data"

	oldPath := filepath.Join("/data", "jobtemp.dbf")
	got := resolveRenamedFilePath(ctx, oldPath, "jobdet")
	want := filepath.Join("/data", "jobdet.dbf")
	if got != want {
		t.Fatalf("resolveRenamedFilePath = %q, want %q", got, want)
	}
}

func TestDispatchRenameFile(t *testing.T) {
	tempDir := t.TempDir()
	oldPath := createTempDBFWithRecords(t, tempDir, "jobtemp.dbf", nil)
	newPath := filepath.Join(tempDir, "jobdet.dbf")

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir

	if err := commandMux.Dispatch(ctx, Command{Verb: "RENAME", Args: "jobtemp", ToClause: "jobdet"}); err != nil {
		t.Fatalf("RENAME: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("expected old file to be gone")
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected renamed file: %v", err)
	}
}

func TestDispatchRenameRejectsOpenFile(t *testing.T) {
	tempDir := t.TempDir()
	createTempDBFWithRecords(t, tempDir, "people.dbf", nil)

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "RENAME", Args: "people", ToClause: "staff"})
	if err == nil || !strings.Contains(err.Error(), "in use") {
		t.Fatalf("expected in use error, got %v", err)
	}
}

func TestDispatchRenameRequiresToClause(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "RENAME", Args: "people"})
	if err == nil || !strings.Contains(err.Error(), "TO") {
		t.Fatalf("expected TO clause error, got %v", err)
	}
}

func TestDispatchEraseRemovesFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "temp.dbf")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false

	if err := commandMux.Dispatch(ctx, Command{Verb: "ERASE", Args: "temp"}); err != nil {
		t.Fatalf("ERASE: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, stat err=%v", err)
	}
}
