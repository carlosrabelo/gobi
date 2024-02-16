package repl

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestBrowseColumnWidths(t *testing.T) {
	tbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
			{Name: "AGE", Type: dbf.FieldTypeNumeric, Length: 3},
		},
	}
	widths := browseColumnWidths(tbl)
	if widths[0] != 10 || widths[1] != 3 {
		t.Fatalf("unexpected widths: %v", widths)
	}
}

func TestPadBrowseCellPadsWidth(t *testing.T) {
	got := padBrowseCell("Alice", 5)
	if got != "Alice" {
		t.Fatalf("expected padded cell, got %q", got)
	}
}

func TestBrowseMatrixRendering(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "browse.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	area := ctx.GetActiveArea()
	rseeker := area.Table.Underlying().(io.ReadSeeker)
	s := newBrowseSession(ctx, area, rseeker)
	if err := s.draw(); err != nil {
		t.Fatalf("draw: %v", err)
	}

	header := screenTextAt(ctx, browseHeaderRow, browseLabelCol, 20)
	if !strings.Contains(header, "Rec#") || !strings.Contains(header, "NAME") {
		t.Fatalf("expected header row, got %q", header)
	}

	row := screenTextAt(ctx, browseFirstDataRow, browseLabelCol, 30)
	if !strings.Contains(row, "25") || !strings.Contains(row, "Alice") {
		t.Fatalf("expected data row, got %q", row)
	}

	title := screenTextAt(ctx, browseTitleRow, browseTitleCol, 20)
	if !strings.Contains(title, "BROWSE") {
		t.Fatalf("expected browse title, got %q", title)
	}
}

func TestBrowseCursorMovement(t *testing.T) {
	tempDir := t.TempDir()
	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 30")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "browsemove.dbf", [][]byte{rec1, rec2})

	ctx := testCtx()
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	area := ctx.GetActiveArea()
	s := newBrowseSession(ctx, area, area.Table.Underlying().(io.ReadSeeker))
	s.curRec = 0
	s.curCol = 0

	s.moveDown()
	if s.curRec != 1 {
		t.Fatalf("expected record 1 after down, got %d", s.curRec)
	}

	s.moveRight()
	if s.curCol != 1 {
		t.Fatalf("expected column 1 after right, got %d", s.curCol)
	}

	s.moveUp()
	if s.curRec != 0 {
		t.Fatalf("expected record 0 after up, got %d", s.curRec)
	}

	s.moveLeft()
	if s.curCol != 0 {
		t.Fatalf("expected column 0 after left, got %d", s.curCol)
	}
}

func TestRunBrowseMatrixArrowKeysMoveCursor(t *testing.T) {
	tempDir := t.TempDir()
	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 30")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "browsekeys.dbf", [][]byte{rec1, rec2})

	ctx := testCtx()
	ctx.Stdin = strings.NewReader(string([]byte{replKeyDown, replKeyRight, editKeyEsc}))
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	if err := runBrowseMatrix(ctx, ctx.GetActiveArea().Table.Underlying().(io.ReadSeeker)); err != nil {
		t.Fatalf("runBrowseMatrix: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.RecordNo != 1 {
		t.Fatalf("expected record pointer 1, got %d", area.RecordNo)
	}
}

func TestBrowseInlineCellEdit(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "browseedit.dbf", [][]byte{rec})

	ctx := testCtx()
	input := string([]byte{editKeyEnter, editKeyCtrlY, 'B', 'o', 'b', editKeyEnter, editKeyEsc})
	ctx.Stdin = strings.NewReader(input)
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	if err := runBrowseMatrix(ctx, ctx.GetActiveArea().Table.Underlying().(io.ReadSeeker)); err != nil {
		t.Fatalf("runBrowseMatrix: %v", err)
	}

	area := ctx.GetActiveArea()
	updated, err := area.Table.ReadRecordAt(area.Table.Underlying().(io.ReadSeeker), 0)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	name, err := updated.DecodeField(area.Table, 0)
	if err != nil {
		t.Fatalf("decode NAME: %v", err)
	}
	if name.(string) != "Bob" {
		t.Fatalf("NAME = %q, want Bob", name)
	}
}

func TestBrowseToggleRecordDelete(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "browsedel.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Stdin = strings.NewReader(string([]byte{editKeyCtrlU, editKeyEsc}))
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	if err := runBrowseMatrix(ctx, ctx.GetActiveArea().Table.Underlying().(io.ReadSeeker)); err != nil {
		t.Fatalf("runBrowseMatrix: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.ActiveRecord == nil || !area.ActiveRecord.Deleted {
		t.Fatal("expected active record to be marked deleted after BROWSE")
	}
}

func TestBrowseEditCancelRestoresValue(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "browsecancel.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Stdin = strings.NewReader(string([]byte{editKeyEnter, 'X', editKeyEsc, editKeyEsc}))
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	if err := runBrowseMatrix(ctx, ctx.GetActiveArea().Table.Underlying().(io.ReadSeeker)); err != nil {
		t.Fatalf("runBrowseMatrix: %v", err)
	}

	area := ctx.GetActiveArea()
	unchanged, err := area.Table.ReadRecordAt(area.Table.Underlying().(io.ReadSeeker), 0)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	name, err := unchanged.DecodeField(area.Table, 0)
	if err != nil {
		t.Fatalf("decode NAME: %v", err)
	}
	if name.(string) != "Alice" {
		t.Fatalf("NAME = %q, want Alice", name)
	}
}

func TestBrowseInsertEditCharRespectsWidth(t *testing.T) {
	s := &browseSession{
		tbl: &dbf.Table{
			Header: &dbf.Header{RecordCount: 1},
			Fields: []dbf.FieldDescriptor{
				{Name: "CODE", Type: dbf.FieldTypeChar, Length: 3},
			},
			Offset: []int{0, 3},
		},
		curRec:     0,
		curCol:     0,
		editing:    true,
		editValue:  "AB",
		editCursor: 2,
	}

	if err := s.insertEditChar('C'); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.editValue != "ABC" {
		t.Fatalf("value = %q, want ABC", s.editValue)
	}

	if err := s.insertEditChar('D'); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.editValue != "ABC" {
		t.Fatalf("value = %q, want ABC after overflow", s.editValue)
	}
}

func TestRunBrowseMatrixExitsOnEsc(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "browseexit.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Stdin = strings.NewReader(string([]byte{editKeyEsc}))
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	if err := runBrowseMatrix(ctx, ctx.GetActiveArea().Table.Underlying().(io.ReadSeeker)); err != nil {
		t.Fatalf("runBrowseMatrix: %v", err)
	}
}

func TestDispatchBrowseRequiresDatabase(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "BROWSE"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}
