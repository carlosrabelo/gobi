package repl

import (
	"bytes"
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

func TestDispatchDeleteFileRemovesFile(t *testing.T) {
	tempDir := t.TempDir()
	path := createTempDBFWithRecords(t, tempDir, "temp.dbf", nil)

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = true
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "DELETE", Args: "FILE temp"}); err != nil {
		t.Fatalf("DELETE FILE: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected file to be deleted")
	}
	if !strings.Contains(ctx.Stdout.(*bytes.Buffer).String(), "FILE HAS BEEN DELETED") {
		t.Fatal("expected deletion confirmation")
	}
}

func TestDispatchDeleteFileRejectsOpenFile(t *testing.T) {
	tempDir := t.TempDir()
	path := createTempDBFWithRecords(t, tempDir, "open.dbf", nil)

	ctx := testCtx()
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: path}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "DELETE", Args: "FILE " + path})
	if err == nil || !strings.Contains(err.Error(), "in use") {
		t.Fatalf("expected in use error, got %v", err)
	}
}

func TestDispatchDeleteFileRequiresFilename(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "DELETE", Args: "FILE"})
	if err == nil || !strings.Contains(err.Error(), "filename") {
		t.Fatalf("expected filename error, got %v", err)
	}
}

func TestDispatchEraseRejectsArguments(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "ERASE", Args: "temp"})
	if err == nil || !strings.Contains(err.Error(), "DELETE FILE") {
		t.Fatalf("expected DELETE FILE hint, got %v", err)
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

func TestDispatchEraseWithoutArgsClearsScreen(t *testing.T) {
	var stdout bytes.Buffer
	ctx := testCtx()
	ctx.Stdout = &stdout

	ctx.Screen.WriteAt(2, 4, "KEEP?")
	if err := commandMux.Dispatch(ctx, Command{Verb: "ERASE"}); err != nil {
		t.Fatalf("ERASE: %v", err)
	}

	if !strings.Contains(stdout.String(), "\033[2J") {
		t.Fatal("expected ERASE to emit a terminal clear sequence")
	}
	for row := 0; row < ctx.Screen.Rows(); row++ {
		for col := 0; col < ctx.Screen.Cols(); col++ {
			if ch := ctx.Screen.At(row, col); ch != ' ' {
				t.Fatalf("expected blank screen at (%d,%d), got %q", row, col, ch)
			}
		}
	}
}
