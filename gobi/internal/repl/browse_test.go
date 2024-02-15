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
