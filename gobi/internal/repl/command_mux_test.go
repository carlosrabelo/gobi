package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
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
func TestDispatchSelectPrimary(t *testing.T) {
	var stdout bytes.Buffer
	ctx := testCtx()
	ctx.Stdout = &stdout

	ctx.ActiveArea = context.Secondary
	err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "PRIMARY"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.ActiveArea != context.Primary {
		t.Fatal("expected PRIMARY area to be active")
	}
}

func TestDispatchSelectSecondary(t *testing.T) {
	var stdout bytes.Buffer
	ctx := testCtx()
	ctx.Stdout = &stdout

	err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "SECONDARY"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.ActiveArea != context.Secondary {
		t.Fatal("expected SECONDARY area to be active")
	}
}

func TestDispatchSelectInvalid(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "TERTIARY"})
	if err == nil {
		t.Fatal("expected error for invalid work area")
	}
}

func TestDispatchSelectNoArgs(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: ""})
	if err == nil {
		t.Fatal("expected error for SELECT without argument")
	}
	if !strings.Contains(err.Error(), "PRIMARY or SECONDARY") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatchSelectSwitchWorkAreas(t *testing.T) {
	tempDir := t.TempDir()

	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 35")...)...)

	primaryPath := createTempDBFWithRecords(t, tempDir, "primary.dbf", [][]byte{rec1})
	secondaryPath := createTempDBFWithRecords(t, tempDir, "secondary.dbf", [][]byte{rec2})

	ctx := testCtx()

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: primaryPath}); err != nil {
		t.Fatalf("unexpected error opening primary: %v", err)
	}
	if ctx.WorkAreas[context.Primary].Table == nil {
		t.Fatal("expected primary table to be open")
	}
	if ctx.WorkAreas[context.Primary].Alias != "PRIMARY" {
		t.Fatalf("expected alias PRIMARY, got %s", ctx.WorkAreas[context.Primary].Alias)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "SECONDARY"}); err != nil {
		t.Fatalf("unexpected error selecting secondary: %v", err)
	}
	if ctx.ActiveArea != context.Secondary {
		t.Fatal("expected secondary to be active")
	}
	if ctx.WorkAreas[context.Primary].Table == nil {
		t.Fatal("expected primary table to remain open after SELECT SECONDARY")
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: secondaryPath}); err != nil {
		t.Fatalf("unexpected error opening secondary: %v", err)
	}
	if ctx.WorkAreas[context.Secondary].Table == nil {
		t.Fatal("expected secondary table to be open")
	}
	if ctx.WorkAreas[context.Secondary].Alias != "SECONDARY" {
		t.Fatalf("expected alias SECONDARY, got %s", ctx.WorkAreas[context.Secondary].Alias)
	}
	if ctx.WorkAreas[context.Primary].Table == nil {
		t.Fatal("expected primary table to remain open after USE in secondary")
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "PRIMARY"}); err != nil {
		t.Fatalf("unexpected error selecting primary: %v", err)
	}
	if ctx.ActiveArea != context.Primary {
		t.Fatal("expected primary to be active after SELECT PRIMARY")
	}
}

func createTempDBFWithRecords(t *testing.T, dir string, name string, records [][]byte) string {
	path := filepath.Join(dir, name)

	// Fields: NAME (C, 10 chars), AGE (N, 3 chars, 0 dec)
	// Total Record Length = 1 (delete flag) + 10 + 3 = 14 bytes
	var fields = []dbf.FieldDescriptor{
		{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
		{Name: "AGE", Type: dbf.FieldTypeNumeric, Length: 3, DecimalCount: 0},
	}

	var buf []byte
	buf = append(buf, 0x02)                                      // Sig
	buf = append(buf, byte(len(records)), byte(len(records)>>8)) // Count
	buf = append(buf, 0x50, 0x06, 0x01)                          // Date
	buf = append(buf, 14, 0x00)                                  // RecordLen (14 bytes)

	// Write field descriptors
	for _, f := range fields {
		fb := make([]byte, 16)
		copy(fb, f.Name)
		fb[10] = byte(f.Type)
		fb[11] = f.Length
		fb[14] = f.DecimalCount
		buf = append(buf, fb...)
	}
	buf = append(buf, 0x0D) // Term

	// Write records
	for _, rec := range records {
		buf = append(buf, rec...)
	}
	buf = append(buf, 0x1A) // EOF

	err := os.WriteFile(path, buf, 0644)
	if err != nil {
		t.Fatalf("failed to write temp DBF: %v", err)
	}
	return path
}
