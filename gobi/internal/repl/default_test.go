package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplySetDefaultTo(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := applySetDefault(ctx, "DEFAULT TO /tmp/data"); err != nil {
		t.Fatalf("applySetDefault: %v", err)
	}
	if ctx.Config.DefaultDir != filepath.Clean("/tmp/data") {
		t.Fatalf("DefaultDir = %q, want %q", ctx.Config.DefaultDir, filepath.Clean("/tmp/data"))
	}
	if !strings.Contains(ctx.Stdout.(*bytes.Buffer).String(), "Default directory:") {
		t.Fatal("expected SET confirmation")
	}
}

func TestApplySetDefaultWithoutTo(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := applySetDefault(ctx, "DEFAULT demos"); err != nil {
		t.Fatalf("applySetDefault: %v", err)
	}
	if ctx.Config.DefaultDir != "demos" {
		t.Fatalf("DefaultDir = %q, want demos", ctx.Config.DefaultDir)
	}
}

func TestApplySetDefaultRequiresPath(t *testing.T) {
	ctx := testCtx()
	err := applySetDefault(ctx, "DEFAULT")
	if err == nil || !strings.Contains(err.Error(), "drive/directory") {
		t.Fatalf("expected path error, got %v", err)
	}
}

func TestResolveDataPathRelative(t *testing.T) {
	ctx := testCtx()
	ctx.Config.DefaultDir = "/data"

	got := resolveDataPath(ctx, "people", ".dbf")
	want := filepath.Join("/data", "people.dbf")
	if got != want {
		t.Fatalf("resolveDataPath = %q, want %q", got, want)
	}
}

func TestResolveDataPathAbsoluteIgnoresDefault(t *testing.T) {
	ctx := testCtx()
	ctx.Config.DefaultDir = "/data"

	got := resolveDataPath(ctx, "/tmp/people.dbf", ".dbf")
	if got != "/tmp/people.dbf" {
		t.Fatalf("resolveDataPath = %q, want absolute path unchanged", got)
	}
}

func TestResolveDBFFilePathKeepsExistingExtension(t *testing.T) {
	ctx := testCtx()
	ctx.Config.DefaultDir = "/data"

	got := resolveDBFFilePath(ctx, "archive/old.dbf")
	want := filepath.Join("/data", "archive", "old.dbf")
	if got != want {
		t.Fatalf("resolveDBFFilePath = %q, want %q", got, want)
	}
}

func TestDispatchUseOpensDatabaseFromSetDefault(t *testing.T) {
	tempDir := t.TempDir()
	createTempDBFWithRecords(t, tempDir, "people.dbf", nil)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "DEFAULT TO " + tempDir}); err != nil {
		t.Fatalf("SET DEFAULT: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		t.Fatal("expected table to be opened")
	}
	if area.Alias != "PEOPLE" {
		t.Fatalf("alias = %q, want PEOPLE", area.Alias)
	}
	if int(area.Table.Header.RecordCount) != 0 {
		t.Fatalf("record count = %d, want 0", area.Table.Header.RecordCount)
	}
}

func TestDispatchDoUsesSetDefault(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "hello.prg")
	if err := os.WriteFile(scriptPath, []byte("STORE 42 TO answer\n"), 0644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "DEFAULT TO " + tempDir}); err != nil {
		t.Fatalf("SET DEFAULT: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "DO", Args: "hello"}); err != nil {
		t.Fatalf("DO: %v", err)
	}

	val, ok := ctx.Variables.Get("answer")
	if !ok {
		t.Fatal("expected answer variable to be set")
	}
	switch v := val.(type) {
	case int:
		if v != 42 {
			t.Fatalf("answer = %d, want 42", v)
		}
	case float64:
		if v != 42 {
			t.Fatalf("answer = %v, want 42", v)
		}
	default:
		t.Fatalf("answer = %v (%T), want numeric 42", val, val)
	}
}
