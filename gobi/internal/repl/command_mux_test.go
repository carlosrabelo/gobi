package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDispatchUnknownVerb(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "XYZZY"})
	if err == nil {
		t.Fatal("expected error for unknown verb")
	}
	if !strings.Contains(err.Error(), "Unrecognized command") {
		t.Fatalf("expected unrecognized command error, got %v", err)
	}
}

func TestDispatchAbbreviation(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "QUI"})
	if err != errQuit {
		t.Fatalf("expected errQuit for QUI abbreviation, got %v", err)
	}
}

func TestDispatchExactVerb(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "QUIT"})
	if err != errQuit {
		t.Fatalf("expected errQuit, got %v", err)
	}
}

func createTempDBF(t *testing.T, dir string, name string) string {
	path := filepath.Join(dir, name)
	var buf []byte
	buf = append(buf, 0x02)
	buf = append(buf, 0x00, 0x00)
	buf = append(buf, 0x50, 0x06, 0x01)
	buf = append(buf, 0x15, 0x00)
	fb := make([]byte, 16)
	copy(fb, "NAME")
	fb[10] = 'C'
	fb[11] = 20
	buf = append(buf, fb...)
	buf = append(buf, 0x0D)
	if err := os.WriteFile(path, buf, 0644); err != nil {
		t.Fatalf("failed to write temp DBF: %v", err)
	}
	return path
}

func TestDispatchUse(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBF(t, tempDir, "testdb.dbf")

	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	activeArea := ctx.GetActiveArea()
	if activeArea.Table == nil {
		t.Fatal("expected table to be opened")
	}
	if activeArea.Alias != "TESTDB" {
		t.Fatalf("expected alias TESTDB, got %s", activeArea.Alias)
	}

	err = commandMux.Dispatch(ctx, Command{Verb: "USE", Args: ""})
	if err != nil {
		t.Fatalf("unexpected error closing database: %v", err)
	}
	if activeArea.Table != nil {
		t.Fatal("expected table to be closed (nil)")
	}

	err = commandMux.Dispatch(ctx, Command{Verb: "USE", Args: filepath.Join(tempDir, "nonexistent")})
	if err == nil {
		t.Fatal("expected error when opening non-existent file")
	}
	if !strings.Contains(err.Error(), "Could not open file") {
		t.Fatalf("expected file opening error, got: %v", err)
	}
}
