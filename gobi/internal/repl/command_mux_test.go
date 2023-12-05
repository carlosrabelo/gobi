package repl

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/ndx"
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

	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME"}); err != nil {
		t.Fatalf("unexpected error listing secondary: %v", err)
	}
	if !strings.Contains(stdout.String(), "Bob") || strings.Contains(stdout.String(), "Alice") {
		t.Fatalf("expected secondary LIST to show Bob only, got: %q", stdout.String())
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "PRIMARY"}); err != nil {
		t.Fatalf("unexpected error selecting primary: %v", err)
	}
	if ctx.ActiveArea != context.Primary {
		t.Fatal("expected primary to be active after SELECT PRIMARY")
	}

	stdout.Reset()
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME"}); err != nil {
		t.Fatalf("unexpected error listing primary: %v", err)
	}
	if !strings.Contains(stdout.String(), "Alice") || strings.Contains(stdout.String(), "Bob") {
		t.Fatalf("expected primary LIST to show Alice only, got: %q", stdout.String())
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

func TestDispatchCloseDatabases(t *testing.T) {
	var stdout bytes.Buffer
	ctx := testCtx()
	ctx.Stdout = &stdout

	ctx.WorkAreas[context.Primary].Table = &dbf.Table{}
	ctx.WorkAreas[context.Primary].Alias = "TESTDB"
	ctx.WorkAreas[context.Secondary].Table = &dbf.Table{}
	ctx.WorkAreas[context.Secondary].Alias = "OTHER"

	err := commandMux.Dispatch(ctx, Command{Verb: "CLOSE", Args: "DATABASES"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.WorkAreas[context.Primary].Table != nil {
		t.Fatal("expected primary table to be nil after CLOSE DATABASES")
	}
	if ctx.WorkAreas[context.Secondary].Table != nil {
		t.Fatal("expected secondary table to be nil after CLOSE DATABASES")
	}
	if ctx.WorkAreas[context.Primary].Alias != "PRIMARY" {
		t.Fatalf("expected primary alias reset, got %s", ctx.WorkAreas[context.Primary].Alias)
	}
	if ctx.WorkAreas[context.Secondary].Alias != "SECONDARY" {
		t.Fatalf("expected secondary alias reset, got %s", ctx.WorkAreas[context.Secondary].Alias)
	}
}

func TestDispatchCloseBare(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	ctx.WorkAreas[context.Primary].Table = &dbf.Table{}
	ctx.WorkAreas[context.Secondary].Table = &dbf.Table{}

	err := commandMux.Dispatch(ctx, Command{Verb: "CLOSE", Args: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.WorkAreas[context.Primary].Table != nil || ctx.WorkAreas[context.Secondary].Table != nil {
		t.Fatal("expected bare CLOSE to close all databases")
	}
}

func TestDispatchCloseDatabasesWithUse(t *testing.T) {
	tempDir := t.TempDir()

	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 35")...)...)

	primaryPath := createTempDBFWithRecords(t, tempDir, "primary.dbf", [][]byte{rec1})
	secondaryPath := createTempDBFWithRecords(t, tempDir, "secondary.dbf", [][]byte{rec2})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: primaryPath}); err != nil {
		t.Fatalf("unexpected error opening primary: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "SECONDARY"}); err != nil {
		t.Fatalf("unexpected error selecting secondary: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: secondaryPath}); err != nil {
		t.Fatalf("unexpected error opening secondary: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "CLOSE", Args: "DATABASES"}); err != nil {
		t.Fatalf("unexpected error closing databases: %v", err)
	}
	if ctx.WorkAreas[context.Primary].Table != nil || ctx.WorkAreas[context.Secondary].Table != nil {
		t.Fatal("expected both work areas to be closed")
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected LIST error after CLOSE DATABASES, got %v", err)
	}
}

func TestDispatchCloseIndex(t *testing.T) {
	var stdout bytes.Buffer
	ctx := testCtx()
	ctx.Stdout = &stdout

	area := ctx.GetActiveArea()
	area.Table = &dbf.Table{}
	area.Indexes = []*ndx.Index{{Path: "stub"}}
	err := commandMux.Dispatch(ctx, Command{Verb: "CLOSE", Args: "INDEX"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(area.Indexes) != 0 {
		t.Fatal("expected indexes to be empty after CLOSE INDEX")
	}
	if area.Table == nil {
		t.Fatal("expected database to remain open after CLOSE INDEX")
	}
}

func TestDispatchCloseIndexWithUse(t *testing.T) {
	tempDir := t.TempDir()

	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "indexdb.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening database: %v", err)
	}

	area := ctx.GetActiveArea()
	area.Indexes = []*ndx.Index{{Path: "stub-index"}}

	if err := commandMux.Dispatch(ctx, Command{Verb: "CLOSE", Args: "INDEX"}); err != nil {
		t.Fatalf("unexpected error closing indexes: %v", err)
	}
	if len(area.Indexes) != 0 {
		t.Fatal("expected indexes to be cleared")
	}
	if area.Table == nil {
		t.Fatal("expected database to remain open after CLOSE INDEX")
	}

	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME"}); err != nil {
		t.Fatalf("unexpected error listing after CLOSE INDEX: %v", err)
	}
	if !strings.Contains(stdout.String(), "Alice") {
		t.Fatalf("expected LIST to work after CLOSE INDEX, got: %q", stdout.String())
	}
}

func TestDispatchCloseIndexOtherAreaUnchanged(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	ctx.WorkAreas[context.Primary].Indexes = []*ndx.Index{{Path: "primary-index"}}
	ctx.WorkAreas[context.Secondary].Indexes = []*ndx.Index{{Path: "secondary-index"}}
	ctx.ActiveArea = context.Secondary

	if err := commandMux.Dispatch(ctx, Command{Verb: "CLOSE", Args: "INDEX"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ctx.WorkAreas[context.Secondary].Indexes) != 0 {
		t.Fatal("expected secondary indexes to be cleared")
	}
	if len(ctx.WorkAreas[context.Primary].Indexes) != 1 {
		t.Fatal("expected primary indexes to remain open")
	}
}

func TestDispatchCloseAll(t *testing.T) {
	var stdout bytes.Buffer
	ctx := testCtx()
	ctx.Stdout = &stdout

	for _, area := range ctx.WorkAreas {
		area.Table = &dbf.Table{}
		area.Indexes = []*ndx.Index{{Path: "stub"}}
	}
	err := commandMux.Dispatch(ctx, Command{Verb: "CLOSE", Args: "ALL"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, area := range ctx.WorkAreas {
		if area.Table != nil {
			t.Fatal("expected table to be nil after CLOSE ALL")
		}
		if len(area.Indexes) != 0 {
			t.Fatal("expected indexes to be empty after CLOSE ALL")
		}
	}
}

func TestDispatchCloseInvalidOption(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "CLOSE", Args: "WINDOWS"})
	if err == nil {
		t.Fatal("expected error for invalid CLOSE option")
	}
}

func TestDispatchDisplayStructure(t *testing.T) {
	tempDir := t.TempDir()

	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "structdb.dbf", [][]byte{rec})

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "DISPLAY", Args: "STRUCTURE"}); err != nil {
		t.Fatalf("unexpected error on DISPLAY STRUCTURE: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "STRUCTURE FOR FILE:  STRUCTDB.DBF") {
		t.Fatalf("unexpected header in output: %q", output)
	}
	if !strings.Contains(output, "NAME") || !strings.Contains(output, "AGE") {
		t.Fatalf("expected field names in output: %q", output)
	}
	if !strings.Contains(output, "** TOTAL **") {
		t.Fatalf("expected total width line in output: %q", output)
	}
}

func TestDispatchDisplayStructureNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "DISPLAY", Args: "STRUCTURE"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchListStructureNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "STRUCTURE"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchGotoNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "1"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchGotoNoArgs(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: ""})
	if err == nil || !strings.Contains(err.Error(), "requires a record number") {
		t.Fatalf("expected missing record number error, got %v", err)
	}
}

func TestDispatchGotoInvalidNumber(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "gotodb.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "abc"})
	if err == nil || !strings.Contains(err.Error(), "Invalid record number") {
		t.Fatalf("expected invalid record number error, got %v", err)
	}
}

func TestDispatchGoto(t *testing.T) {
	tempDir := t.TempDir()

	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 35")...)...)
	rec3 := append([]byte{0x20}, append([]byte("Charlie   "), []byte(" 45")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "gotodb.dbf", [][]byte{rec1, rec2, rec3})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.RecordNo != 0 {
		t.Fatalf("expected initial record 0, got %d", area.RecordNo)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "2"}); err != nil {
		t.Fatalf("unexpected error on GOTO 2: %v", err)
	}
	if area.RecordNo != 1 {
		t.Fatalf("expected record index 1, got %d", area.RecordNo)
	}
	if area.ActiveRecord == nil {
		t.Fatal("expected active record to be loaded")
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TO 3"}); err != nil {
		t.Fatalf("unexpected error on GO TO 3: %v", err)
	}
	if area.RecordNo != 2 {
		t.Fatalf("expected record index 2, got %d", area.RecordNo)
	}
	name, err := area.ActiveRecord.DecodeField(area.Table, 0)
	if err != nil {
		t.Fatalf("decode NAME: %v", err)
	}
	if s, ok := name.(string); !ok || !strings.Contains(s, "Charlie") {
		t.Fatalf("expected NAME=Charlie, got %#v", name)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "4"}); err != nil {
		t.Fatalf("unexpected error on GOTO past EOF: %v", err)
	}
	if area.RecordNo != 3 {
		t.Fatalf("expected record index 3 at EOF, got %d", area.RecordNo)
	}
	if area.ActiveRecord != nil {
		t.Fatal("expected no active record past EOF")
	}

	err = commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "0"})
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out of range error for GOTO 0, got %v", err)
	}
}

func TestDispatchGoTopNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TOP"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchGoBottomNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "BOTTOM"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchGoTopAndBottom(t *testing.T) {
	tempDir := t.TempDir()

	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 35")...)...)
	rec3 := append([]byte{0x20}, append([]byte("Charlie   "), []byte(" 45")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "gotodb.dbf", [][]byte{rec1, rec2, rec3})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "2"}); err != nil {
		t.Fatalf("unexpected error on GOTO 2: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TOP"}); err != nil {
		t.Fatalf("unexpected error on GO TOP: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.RecordNo != 0 {
		t.Fatalf("expected record index 0 after GO TOP, got %d", area.RecordNo)
	}
	name, err := area.ActiveRecord.DecodeField(area.Table, 0)
	if err != nil {
		t.Fatalf("decode NAME: %v", err)
	}
	if s, ok := name.(string); !ok || !strings.Contains(s, "Alice") {
		t.Fatalf("expected NAME=Alice after GO TOP, got %#v", name)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "BOTTOM"}); err != nil {
		t.Fatalf("unexpected error on GO BOTTOM: %v", err)
	}
	if area.RecordNo != 2 {
		t.Fatalf("expected record index 2 after GO BOTTOM, got %d", area.RecordNo)
	}
	name, err = area.ActiveRecord.DecodeField(area.Table, 0)
	if err != nil {
		t.Fatalf("decode NAME: %v", err)
	}
	if s, ok := name.(string); !ok || !strings.Contains(s, "Charlie") {
		t.Fatalf("expected NAME=Charlie after GO BOTTOM, got %#v", name)
	}
}

func TestDispatchGoTopEmptyDatabase(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBF(t, tempDir, "empty.dbf")

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TOP"}); err != nil {
		t.Fatalf("unexpected error on GO TOP empty table: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.ActiveRecord != nil {
		t.Fatal("expected no active record on empty table")
	}
}

func TestDispatchSkipNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "SKIP"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchSkipInvalid(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "skipdb.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "SKIP", Args: "abc"})
	if err == nil || !strings.Contains(err.Error(), "Invalid SKIP value") {
		t.Fatalf("expected invalid SKIP value error, got %v", err)
	}
}

func TestDispatchSkip(t *testing.T) {
	tempDir := t.TempDir()

	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 35")...)...)
	rec3 := append([]byte{0x20}, append([]byte("Charlie   "), []byte(" 45")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "skipdb.dbf", [][]byte{rec1, rec2, rec3})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	area := ctx.GetActiveArea()

	if err := commandMux.Dispatch(ctx, Command{Verb: "SKIP"}); err != nil {
		t.Fatalf("unexpected error on SKIP: %v", err)
	}
	if area.RecordNo != 1 {
		t.Fatalf("expected record index 1 after SKIP, got %d", area.RecordNo)
	}
	name, err := area.ActiveRecord.DecodeField(area.Table, 0)
	if err != nil {
		t.Fatalf("decode NAME: %v", err)
	}
	if s, ok := name.(string); !ok || !strings.Contains(s, "Bob") {
		t.Fatalf("expected NAME=Bob after SKIP, got %#v", name)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SKIP", Args: "1"}); err != nil {
		t.Fatalf("unexpected error on SKIP 1: %v", err)
	}
	if area.RecordNo != 2 {
		t.Fatalf("expected record index 2 after SKIP 1, got %d", area.RecordNo)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SKIP", Args: "-2"}); err != nil {
		t.Fatalf("unexpected error on SKIP -2: %v", err)
	}
	if area.RecordNo != 0 {
		t.Fatalf("expected record index 0 after SKIP -2, got %d", area.RecordNo)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SKIP", Args: "3"}); err != nil {
		t.Fatalf("unexpected error on SKIP past EOF: %v", err)
	}
	if area.RecordNo != 3 {
		t.Fatalf("expected record index 3 at EOF, got %d", area.RecordNo)
	}
	if area.ActiveRecord != nil {
		t.Fatal("expected no active record past EOF")
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SKIP", Args: "-1"}); err != nil {
		t.Fatalf("unexpected error on SKIP -1 from EOF: %v", err)
	}
	if area.RecordNo != 2 {
		t.Fatalf("expected record index 2 after SKIP -1 from EOF, got %d", area.RecordNo)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TOP"}); err != nil {
		t.Fatalf("unexpected error on GO TOP: %v", err)
	}
	err = commandMux.Dispatch(ctx, Command{Verb: "SKIP", Args: "-1"})
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out of range error for SKIP -1 at first record, got %v", err)
	}
}

func TestDispatchList(t *testing.T) {
	tempDir := t.TempDir()

	rec1 := append([]byte{0x2A}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x2A}, append([]byte("Bob       "), []byte(" 35")...)...)
	rec3 := append([]byte{0x20}, append([]byte("Charlie   "), []byte(" 45")...)...)

	dbfPath := createTempDBFWithRecords(t, tempDir, "listdb.dbf", [][]byte{rec1, rec2, rec3})

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	err := commandMux.Dispatch(ctx, Command{Verb: "LIST"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected 'No database file is in use' error, got %v", err)
	}

	err = commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath})
	if err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	stdout.Reset()
	err = commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "STRUCTURE"})
	if err != nil {
		t.Fatalf("unexpected error on LIST STRUCTURE: %v", err)
	}
	structOut := stdout.String()
	if !strings.Contains(structOut, "STRUCTURE FOR FILE:  LISTDB.DBF") || !strings.Contains(structOut, "NAME") || !strings.Contains(structOut, "AGE") {
		t.Fatalf("unexpected list structure output: %q", structOut)
	}

	stdout.Reset()
	err = commandMux.Dispatch(ctx, Command{Verb: "LIST"})
	if err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	listOut := stdout.String()
	if !strings.Contains(listOut, "Alice") || !strings.Contains(listOut, "Bob") || !strings.Contains(listOut, "Charlie") {
		t.Fatalf("unexpected plain list output: %q", listOut)
	}

	stdout.Reset()
	err = commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	listNameOut := stdout.String()
	if !strings.Contains(listNameOut, "Alice") || strings.Contains(listNameOut, "25") {
		t.Fatalf("expected only NAME column in output, got: %q", listNameOut)
	}

	stdout.Reset()
	err = commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME, AGE", ForClause: "DELETED()"})
	if err != nil {
		t.Fatalf("unexpected error on LIST FOR: %v", err)
	}
	listForOut := stdout.String()
	if !strings.Contains(listForOut, "Alice") || !strings.Contains(listForOut, "Bob") || strings.Contains(listForOut, "Charlie") {
		t.Fatalf("expected only Alice and Bob, got: %q", listForOut)
	}

	area := ctx.GetActiveArea()
	area.RecordNo = 1
	rseeker, _ := area.Table.Underlying().(io.ReadSeeker)
	area.ActiveRecord, _ = area.Table.ReadRecordAt(rseeker, 1)

	stdout.Reset()
	err = commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME, AGE", WhileClause: "DELETED()"})
	if err != nil {
		t.Fatalf("unexpected error on LIST WHILE: %v", err)
	}
	listWhileOut := stdout.String()
	if !strings.Contains(listWhileOut, "Bob") || strings.Contains(listWhileOut, "Charlie") || strings.Contains(listWhileOut, "Alice") {
		t.Fatalf("expected only Bob, got: %q", listWhileOut)
	}

	outFilePath := filepath.Join(tempDir, "output.txt")
	stdout.Reset()
	err = commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME", ToClause: outFilePath})
	if err != nil {
		t.Fatalf("unexpected error on LIST TO: %v", err)
	}

	fileBytes, err := os.ReadFile(outFilePath)
	if err != nil {
		t.Fatalf("failed to read redirected output file: %v", err)
	}
	fileOut := string(fileBytes)
	if !strings.Contains(fileOut, "Alice") || !strings.Contains(fileOut, "Bob") || !strings.Contains(fileOut, "Charlie") {
		t.Fatalf("redirected file content incorrect: %q", fileOut)
	}
}

func createTempDBFWithNRecords(t *testing.T, dir string, name string, count int) string {
	records := make([][]byte, count)
	for i := 0; i < count; i++ {
		nameField := fmt.Sprintf("U%-9d", i+1)
		records[i] = append([]byte{0x20}, append([]byte(nameField), []byte("  1")...)...)
	}
	return createTempDBFWithRecords(t, dir, name, records)
}

func countRecordDataLines(output string) int {
	count := 0
	for _, line := range strings.Split(output, "\r\n") {
		if len(line) >= 6 && line[0] == ' ' && line[5] == ' ' {
			count++
		}
	}
	return count
}

func TestDispatchDisplayNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "DISPLAY"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchDisplayRecords(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithNRecords(t, tempDir, "dispdb.dbf", 25)

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TOP"}); err != nil {
		t.Fatalf("unexpected error on GO TOP: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "DISPLAY", Args: "NAME"}); err != nil {
		t.Fatalf("unexpected error on DISPLAY: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Record#") || !strings.Contains(output, "NAME") {
		t.Fatalf("expected header in output: %q", output)
	}
	pageSize := displayPageSize(ctx)
	if count := countRecordDataLines(output); count != pageSize {
		t.Fatalf("expected %d data lines, got %d", pageSize, count)
	}

	area := ctx.GetActiveArea()
	if area.RecordNo != pageSize-1 {
		t.Fatalf("expected cursor on last displayed record, got index %d", area.RecordNo)
	}

	stdout.Reset()
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	if count := countRecordDataLines(stdout.String()); count != 25 {
		t.Fatalf("expected LIST to show all 25 records, got %d", count)
	}
}

func TestDispatchDisplayPagination(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithNRecords(t, tempDir, "pagedb.dbf", 25)

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TOP"}); err != nil {
		t.Fatalf("unexpected error on GO TOP: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "DISPLAY"}); err != nil {
		t.Fatalf("unexpected error on first DISPLAY: %v", err)
	}
	if count := countRecordDataLines(stdout.String()); count != displayPageSize(ctx) {
		t.Fatalf("expected first page of %d records, got %d", displayPageSize(ctx), count)
	}

	area := ctx.GetActiveArea()
	if err := commandMux.Dispatch(ctx, Command{Verb: "SKIP", Args: "1"}); err != nil {
		t.Fatalf("unexpected error on SKIP: %v", err)
	}

	stdout.Reset()
	if err := commandMux.Dispatch(ctx, Command{Verb: "DISPLAY"}); err != nil {
		t.Fatalf("unexpected error on second DISPLAY: %v", err)
	}
	if count := countRecordDataLines(stdout.String()); count != 5 {
		t.Fatalf("expected second page of 5 records, got %d", count)
	}
	if area.RecordNo != 24 {
		t.Fatalf("expected cursor on last record, got index %d", area.RecordNo)
	}
}

func TestDispatchDisplayFromCurrentRecord(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithNRecords(t, tempDir, "curdb.dbf", 10)

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "8"}); err != nil {
		t.Fatalf("unexpected error on GOTO: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "DISPLAY", Args: "NAME"}); err != nil {
		t.Fatalf("unexpected error on DISPLAY: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "    8") {
		t.Fatalf("expected DISPLAY to start at record 8, got: %q", output)
	}
	if count := countRecordDataLines(output); count != 3 {
		t.Fatalf("expected 3 records from position 8, got %d", count)
	}
}

func TestDispatchDisplayForWhile(t *testing.T) {
	tempDir := t.TempDir()

	rec1 := append([]byte{0x2A}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x2A}, append([]byte("Bob       "), []byte(" 35")...)...)
	rec3 := append([]byte{0x20}, append([]byte("Charlie   "), []byte(" 45")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "dispfilter.dbf", [][]byte{rec1, rec2, rec3})

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:      "DISPLAY",
		Args:      "NAME",
		ForClause: "DELETED()",
	}); err != nil {
		t.Fatalf("unexpected error on DISPLAY FOR: %v", err)
	}
	output := stdout.String()
	if count := countRecordDataLines(output); count != 2 {
		t.Fatalf("expected 2 deleted records, got %d", count)
	}
	if strings.Contains(output, "Charlie") {
		t.Fatalf("expected deleted records only, got: %q", output)
	}
}

func TestDispatchDisplayRejectsTO(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "dispdb.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "DISPLAY", ToClause: "out.txt"})
	if err == nil || !strings.Contains(err.Error(), "does not support TO") {
		t.Fatalf("expected TO clause error, got %v", err)
	}
}

func TestDispatchAppendNoDatabase(t *testing.T) {
	ctx := testCtx()
	ctx.Stdin = strings.NewReader("\n")

	err := commandMux.Dispatch(ctx, Command{Verb: "APPEND"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchAppendCancelImmediately(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithRecords(t, tempDir, "appenddb.dbf", nil)

	ctx := testCtx()
	ctx.Stdin = strings.NewReader("\n")
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "APPEND"}); err != nil {
		t.Fatalf("unexpected error on APPEND: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 0 {
		t.Fatalf("expected no records appended, got %d", area.Table.Header.RecordCount)
	}
}

func TestDispatchAppendRecord(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithRecords(t, tempDir, "appenddb.dbf", nil)

	ctx := testCtx()
	ctx.Stdin = strings.NewReader("Alice\n25\n\n")
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "APPEND"}); err != nil {
		t.Fatalf("unexpected error on APPEND: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 1 {
		t.Fatalf("expected 1 record, got %d", area.Table.Header.RecordCount)
	}
	if area.RecordNo != 0 {
		t.Fatalf("expected cursor on new record 0, got %d", area.RecordNo)
	}
	if area.ActiveRecord == nil {
		t.Fatal("expected active record to be loaded")
	}

	stdout.Reset()
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME, AGE"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Alice") || !strings.Contains(output, "25") {
		t.Fatalf("expected appended record in LIST output, got: %q", output)
	}
}

func TestDispatchAppendMultipleRecords(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithRecords(t, tempDir, "appenddb.dbf", nil)

	ctx := testCtx()
	ctx.Stdin = strings.NewReader("Alice\n25\nBob\n35\n\n")
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "APPEND"}); err != nil {
		t.Fatalf("unexpected error on APPEND: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 2 {
		t.Fatalf("expected 2 records, got %d", area.Table.Header.RecordCount)
	}
	if area.RecordNo != 1 {
		t.Fatalf("expected cursor on second record, got index %d", area.RecordNo)
	}
}

func TestDispatchAppendInvalidNumeric(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithRecords(t, tempDir, "appenddb.dbf", nil)

	ctx := testCtx()
	ctx.Stdin = strings.NewReader("Alice\nxyz\n")
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "APPEND"})
	if err == nil || !strings.Contains(err.Error(), "Invalid numeric value") {
		t.Fatalf("expected invalid numeric error, got %v", err)
	}
}

func TestDispatchAppendRequiresFrom(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithRecords(t, tempDir, "appenddb.dbf", nil)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "APPEND", Args: "something"})
	if err == nil || !strings.Contains(err.Error(), "requires FROM") {
		t.Fatalf("expected FROM requirement error, got %v", err)
	}
}

func createListTestDBF(t *testing.T, tempDir string) string {
	rec1 := append([]byte{0x2A}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x2A}, append([]byte("Bob       "), []byte(" 35")...)...)
	rec3 := append([]byte{0x20}, append([]byte("Charlie   "), []byte(" 45")...)...)
	return createTempDBFWithRecords(t, tempDir, "listdb.dbf", [][]byte{rec1, rec2, rec3})
}

func TestDispatchReplaceNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "REPLACE", Args: `NAME WITH "X"`})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchReplaceInvalidSyntax(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "REPLACE", Args: "NAME"})
	if err == nil || !strings.Contains(err.Error(), "WITH") {
		t.Fatalf("expected REPLACE syntax error, got %v", err)
	}
}

func TestDispatchReplaceUnknownField(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "REPLACE", Args: `NOPE WITH "X"`})
	if err == nil || !strings.Contains(err.Error(), "Unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestDispatchReplaceCurrentRecord(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "REPLACE", Args: `NAME WITH "Updated"`}); err != nil {
		t.Fatalf("unexpected error on REPLACE: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.ActiveRecord == nil {
		t.Fatal("expected active record after REPLACE")
	}

	stdout.Reset()
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Updated") || strings.Contains(output, "Alice") {
		t.Fatalf("expected NAME=Updated on current record, got: %q", output)
	}
}

func TestDispatchReplaceFor(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:      "REPLACE",
		Args:      "AGE WITH 50",
		ForClause: "DELETED()",
	}); err != nil {
		t.Fatalf("unexpected error on REPLACE FOR: %v", err)
	}

	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME, AGE"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Alice") || !strings.Contains(output, "50") {
		t.Fatalf("expected deleted Alice with AGE 50, got: %q", output)
	}
	if !strings.Contains(output, "Bob") {
		t.Fatalf("expected Bob in output, got: %q", output)
	}
	if strings.Contains(output, "Charlie") && strings.Contains(strings.Split(output, "Charlie")[1], "50") {
		t.Fatalf("expected Charlie to keep original age, got: %q", output)
	}
}

func TestDispatchReplaceExpression(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "3"}); err != nil {
		t.Fatalf("unexpected error on GOTO: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "REPLACE", Args: `NAME WITH UPPER("charlie")`}); err != nil {
		t.Fatalf("unexpected error on REPLACE: %v", err)
	}

	area := ctx.GetActiveArea()
	name, err := area.ActiveRecord.DecodeField(area.Table, 0)
	if err != nil {
		t.Fatalf("decode NAME: %v", err)
	}
	if s, ok := name.(string); !ok || !strings.Contains(s, "CHARLIE") {
		t.Fatalf("expected NAME=CHARLIE, got %#v", name)
	}
}

func TestDispatchDeleteNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "DELETE"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchDeleteCurrentRecord(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "3"}); err != nil {
		t.Fatalf("unexpected error on GOTO: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "DELETE"}); err != nil {
		t.Fatalf("unexpected error on DELETE: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.ActiveRecord == nil || !area.ActiveRecord.Deleted {
		t.Fatal("expected current record to be marked deleted")
	}
}

func TestDispatchDeleteFor(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:      "DELETE",
		ForClause: ".T.",
	}); err != nil {
		t.Fatalf("unexpected error on DELETE FOR: %v", err)
	}

	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME", ForClause: "DELETED()"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Alice") || !strings.Contains(output, "Bob") || !strings.Contains(output, "Charlie") {
		t.Fatalf("expected all records deleted, got: %q", output)
	}
}

func TestDispatchDeleteAlreadyDeleted(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "DELETE"}); err != nil {
		t.Fatalf("unexpected error on DELETE already-deleted record: %v", err)
	}

	area := ctx.GetActiveArea()
	if !area.ActiveRecord.Deleted {
		t.Fatal("expected active record to remain deleted")
	}
}

func TestDispatchRecallNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "RECALL"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchRecallCurrentRecord(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "RECALL"}); err != nil {
		t.Fatalf("unexpected error on RECALL: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.ActiveRecord == nil || area.ActiveRecord.Deleted {
		t.Fatal("expected current record to be recalled")
	}
}

func TestDispatchRecallFor(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:      "RECALL",
		ForClause: "DELETED()",
	}); err != nil {
		t.Fatalf("unexpected error on RECALL FOR: %v", err)
	}

	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME", ForClause: "DELETED()"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	if count := countRecordDataLines(stdout.String()); count != 0 {
		t.Fatalf("expected no deleted records after RECALL FOR, got %d", count)
	}
}

func TestDispatchRecallNotDeleted(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "3"}); err != nil {
		t.Fatalf("unexpected error on GOTO: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "RECALL"}); err != nil {
		t.Fatalf("unexpected error on RECALL active record: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.ActiveRecord.Deleted {
		t.Fatal("expected active record to remain not deleted")
	}
}

func TestDispatchPackNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "PACK"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchPackDropsDeleted(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "PACK"}); err != nil {
		t.Fatalf("unexpected error on PACK: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 1 {
		t.Fatalf("expected 1 record after PACK, got %d", area.Table.Header.RecordCount)
	}

	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Charlie") {
		t.Fatalf("expected Charlie after PACK, got: %q", output)
	}
	if strings.Contains(output, "Alice") || strings.Contains(output, "Bob") {
		t.Fatalf("expected deleted records removed, got: %q", output)
	}

	info, err := os.Stat(dbfPath)
	if err != nil {
		t.Fatalf("stat packed file: %v", err)
	}
	headerSize := 8 + 2*16 + 1
	wantSize := int64(headerSize + 14 + 1)
	if info.Size() != wantSize {
		t.Fatalf("expected packed file size %d, got %d", wantSize, info.Size())
	}
}

func TestDispatchPackNoDeleted(t *testing.T) {
	tempDir := t.TempDir()
	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 35")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "packdb.dbf", [][]byte{rec1, rec2})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "PACK"}); err != nil {
		t.Fatalf("unexpected error on PACK: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 2 {
		t.Fatalf("expected 2 records after PACK, got %d", area.Table.Header.RecordCount)
	}
}

func TestDispatchPackAllDeleted(t *testing.T) {
	tempDir := t.TempDir()
	rec1 := append([]byte{0x2A}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x2A}, append([]byte("Bob       "), []byte(" 35")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "packdb.dbf", [][]byte{rec1, rec2})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "PACK"}); err != nil {
		t.Fatalf("unexpected error on PACK: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 0 {
		t.Fatalf("expected 0 records after PACK, got %d", area.Table.Header.RecordCount)
	}
	if area.ActiveRecord != nil {
		t.Fatal("expected no active record after packing empty table")
	}
}

func TestDispatchZapNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "ZAP"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchZapTruncates(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "ZAP"}); err != nil {
		t.Fatalf("unexpected error on ZAP: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 0 {
		t.Fatalf("expected 0 records after ZAP, got %d", area.Table.Header.RecordCount)
	}
	if area.ActiveRecord != nil {
		t.Fatal("expected no active record after ZAP")
	}

	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	if count := countRecordDataLines(stdout.String()); count != 0 {
		t.Fatalf("expected no records listed after ZAP, got %d", count)
	}

	info, err := os.Stat(dbfPath)
	if err != nil {
		t.Fatalf("stat zapped file: %v", err)
	}
	headerSize := 8 + 2*16 + 1
	wantSize := int64(headerSize + 1)
	if info.Size() != wantSize {
		t.Fatalf("expected zapped file size %d, got %d", wantSize, info.Size())
	}
}

func TestDispatchZapEmptyTable(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithRecords(t, tempDir, "zapdb.dbf", nil)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error opening table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "ZAP"}); err != nil {
		t.Fatalf("unexpected error on ZAP empty table: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 0 {
		t.Fatalf("expected 0 records after ZAP, got %d", area.Table.Header.RecordCount)
	}
}
func TestDispatchCreateInteractive(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := filepath.Join(tempDir, "newdb.dbf")

	ctx := testCtx()
	ctx.Stdin = strings.NewReader("NAME,C,10\nAGE,N,3,0\n\n")
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "CREATE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error on CREATE: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table == nil {
		t.Fatal("expected table to be opened after CREATE")
	}
	if area.Alias != "NEWDB" {
		t.Fatalf("expected alias NEWDB, got %s", area.Alias)
	}
	if area.Table.Header.RecordCount != 0 {
		t.Fatalf("expected 0 records, got %d", area.Table.Header.RecordCount)
	}
	if len(area.Table.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(area.Table.Fields))
	}

	if _, err := os.Stat(dbfPath); err != nil {
		t.Fatalf("expected created file on disk: %v", err)
	}

	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "STRUCTURE"}); err != nil {
		t.Fatalf("unexpected error on LIST STRUCTURE: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "NAME") || !strings.Contains(output, "AGE") {
		t.Fatalf("expected created structure in output, got: %q", output)
	}
}

func TestDispatchCreatePromptedFilename(t *testing.T) {
	tempDir := t.TempDir()

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Stdin = strings.NewReader("promptdb\nNAME,C,5\n\n")
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "CREATE"}); err != nil {
		t.Fatalf("unexpected error on CREATE: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table == nil {
		t.Fatal("expected table to be opened after CREATE")
	}
	if area.Alias != "PROMPTDB" {
		t.Fatalf("expected alias PROMPTDB, got %s", area.Alias)
	}

	dbfPath := filepath.Join(tempDir, "promptdb.dbf")
	if _, err := os.Stat(dbfPath); err != nil {
		t.Fatalf("expected created file on disk: %v", err)
	}
}

func TestDispatchCreateDestroyExistingNo(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)

	infoBefore, err := os.Stat(dbfPath)
	if err != nil {
		t.Fatalf("stat existing file: %v", err)
	}

	ctx := testCtx()
	ctx.Stdin = strings.NewReader("N\n")
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "CREATE", Args: dbfPath}); err != nil {
		t.Fatalf("unexpected error on CREATE: %v", err)
	}

	infoAfter, err := os.Stat(dbfPath)
	if err != nil {
		t.Fatalf("stat existing file after cancel: %v", err)
	}
	if infoAfter.Size() != infoBefore.Size() {
		t.Fatalf("expected existing file unchanged, size before=%d after=%d", infoBefore.Size(), infoAfter.Size())
	}
}

func TestDispatchCreateBadFieldName(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := filepath.Join(tempDir, "baddb.dbf")

	ctx := testCtx()
	ctx.Stdin = strings.NewReader("1BAD,C,5\n")
	ctx.Stdout = &bytes.Buffer{}

	err := commandMux.Dispatch(ctx, Command{Verb: "CREATE", Args: dbfPath})
	if err == nil || !strings.Contains(err.Error(), "BAD NAME FIELD") {
		t.Fatalf("expected BAD NAME FIELD error, got %v", err)
	}
	if _, statErr := os.Stat(dbfPath); statErr == nil {
		t.Fatal("expected no file created after bad field name")
	}
}

func TestDispatchCreateNoFields(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := filepath.Join(tempDir, "emptydb.dbf")

	ctx := testCtx()
	ctx.Stdin = strings.NewReader("\n")
	ctx.Stdout = &bytes.Buffer{}

	err := commandMux.Dispatch(ctx, Command{Verb: "CREATE", Args: dbfPath})
	if err == nil || !strings.Contains(err.Error(), "At least one field is required") {
		t.Fatalf("expected at least one field error, got %v", err)
	}
}

func TestDispatchEditNoDatabase(t *testing.T) {
	ctx := testCtx()
	ctx.Stdin = strings.NewReader("\n")

	err := commandMux.Dispatch(ctx, Command{Verb: "EDIT", Args: "1"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchEditRequiresRecordNumber(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithRecords(t, tempDir, "editdb.dbf", nil)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "EDIT"})
	if err == nil || !strings.Contains(err.Error(), "requires a record number") {
		t.Fatalf("expected record number error, got %v", err)
	}
}

func TestDispatchEditOutOfRange(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "editdb.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "EDIT", Args: "5"})
	if err == nil || !strings.Contains(err.Error(), "Record number out of range") {
		t.Fatalf("expected out of range error, got %v", err)
	}
}

func TestDispatchEditLineModeSave(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "editdb.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Stdin = strings.NewReader("Bob\n\n")
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "EDIT", Args: "1"}); err != nil {
		t.Fatalf("unexpected error on EDIT: %v", err)
	}

	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	if !strings.Contains(stdout.String(), "Bob") {
		t.Fatalf("expected edited name Bob, got: %q", stdout.String())
	}
}

func TestDispatchModifyStructureNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "MODIFY", Args: "STRUCTURE"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchModifyStructureNotImplemented(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithRecords(t, tempDir, "moddb.dbf", nil)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "MODIFY", Args: "COMMAND"})
	if err == nil || !strings.Contains(err.Error(), "not yet implemented") {
		t.Fatalf("expected not implemented error, got %v", err)
	}
}

func TestDispatchModifyStructureCancelDataLoss(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "moddb.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Stdin = strings.NewReader("N\n")
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "MODIFY", Args: "STRUCTURE"}); err != nil {
		t.Fatalf("unexpected error on MODIFY STRUCTURE: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 1 {
		t.Fatalf("expected record count unchanged, got %d", area.Table.Header.RecordCount)
	}
	if len(area.Table.Fields) != 2 {
		t.Fatalf("expected 2 fields unchanged, got %d", len(area.Table.Fields))
	}
}

func TestDispatchModifyStructureAddField(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithRecords(t, tempDir, "moddb.dbf", nil)

	ctx := testCtx()
	ctx.Stdin = strings.NewReader("NAME,C,10\nAGE,N,3,0\nPHONE,C,10\n\n")
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "MODIFY", Args: "STRUCTURE"}); err != nil {
		t.Fatalf("unexpected error on MODIFY STRUCTURE: %v", err)
	}

	area := ctx.GetActiveArea()
	if len(area.Table.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(area.Table.Fields))
	}
	if area.Table.Fields[2].Name != "PHONE" {
		t.Fatalf("expected third field PHONE, got %s", area.Table.Fields[2].Name)
	}
	if area.Table.Header.RecordCount != 0 {
		t.Fatalf("expected 0 records after structure change, got %d", area.Table.Header.RecordCount)
	}

	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "STRUCTURE"}); err != nil {
		t.Fatalf("unexpected error on LIST STRUCTURE: %v", err)
	}
	if !strings.Contains(stdout.String(), "PHONE") {
		t.Fatalf("expected PHONE in structure, got: %q", stdout.String())
	}
}
func TestDispatchCopyNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "COPY", ToClause: "out.dbf"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchCopyNoToClause(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithRecords(t, tempDir, "copydb.dbf", nil)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "COPY"})
	if err == nil || !strings.Contains(err.Error(), "COPY requires TO") {
		t.Fatalf("expected COPY requires TO error, got %v", err)
	}
}

func TestDispatchCopyAllActiveRecords(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)
	outPath := filepath.Join(tempDir, "copyout.dbf")

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "COPY", ToClause: outPath}); err != nil {
		t.Fatalf("unexpected error on COPY TO: %v", err)
	}

	outCtx := testCtx()
	outCtx.Stdout = &bytes.Buffer{}
	if err := commandMux.Dispatch(outCtx, Command{Verb: "USE", Args: outPath}); err != nil {
		t.Fatalf("open copy: %v", err)
	}

	area := outCtx.GetActiveArea()
	if area.Table.Header.RecordCount != 1 {
		t.Fatalf("expected 1 copied record, got %d", area.Table.Header.RecordCount)
	}
	if len(area.Table.Fields) != 2 {
		t.Fatalf("expected 2 fields in copy, got %d", len(area.Table.Fields))
	}

	var stdout bytes.Buffer
	outCtx.Stdout = &stdout
	if err := commandMux.Dispatch(outCtx, Command{Verb: "LIST", Args: "NAME"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	if !strings.Contains(stdout.String(), "Charlie") {
		t.Fatalf("expected Charlie in copy, got: %q", stdout.String())
	}
}

func TestDispatchCopyFieldSubset(t *testing.T) {
	tempDir := t.TempDir()
	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 35")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "copydb.dbf", [][]byte{rec1, rec2})
	outPath := filepath.Join(tempDir, "nameonly.dbf")

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:     "COPY",
		ToClause: outPath,
		Args:     "FIELD NAME",
	}); err != nil {
		t.Fatalf("unexpected error on COPY TO FIELD: %v", err)
	}

	outCtx := testCtx()
	outCtx.Stdout = &bytes.Buffer{}
	if err := commandMux.Dispatch(outCtx, Command{Verb: "USE", Args: outPath}); err != nil {
		t.Fatalf("open copy: %v", err)
	}

	area := outCtx.GetActiveArea()
	if len(area.Table.Fields) != 1 {
		t.Fatalf("expected 1 field in copy, got %d", len(area.Table.Fields))
	}
	if area.Table.Fields[0].Name != "NAME" {
		t.Fatalf("expected NAME field, got %s", area.Table.Fields[0].Name)
	}
	if area.Table.Header.RecordCount != 2 {
		t.Fatalf("expected 2 copied records, got %d", area.Table.Header.RecordCount)
	}
}

func TestDispatchCopyForClause(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)
	outPath := filepath.Join(tempDir, "filtered.dbf")

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:      "COPY",
		ToClause:  outPath,
		ForClause: ".F.",
	}); err != nil {
		t.Fatalf("unexpected error on COPY TO FOR: %v", err)
	}

	outCtx := testCtx()
	outCtx.Stdout = &bytes.Buffer{}
	if err := commandMux.Dispatch(outCtx, Command{Verb: "USE", Args: outPath}); err != nil {
		t.Fatalf("open copy: %v", err)
	}

	area := outCtx.GetActiveArea()
	if area.Table.Header.RecordCount != 0 {
		t.Fatalf("expected 0 records copied for FOR .F., got %d", area.Table.Header.RecordCount)
	}
	if len(area.Table.Fields) != 2 {
		t.Fatalf("expected structure copied with 2 fields, got %d", len(area.Table.Fields))
	}
}

func TestDispatchCopyWhileFromRecord(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createListTestDBF(t, tempDir)
	outPath := filepath.Join(tempDir, "whileout.dbf")

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "3"}); err != nil {
		t.Fatalf("goto record 3: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:        "COPY",
		ToClause:    outPath,
		WhileClause: ".T.",
	}); err != nil {
		t.Fatalf("unexpected error on COPY TO WHILE: %v", err)
	}

	outCtx := testCtx()
	outCtx.Stdout = &bytes.Buffer{}
	if err := commandMux.Dispatch(outCtx, Command{Verb: "USE", Args: outPath}); err != nil {
		t.Fatalf("open copy: %v", err)
	}

	area := outCtx.GetActiveArea()
	if area.Table.Header.RecordCount != 1 {
		t.Fatalf("expected 1 record copied from record 3, got %d", area.Table.Header.RecordCount)
	}

	var stdout bytes.Buffer
	outCtx.Stdout = &stdout
	if err := commandMux.Dispatch(outCtx, Command{Verb: "LIST", Args: "NAME"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	if !strings.Contains(stdout.String(), "Charlie") {
		t.Fatalf("expected Charlie in WHILE copy, got: %q", stdout.String())
	}
}
func TestDispatchAppendFromNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "APPEND", FromClause: "src.dbf"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchAppendFromDBF(t *testing.T) {
	tempDir := t.TempDir()
	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 35")...)...)
	srcPath := createTempDBFWithRecords(t, tempDir, "source.dbf", [][]byte{rec1, rec2})
	dstPath := createTempDBFWithRecords(t, tempDir, "dest.dbf", nil)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dstPath}); err != nil {
		t.Fatalf("open dest: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "APPEND", FromClause: srcPath}); err != nil {
		t.Fatalf("unexpected error on APPEND FROM: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 2 {
		t.Fatalf("expected 2 appended records, got %d", area.Table.Header.RecordCount)
	}

	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME, AGE"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Alice") || !strings.Contains(output, "Bob") {
		t.Fatalf("expected imported records in LIST output, got: %q", output)
	}
}

func TestDispatchAppendFromDBFSkipsDeleted(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := createListTestDBF(t, tempDir)
	dstPath := createTempDBFWithRecords(t, tempDir, "dest.dbf", nil)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dstPath}); err != nil {
		t.Fatalf("open dest: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "APPEND", FromClause: srcPath}); err != nil {
		t.Fatalf("unexpected error on APPEND FROM: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 1 {
		t.Fatalf("expected 1 active record appended, got %d", area.Table.Header.RecordCount)
	}

	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NAME"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	if !strings.Contains(stdout.String(), "Charlie") {
		t.Fatalf("expected Charlie in appended records, got: %q", stdout.String())
	}
}

func TestDispatchAppendFromSDF(t *testing.T) {
	tempDir := t.TempDir()
	dstPath := createTempDBFWithRecords(t, tempDir, "dest.dbf", nil)
	txtPath := filepath.Join(tempDir, "import.txt")
	content := "Alice     25\r\nBob       35\r\n"
	if err := os.WriteFile(txtPath, []byte(content), 0644); err != nil {
		t.Fatalf("write txt: %v", err)
	}

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dstPath}); err != nil {
		t.Fatalf("open dest: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:       "APPEND",
		FromClause: txtPath,
		Args:       "SDF",
	}); err != nil {
		t.Fatalf("unexpected error on APPEND FROM SDF: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 2 {
		t.Fatalf("expected 2 SDF records appended, got %d", area.Table.Header.RecordCount)
	}
}

func TestDispatchAppendFromDelimited(t *testing.T) {
	tempDir := t.TempDir()
	dstPath := createTempDBFWithRecords(t, tempDir, "dest.dbf", nil)
	txtPath := filepath.Join(tempDir, "import.txt")
	content := "'Alice', 25\r\n'Bob', 35\r\n"
	if err := os.WriteFile(txtPath, []byte(content), 0644); err != nil {
		t.Fatalf("write txt: %v", err)
	}

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dstPath}); err != nil {
		t.Fatalf("open dest: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:       "APPEND",
		FromClause: txtPath,
		Args:       "DELIMITED",
	}); err != nil {
		t.Fatalf("unexpected error on APPEND FROM DELIMITED: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 2 {
		t.Fatalf("expected 2 delimited records appended, got %d", area.Table.Header.RecordCount)
	}
}

func TestDispatchAppendFromForFalse(t *testing.T) {
	tempDir := t.TempDir()
	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	srcPath := createTempDBFWithRecords(t, tempDir, "source.dbf", [][]byte{rec1})
	dstPath := createTempDBFWithRecords(t, tempDir, "dest.dbf", nil)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dstPath}); err != nil {
		t.Fatalf("open dest: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:       "APPEND",
		FromClause: srcPath,
		ForClause:  ".F.",
	}); err != nil {
		t.Fatalf("unexpected error on APPEND FROM FOR: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 0 {
		t.Fatalf("expected 0 records for FOR .F., got %d", area.Table.Header.RecordCount)
	}
}
func TestDispatchUpdateNoPrimaryDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "UPDATE", Args: "ON PARTNO ADD ONHAND"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchUpdateNoSecondaryDatabase(t *testing.T) {
	tempDir := t.TempDir()
	primaryPath := createInventoryDBF(t, tempDir, "primary.dbf", nil)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: primaryPath}); err != nil {
		t.Fatalf("open primary: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "UPDATE", Args: "ON PARTNO ADD ONHAND REPLACE COST"})
	if err == nil || !strings.Contains(err.Error(), "No secondary database file is in use") {
		t.Fatalf("expected no secondary database error, got %v", err)
	}
}

func TestDispatchUpdateFromSecondary(t *testing.T) {
	tempDir := t.TempDir()
	primaryPath := createInventoryDBF(t, tempDir, "primary.dbf", []inventoryRecord{
		{partNo: "11528", onHand: "16", cost: "22.00"},
		{partNo: "21828", onHand: "16", cost: "34.72"},
		{partNo: "70296", onHand: "5", cost: "200.00"},
		{partNo: "89793", onHand: "5", cost: "134999.00"},
	})
	secondaryPath := createInventoryDBF(t, tempDir, "secondary.dbf", []inventoryRecord{
		{partNo: "21828", onHand: "77", cost: "35.88"},
		{partNo: "70296", onHand: "0", cost: "250.00"},
		{partNo: "89793", onHand: "2", cost: "134999.00"},
	})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: primaryPath}); err != nil {
		t.Fatalf("open primary: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "SECONDARY"}); err != nil {
		t.Fatalf("select secondary: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: secondaryPath}); err != nil {
		t.Fatalf("open secondary: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "PRIMARY"}); err != nil {
		t.Fatalf("select primary: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb: "UPDATE",
		Args: "ON PARTNO ADD ONHAND REPLACE COST",
	}); err != nil {
		t.Fatalf("unexpected error on UPDATE: %v", err)
	}

	area := ctx.GetActiveArea()
	wseeker := area.Table.Underlying().(io.ReadSeeker)
	_, onHandIdx := area.Table.FieldByName("ONHAND")
	_, costIdx := area.Table.FieldByName("COST")

	checkRecord := func(partNo string, wantOnHand float64, wantCost float64) {
		t.Helper()
		for i := 0; i < int(area.Table.Header.RecordCount); i++ {
			rec, err := area.Table.ReadRecordAt(wseeker, i)
			if err != nil {
				t.Fatalf("read record %d: %v", i, err)
			}
			partVal, _ := rec.DecodeField(area.Table, 0)
			if strings.TrimSpace(fmt.Sprintf("%v", partVal)) != partNo {
				continue
			}
			onHand, _ := rec.DecodeField(area.Table, onHandIdx)
			cost, _ := rec.DecodeField(area.Table, costIdx)
			if onHand.(float64) != wantOnHand {
				t.Fatalf("part %s onhand = %v, want %v", partNo, onHand, wantOnHand)
			}
			if cost.(float64) != wantCost {
				t.Fatalf("part %s cost = %v, want %v", partNo, cost, wantCost)
			}
			return
		}
		t.Fatalf("record %s not found", partNo)
	}

	checkRecord("11528", 16, 22.00)
	checkRecord("21828", 93, 35.88)
	checkRecord("70296", 5, 250.00)
	checkRecord("89793", 7, 134999.00)
}

func TestDispatchUpdateFromFile(t *testing.T) {
	tempDir := t.TempDir()
	primaryPath := createInventoryDBF(t, tempDir, "primary.dbf", []inventoryRecord{
		{partNo: "21828", onHand: "16", cost: "34.72"},
	})
	secondaryPath := createInventoryDBF(t, tempDir, "updates.dbf", []inventoryRecord{
		{partNo: "21828", onHand: "77", cost: "35.88"},
	})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: primaryPath}); err != nil {
		t.Fatalf("open primary: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:       "UPDATE",
		FromClause: secondaryPath,
		Args:       "ON PARTNO ADD ONHAND REPLACE COST",
	}); err != nil {
		t.Fatalf("unexpected error on UPDATE FROM file: %v", err)
	}

	area := ctx.GetActiveArea()
	wseeker := area.Table.Underlying().(io.ReadSeeker)
	rec, err := area.Table.ReadRecordAt(wseeker, 0)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	_, onHandIdx := area.Table.FieldByName("ONHAND")
	_, costIdx := area.Table.FieldByName("COST")
	onHand, _ := rec.DecodeField(area.Table, onHandIdx)
	cost, _ := rec.DecodeField(area.Table, costIdx)
	if onHand.(float64) != 93 || cost.(float64) != 35.88 {
		t.Fatalf("unexpected updated values: onhand=%v cost=%v", onHand, cost)
	}
}

func TestDispatchUpdateKeyLengthMismatch(t *testing.T) {
	tempDir := t.TempDir()
	primaryPath := createInventoryDBF(t, tempDir, "primary.dbf", []inventoryRecord{
		{partNo: "21828", onHand: "16", cost: "34.72"},
	})
	secondaryPath := createInventoryDBFWithPartWidth(t, tempDir, "secondary.dbf", 6, []inventoryRecord{
		{partNo: "21828", onHand: "77", cost: "35.88"},
	})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: primaryPath}); err != nil {
		t.Fatalf("open primary: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "SECONDARY"}); err != nil {
		t.Fatalf("select secondary: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: secondaryPath}); err != nil {
		t.Fatalf("open secondary: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "PRIMARY"}); err != nil {
		t.Fatalf("select primary: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "UPDATE", Args: "ON PARTNO ADD ONHAND REPLACE COST"})
	if err == nil || !strings.Contains(err.Error(), "KEYS ARE NOT THE SAME LENGTH") {
		t.Fatalf("expected key length error, got %v", err)
	}
}

type inventoryRecord struct {
	partNo string
	onHand string
	cost   string
}

func createInventoryDBF(t *testing.T, dir, name string, records []inventoryRecord) string {
	return createInventoryDBFWithPartWidth(t, dir, name, 5, records)
}

func createInventoryDBFWithPartWidth(t *testing.T, dir, name string, partWidth byte, records []inventoryRecord) string {
	t.Helper()
	path := filepath.Join(dir, name)

	fields := []dbf.FieldDescriptor{
		{Name: "PARTNO", Type: dbf.FieldTypeChar, Length: partWidth},
		{Name: "ONHAND", Type: dbf.FieldTypeNumeric, Length: 5, DecimalCount: 0},
		{Name: "COST", Type: dbf.FieldTypeNumeric, Length: 10, DecimalCount: 2},
	}

	recordLen := 1 + int(partWidth) + 5 + 10
	var buf []byte
	buf = append(buf, 0x02)
	buf = append(buf, byte(len(records)), byte(len(records)>>8))
	buf = append(buf, 0x50, 0x06, 0x01)
	buf = append(buf, byte(recordLen), byte(recordLen>>8))

	for _, f := range fields {
		fb := make([]byte, 16)
		copy(fb, f.Name)
		fb[10] = byte(f.Type)
		fb[11] = f.Length
		fb[14] = f.DecimalCount
		buf = append(buf, fb...)
	}
	buf = append(buf, 0x0D)

	for _, rec := range records {
		row := make([]byte, recordLen)
		row[0] = 0x20
		copy(row[1:], fmt.Sprintf("%-*s", partWidth, rec.partNo))
		copy(row[1+int(partWidth):], fmt.Sprintf("%5s", rec.onHand))
		copy(row[1+int(partWidth)+5:], fmt.Sprintf("%10s", rec.cost))
		buf = append(buf, row...)
	}
	buf = append(buf, 0x1A)

	if err := os.WriteFile(path, buf, 0644); err != nil {
		t.Fatalf("write inventory dbf: %v", err)
	}
	return path
}
func TestDispatchJoinNoToClause(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "JOIN", ForClause: ".T."})
	if err == nil || !strings.Contains(err.Error(), "JOIN requires TO") {
		t.Fatalf("expected JOIN requires TO error, got %v", err)
	}
}

func TestDispatchJoinNoForClause(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "JOIN", ToClause: "out.dbf"})
	if err == nil || !strings.Contains(err.Error(), "JOIN requires FOR") {
		t.Fatalf("expected JOIN requires FOR error, got %v", err)
	}
}

func TestDispatchJoinNoSecondaryDatabase(t *testing.T) {
	tempDir := t.TempDir()
	primaryPath := createJoinPrimaryDBF(t, tempDir, "primary.dbf", nil)
	outPath := filepath.Join(tempDir, "joinout.dbf")

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: primaryPath}); err != nil {
		t.Fatalf("open primary: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "JOIN", ToClause: outPath, ForClause: ".T."})
	if err == nil || !strings.Contains(err.Error(), "No secondary database file is in use") {
		t.Fatalf("expected no secondary database error, got %v", err)
	}
}

func TestDispatchJoinMatchingKeys(t *testing.T) {
	tempDir := t.TempDir()
	primaryPath := createJoinPrimaryDBF(t, tempDir, "primary.dbf", []joinPersonRecord{
		{name: "Harris", key: "d2"},
		{name: "Shaffer", key: "d8"},
	})
	secondaryPath := createJoinSecondaryDBF(t, tempDir, "secondary.dbf", []joinTitleRecord{
		{key: "d2", title: "Shift Leader"},
		{key: "d8", title: "Shift Leader"},
		{key: "p3", title: "Programmer"},
	})
	outPath := filepath.Join(tempDir, "joinout.dbf")

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: primaryPath}); err != nil {
		t.Fatalf("open primary: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "SECONDARY"}); err != nil {
		t.Fatalf("select secondary: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: secondaryPath}); err != nil {
		t.Fatalf("open secondary: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "PRIMARY"}); err != nil {
		t.Fatalf("select primary: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:      "JOIN",
		ToClause:  outPath,
		ForClause: "P.KEY = S.KEY",
		Args:      "FIELD NAME, KEY, S.TITLE",
	}); err != nil {
		t.Fatalf("unexpected error on JOIN: %v", err)
	}

	outCtx := testCtx()
	outCtx.Stdout = &bytes.Buffer{}
	if err := commandMux.Dispatch(outCtx, Command{Verb: "USE", Args: outPath}); err != nil {
		t.Fatalf("open join output: %v", err)
	}

	area := outCtx.GetActiveArea()
	if area.Table.Header.RecordCount != 2 {
		t.Fatalf("expected 2 joined records, got %d", area.Table.Header.RecordCount)
	}

	var stdout bytes.Buffer
	outCtx.Stdout = &stdout
	if err := commandMux.Dispatch(outCtx, Command{Verb: "LIST", Args: "NAME, TITLE"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Harris") || !strings.Contains(output, "Shaffer") {
		t.Fatalf("expected joined names in output, got: %q", output)
	}
	if !strings.Contains(output, "Shift Leader") {
		t.Fatalf("expected joined titles in output, got: %q", output)
	}
}

func TestDispatchJoinForTrue(t *testing.T) {
	tempDir := t.TempDir()
	primaryPath := createJoinPrimaryDBF(t, tempDir, "primary.dbf", []joinPersonRecord{
		{name: "Alice", key: "001"},
	})
	secondaryPath := createJoinSecondaryDBF(t, tempDir, "secondary.dbf", []joinTitleRecord{
		{key: "001", title: "One"},
		{key: "002", title: "Two"},
	})
	outPath := filepath.Join(tempDir, "joinout.dbf")

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: primaryPath}); err != nil {
		t.Fatalf("open primary: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "SECONDARY"}); err != nil {
		t.Fatalf("select secondary: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: secondaryPath}); err != nil {
		t.Fatalf("open secondary: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SELECT", Args: "PRIMARY"}); err != nil {
		t.Fatalf("select primary: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:      "JOIN",
		ToClause:  outPath,
		ForClause: ".T.",
	}); err != nil {
		t.Fatalf("unexpected error on JOIN FOR .T.: %v", err)
	}

	outCtx := testCtx()
	if err := commandMux.Dispatch(outCtx, Command{Verb: "USE", Args: outPath}); err != nil {
		t.Fatalf("open join output: %v", err)
	}
	if outCtx.GetActiveArea().Table.Header.RecordCount != 2 {
		t.Fatalf("expected 2 joined records for FOR .T., got %d", outCtx.GetActiveArea().Table.Header.RecordCount)
	}
}

type joinPersonRecord struct {
	name string
	key  string
}

type joinTitleRecord struct {
	key   string
	title string
}

func createJoinPrimaryDBF(t *testing.T, dir, name string, records []joinPersonRecord) string {
	t.Helper()
	path := filepath.Join(dir, name)
	fields := []dbf.FieldDescriptor{
		{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
		{Name: "KEY", Type: dbf.FieldTypeChar, Length: 3},
	}
	recordLen := 1 + 10 + 3
	return writeJoinDBF(t, path, fields, recordLen, len(records), func(i int, row []byte) {
		row[0] = 0x20
		copy(row[1:], fmt.Sprintf("%-10s", records[i].name))
		copy(row[11:], fmt.Sprintf("%-3s", records[i].key))
	})
}

func createJoinSecondaryDBF(t *testing.T, dir, name string, records []joinTitleRecord) string {
	t.Helper()
	path := filepath.Join(dir, name)
	fields := []dbf.FieldDescriptor{
		{Name: "KEY", Type: dbf.FieldTypeChar, Length: 3},
		{Name: "TITLE", Type: dbf.FieldTypeChar, Length: 12},
	}
	recordLen := 1 + 3 + 12
	return writeJoinDBF(t, path, fields, recordLen, len(records), func(i int, row []byte) {
		row[0] = 0x20
		copy(row[1:], fmt.Sprintf("%-3s", records[i].key))
		copy(row[4:], fmt.Sprintf("%-12s", records[i].title))
	})
}

func writeJoinDBF(t *testing.T, path string, fields []dbf.FieldDescriptor, recordLen, recCount int, fill func(int, []byte)) string {
	t.Helper()
	var buf []byte
	buf = append(buf, 0x02)
	buf = append(buf, byte(recCount), byte(recCount>>8))
	buf = append(buf, 0x50, 0x06, 0x01)
	buf = append(buf, byte(recordLen), byte(recordLen>>8))

	for _, f := range fields {
		fb := make([]byte, 16)
		copy(fb, f.Name)
		fb[10] = byte(f.Type)
		fb[11] = f.Length
		fb[14] = f.DecimalCount
		buf = append(buf, fb...)
	}
	buf = append(buf, 0x0D)

	for i := 0; i < recCount; i++ {
		row := make([]byte, recordLen)
		fill(i, row)
		buf = append(buf, row...)
	}
	buf = append(buf, 0x1A)

	if err := os.WriteFile(path, buf, 0644); err != nil {
		t.Fatalf("write join dbf: %v", err)
	}
	return path
}
func TestDispatchTotalNoToClause(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := createTotalSalesDBF(t, tempDir, "sales.dbf", nil)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: srcPath}); err != nil {
		t.Fatalf("open source: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "TOTAL", Args: "ON DEPTNUM FIELD SALARY"})
	if err == nil || !strings.Contains(err.Error(), "TO") {
		t.Fatalf("expected TO error, got %v", err)
	}
}

func TestDispatchTotalSummarizeField(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := createTotalSalesDBF(t, tempDir, "sales.dbf", []totalSalesRecord{
		{dept: "16", salary: "25000.00", name: "Alice"},
		{dept: "16", salary: "13625.00", name: "Bob"},
		{dept: "54", salary: "61700.00", name: "John"},
	})
	outPath := filepath.Join(tempDir, "deptsals.dbf")

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: srcPath}); err != nil {
		t.Fatalf("open source: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:     "TOTAL",
		ToClause: outPath,
		Args:     "ON DEPTNUM FIELD SALARY",
	}); err != nil {
		t.Fatalf("unexpected error on TOTAL: %v", err)
	}

	outCtx := testCtx()
	outCtx.Stdout = &bytes.Buffer{}
	if err := commandMux.Dispatch(outCtx, Command{Verb: "USE", Args: outPath}); err != nil {
		t.Fatalf("open total output: %v", err)
	}

	area := outCtx.GetActiveArea()
	if area.Table.Header.RecordCount != 2 {
		t.Fatalf("expected 2 total records, got %d", area.Table.Header.RecordCount)
	}
	if len(area.Table.Fields) != 2 {
		t.Fatalf("expected 2 fields in total output, got %d", len(area.Table.Fields))
	}

	var stdout bytes.Buffer
	outCtx.Stdout = &stdout
	if err := commandMux.Dispatch(outCtx, Command{Verb: "LIST", Args: "DEPTNUM, SALARY"}); err != nil {
		t.Fatalf("unexpected error on LIST: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "38625") {
		t.Fatalf("expected summed salary 38625 for dept 16, got: %q", output)
	}
	if !strings.Contains(output, "61700") {
		t.Fatalf("expected salary 61700 for dept 54, got: %q", output)
	}
}

func TestDispatchTotalForClause(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := createTotalSalesDBF(t, tempDir, "sales.dbf", []totalSalesRecord{
		{dept: "16", salary: "25000.00", name: "Alice"},
		{dept: "16", salary: "13625.00", name: "Bob"},
		{dept: "54", salary: "61700.00", name: "John"},
	})
	outPath := filepath.Join(tempDir, "deptsals.dbf")

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: srcPath}); err != nil {
		t.Fatalf("open source: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:      "TOTAL",
		ToClause:  outPath,
		ForClause: "DEPTNUM = '16'",
		Args:      "ON DEPTNUM FIELD SALARY",
	}); err != nil {
		t.Fatalf("unexpected error on TOTAL FOR: %v", err)
	}

	outCtx := testCtx()
	outCtx.Stdout = &bytes.Buffer{}
	if err := commandMux.Dispatch(outCtx, Command{Verb: "USE", Args: outPath}); err != nil {
		t.Fatalf("open total output: %v", err)
	}

	area := outCtx.GetActiveArea()
	if area.Table.Header.RecordCount != 1 {
		t.Fatalf("expected 1 total record for FOR clause, got %d", area.Table.Header.RecordCount)
	}
}

type totalSalesRecord struct {
	dept   string
	salary string
	name   string
}

func createTotalSalesDBF(t *testing.T, dir, name string, records []totalSalesRecord) string {
	t.Helper()
	path := filepath.Join(dir, name)

	fields := []dbf.FieldDescriptor{
		{Name: "DEPTNUM", Type: dbf.FieldTypeChar, Length: 3},
		{Name: "SALARY", Type: dbf.FieldTypeNumeric, Length: 8, DecimalCount: 2},
		{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
	}
	recordLen := 1 + 3 + 8 + 10

	var buf []byte
	buf = append(buf, 0x02)
	buf = append(buf, byte(len(records)), byte(len(records)>>8))
	buf = append(buf, 0x50, 0x06, 0x01)
	buf = append(buf, byte(recordLen), byte(recordLen>>8))

	for _, f := range fields {
		fb := make([]byte, 16)
		copy(fb, f.Name)
		fb[10] = byte(f.Type)
		fb[11] = f.Length
		fb[14] = f.DecimalCount
		buf = append(buf, fb...)
	}
	buf = append(buf, 0x0D)

	for _, rec := range records {
		row := make([]byte, recordLen)
		row[0] = 0x20
		copy(row[1:], fmt.Sprintf("%-3s", rec.dept))
		copy(row[4:], fmt.Sprintf("%8s", rec.salary))
		copy(row[12:], fmt.Sprintf("%-10s", rec.name))
		buf = append(buf, row...)
	}
	buf = append(buf, 0x1A)

	if err := os.WriteFile(path, buf, 0644); err != nil {
		t.Fatalf("write total sales dbf: %v", err)
	}
	return path
}
func TestDispatchLocateNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "LOCATE", ForClause: "AGE > 30"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchLocateNoForClause(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "LOCATE"})
	if err == nil || !strings.Contains(err.Error(), "FOR") {
		t.Fatalf("expected FOR error, got %v", err)
	}
}

func TestDispatchContinueNoLocate(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "CONTINUE"})
	if err == nil || !strings.Contains(err.Error(), "No active LOCATE") {
		t.Fatalf("expected no active locate error, got %v", err)
	}
}

func TestDispatchLocateAndContinue(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "LOCATE", ForClause: "AGE >= 35"}); err != nil {
		t.Fatalf("unexpected error on LOCATE: %v", err)
	}

	area := ctx.GetActiveArea()
	if !area.Found || area.RecordNo != 1 {
		t.Fatalf("expected first match at record 2, found=%v recordNo=%d", area.Found, area.RecordNo)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "CONTINUE"}); err != nil {
		t.Fatalf("unexpected error on first CONTINUE: %v", err)
	}
	if !area.Found || area.RecordNo != 2 {
		t.Fatalf("expected second match at record 3, found=%v recordNo=%d", area.Found, area.RecordNo)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "CONTINUE"}); err != nil {
		t.Fatalf("unexpected error on second CONTINUE: %v", err)
	}
	if !area.Found || area.RecordNo != 3 {
		t.Fatalf("expected third match at record 4, found=%v recordNo=%d", area.Found, area.RecordNo)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "CONTINUE"}); err != nil {
		t.Fatalf("unexpected error on third CONTINUE: %v", err)
	}
	if area.Found || area.RecordNo != 4 || area.ActiveRecord != nil {
		t.Fatalf("expected end of locate scope, found=%v recordNo=%d active=%v", area.Found, area.RecordNo, area.ActiveRecord)
	}
}

func TestDispatchLocateNotFound(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "LOCATE", ForClause: "AGE > 100"}); err != nil {
		t.Fatalf("unexpected error on LOCATE: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Found || area.ActiveRecord != nil {
		t.Fatalf("expected no match, found=%v", area.Found)
	}
	if !strings.Contains(stdout.String(), "End of Locate scope") {
		t.Fatalf("expected end-of-scope talk message, got %q", stdout.String())
	}
}

func createLocateTestDBF(t *testing.T, tempDir string) string {
	t.Helper()
	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 35")...)...)
	rec3 := append([]byte{0x20}, append([]byte("Charlie   "), []byte(" 45")...)...)
	rec4 := append([]byte{0x20}, append([]byte("Dave      "), []byte(" 35")...)...)
	return createTempDBFWithRecords(t, tempDir, "locatedb.dbf", [][]byte{rec1, rec2, rec3, rec4})
}
func TestDispatchCountNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "COUNT", ForClause: "AGE > 30"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchCountAllRecords(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "COUNT"}); err != nil {
		t.Fatalf("unexpected error on COUNT: %v", err)
	}
	if !strings.Contains(stdout.String(), "COUNT = 00004") {
		t.Fatalf("expected count of 4, got %q", stdout.String())
	}
}

func TestDispatchCountForClause(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "COUNT", ForClause: "AGE >= 35"}); err != nil {
		t.Fatalf("unexpected error on COUNT FOR: %v", err)
	}
	if !strings.Contains(stdout.String(), "COUNT = 00003") {
		t.Fatalf("expected count of 3, got %q", stdout.String())
	}
}

func TestDispatchCountToVariable(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:      "COUNT",
		ForClause: "AGE >= 35",
		ToClause:  "adults",
	}); err != nil {
		t.Fatalf("unexpected error on COUNT TO: %v", err)
	}

	val, ok := ctx.Variables.Get("ADULTS")
	if !ok {
		t.Fatal("expected ADULTS memory variable")
	}
	if num, ok := val.(float64); !ok || num != 3 {
		t.Fatalf("expected ADULTS=3, got %v", val)
	}
}

func TestDispatchCountPreservesRecordPointer(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	area := ctx.GetActiveArea()
	area.RecordNo = 2
	rseeker := area.Table.Underlying().(io.ReadSeeker)
	rec, err := area.Table.ReadRecordAt(rseeker, 2)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	area.ActiveRecord = rec

	if err := commandMux.Dispatch(ctx, Command{Verb: "COUNT", ForClause: "AGE >= 35"}); err != nil {
		t.Fatalf("unexpected error on COUNT: %v", err)
	}
	if area.RecordNo != 2 || area.ActiveRecord != rec {
		t.Fatalf("expected record pointer preserved at record 3")
	}
}
func TestDispatchSumNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "SUM", Args: "AGE"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchSumNoExpression(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "SUM"})
	if err == nil || !strings.Contains(err.Error(), "numeric expression") {
		t.Fatalf("expected expression error, got %v", err)
	}
}

func TestDispatchSumAllRecords(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SUM", Args: "AGE"}); err != nil {
		t.Fatalf("unexpected error on SUM: %v", err)
	}
	if !strings.Contains(stdout.String(), "140.00") {
		t.Fatalf("expected sum of 140.00, got %q", stdout.String())
	}
}

func TestDispatchSumForClause(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SUM", Args: "AGE", ForClause: "AGE >= 35"}); err != nil {
		t.Fatalf("unexpected error on SUM FOR: %v", err)
	}
	if !strings.Contains(stdout.String(), "115.00") {
		t.Fatalf("expected sum of 115.00, got %q", stdout.String())
	}
}

func TestDispatchSumToVariable(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:      "SUM",
		Args:      "AGE",
		ForClause: "AGE >= 35",
		ToClause:  "adultage",
	}); err != nil {
		t.Fatalf("unexpected error on SUM TO: %v", err)
	}

	val, ok := ctx.Variables.Get("ADULTAGE")
	if !ok {
		t.Fatal("expected ADULTAGE memory variable")
	}
	if num, ok := val.(float64); !ok || num != 115 {
		t.Fatalf("expected ADULTAGE=115, got %v", val)
	}
}

func TestDispatchSumMultipleExpressions(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:     "SUM",
		Args:     "AGE, AGE * 2",
		ToClause: "TOTAL, DOUBLE",
	}); err != nil {
		t.Fatalf("unexpected error on SUM multiple: %v", err)
	}
	if !strings.Contains(stdout.String(), "140.00") || !strings.Contains(stdout.String(), "280.00") {
		t.Fatalf("expected both totals in output, got %q", stdout.String())
	}
}
func TestDispatchAverageNoDatabase(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "AVERAGE", Args: "AGE"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchAverageAllRecords(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "AVERAGE", Args: "AGE"}); err != nil {
		t.Fatalf("unexpected error on AVERAGE: %v", err)
	}
	if !strings.Contains(stdout.String(), "35.00") {
		t.Fatalf("expected average of 35.00, got %q", stdout.String())
	}
}

func TestDispatchAverageForClause(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "AVERAGE", Args: "AGE", ForClause: "AGE >= 35"}); err != nil {
		t.Fatalf("unexpected error on AVERAGE FOR: %v", err)
	}
	if !strings.Contains(stdout.String(), "38.33") {
		t.Fatalf("expected average of 38.33, got %q", stdout.String())
	}
}

func TestDispatchAverageNoMatches(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "AVERAGE", Args: "AGE", ForClause: "AGE > 100"}); err != nil {
		t.Fatalf("unexpected error on AVERAGE FOR: %v", err)
	}
	if !strings.Contains(stdout.String(), "0.00") {
		t.Fatalf("expected average of 0.00, got %q", stdout.String())
	}
}

func TestDispatchAverageToVariable(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createLocateTestDBF(t, tempDir)

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{
		Verb:      "AVERAGE",
		Args:      "AGE",
		ForClause: "AGE >= 35",
		ToClause:  "avgage",
	}); err != nil {
		t.Fatalf("unexpected error on AVERAGE TO: %v", err)
	}

	val, ok := ctx.Variables.Get("AVGAGE")
	if !ok {
		t.Fatal("expected AVGAGE memory variable")
	}
	num, ok := val.(float64)
	if !ok || num < 38.32 || num > 38.34 {
		t.Fatalf("expected AVGAGE≈38.33, got %v", val)
	}
}
func TestDispatchQuestion(t *testing.T) {
	var stdout bytes.Buffer
	ctx := testCtx()
	ctx.Stdout = &stdout

	// 1. Number literal
	err := commandMux.Dispatch(ctx, Command{Verb: "?", Args: "42"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.String() != "42\n" {
		t.Errorf("output = %q, want %q", stdout.String(), "42\n")
	}
	stdout.Reset()

	// 2. String literal
	err = commandMux.Dispatch(ctx, Command{Verb: "?", Args: `"hello"`})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.String() != "hello\n" {
		t.Errorf("output = %q, want %q", stdout.String(), "hello\n")
	}
	stdout.Reset()

	// 3. Empty expression
	err = commandMux.Dispatch(ctx, Command{Verb: "?", Args: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.String() != "\n" {
		t.Errorf("output = %q, want %q", stdout.String(), "\n")
	}
	stdout.Reset()

	// 4. Memory variable resolution
	if err := ctx.Variables.Set("FOO", "bar"); err != nil {
		t.Fatalf("set FOO: %v", err)
	}
	err = commandMux.Dispatch(ctx, Command{Verb: "?", Args: "foo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.String() != "bar\n" {
		t.Errorf("output = %q, want %q", stdout.String(), "bar\n")
	}
}

func TestDispatchDoubleQuestion(t *testing.T) {
	var stdout bytes.Buffer
	ctx := testCtx()
	ctx.Stdout = &stdout

	err := commandMux.Dispatch(ctx, Command{Verb: "??", Args: `"no-newline"`})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.String() != "no-newline" {
		t.Errorf("output = %q, want %q", stdout.String(), "no-newline")
	}
	stdout.Reset()

	// Empty double question does nothing
	err = commandMux.Dispatch(ctx, Command{Verb: "??", Args: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("output = %q, want empty", stdout.String())
	}
}

func TestDispatchQuestionErrors(t *testing.T) {
	ctx := testCtx()

	// Syntax error
	err := commandMux.Dispatch(ctx, Command{Verb: "?", Args: "1 +"})
	if err == nil || !strings.Contains(err.Error(), "Syntax error") {
		t.Errorf("expected Syntax error, got %v", err)
	}

	// Evaluation error (missing identifier)
	err = commandMux.Dispatch(ctx, Command{Verb: "?", Args: "notfound"})
	if err == nil || !strings.Contains(err.Error(), "Evaluation error") {
		t.Errorf("expected Evaluation error, got %v", err)
	}
}
func TestDispatchStore(t *testing.T) {
	var stdout bytes.Buffer
	ctx := testCtx()
	ctx.Stdout = &stdout

	err := commandMux.Dispatch(ctx, Command{
		Verb:     "STORE",
		Args:     "42",
		ToClause: "m_var",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val, ok := ctx.Variables.Get("m_var")
	if !ok {
		t.Fatal("expected m_var to be set")
	}
	num, ok := val.(float64)
	if !ok || num != 42 {
		t.Fatalf("expected m_var=42, got %v", val)
	}
	if !strings.Contains(stdout.String(), "42") {
		t.Fatalf("expected talk output with value 42, got %q", stdout.String())
	}
}

func TestDispatchStoreExpression(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := ctx.Variables.Set("NUMBER", float64(3)); err != nil {
		t.Fatalf("set NUMBER: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{
		Verb:     "STORE",
		Args:     "NUMBER + 9",
		ToClause: "NUMBER2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := ctx.Variables.Get("NUMBER2")
	if !ok || val.(float64) != 12 {
		t.Fatalf("expected NUMBER2=12, got %v", val)
	}
}

func TestDispatchStoreStringLiteral(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	err := commandMux.Dispatch(ctx, Command{
		Verb:     "STORE",
		Args:     "'HOWARD'",
		ToClause: "NAME",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := ctx.Variables.Get("NAME")
	if !ok || val.(string) != "HOWARD" {
		t.Fatalf("expected NAME=HOWARD, got %v", val)
	}
}

func TestDispatchStoreMultipleMemvars(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	err := commandMux.Dispatch(ctx, Command{
		Verb:     "STORE",
		Args:     "0",
		ToClause: "i, j, k",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, name := range []string{"I", "J", "K"} {
		val, ok := ctx.Variables.Get(name)
		if !ok || val.(float64) != 0 {
			t.Fatalf("expected %s=0, got %v ok=%v", name, val, ok)
		}
	}
}

func TestDispatchStoreInvalidExpression(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	err := commandMux.Dispatch(ctx, Command{
		Verb:     "STORE",
		Args:     "NUMBER +",
		ToClause: "x",
	})
	if err == nil || !strings.Contains(err.Error(), "Syntax error") {
		t.Fatalf("expected syntax error, got %v", err)
	}
}

func TestDispatchStoreNoTo(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "STORE", Args: "42"})
	if err == nil {
		t.Fatal("expected error for STORE without TO clause")
	}
}

func TestDispatchStoreNoArgs(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "STORE", ToClause: "x"})
	if err == nil {
		t.Fatal("expected error for STORE without expression")
	}
}
