package repl

import (
	"bytes"
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
