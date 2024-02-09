package repl

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestFieldPictureFromDescriptor(t *testing.T) {
	if got := fieldPicture(dbf.FieldDescriptor{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10}); got != "XXXXXXXXXX" {
		t.Fatalf("char picture = %q", got)
	}
	if got := fieldPicture(dbf.FieldDescriptor{Name: "AGE", Type: dbf.FieldTypeNumeric, Length: 3}); got != "999" {
		t.Fatalf("numeric picture = %q", got)
	}
	if got := fieldPicture(dbf.FieldDescriptor{Name: "SAL", Type: dbf.FieldTypeNumeric, Length: 6, DecimalCount: 2}); got != "999.99" {
		t.Fatalf("decimal picture = %q", got)
	}
	if got := fieldPicture(dbf.FieldDescriptor{Name: "OK", Type: dbf.FieldTypeLogical, Length: 1}); got != "L" {
		t.Fatalf("logical picture = %q", got)
	}
}

func TestBuildAppendScreenRegistersFields(t *testing.T) {
	ctx := testCtx()
	tbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
			{Name: "AGE", Type: dbf.FieldTypeNumeric, Length: 3},
		},
	}
	ctx.GetActiveArea().Alias = "PEOPLE"
	ctx.GetActiveArea().Table = tbl

	if err := buildAppendScreen(ctx, tbl); err != nil {
		t.Fatalf("buildAppendScreen: %v", err)
	}

	fields := ctx.Screen.GetFields()
	if len(fields) != 2 {
		t.Fatalf("expected 2 GET fields, got %d", len(fields))
	}
	if fields[0].Name != "NAME" || fields[0].Row != appendFirstFieldRow || fields[0].Picture != "XXXXXXXXXX" {
		t.Fatalf("unexpected NAME field: %+v", fields[0])
	}
	if fields[1].Name != "AGE" || fields[1].Row != appendFirstFieldRow+1 || fields[1].Picture != "999" {
		t.Fatalf("unexpected AGE field: %+v", fields[1])
	}

	title := screenTextAt(ctx, appendTitleRow, appendTitleCol, 15)
	if !strings.Contains(title, "APPEND - PEOPLE") {
		t.Fatalf("expected append title, got %q", title)
	}
}

func TestCommitReadValueUpdatesActiveRecord(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithRecords(t, tempDir, "commitread.dbf", nil)

	ctx := testCtx()
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	area := ctx.GetActiveArea()
	if err := newBlankActiveRecord(area, area.Table); err != nil {
		t.Fatalf("newBlankActiveRecord: %v", err)
	}
	if err := commitReadValue(ctx, "NAME", "Alice"); err != nil {
		t.Fatalf("commit NAME: %v", err)
	}
	if err := commitReadValue(ctx, "AGE", "25"); err != nil {
		t.Fatalf("commit AGE: %v", err)
	}

	name, err := area.ActiveRecord.DecodeField(area.Table, 0)
	if err != nil {
		t.Fatalf("decode NAME: %v", err)
	}
	if strings.TrimSpace(name.(string)) != "Alice" {
		t.Fatalf("NAME = %q, want Alice", name)
	}
}

func TestRunReadFormCommitsTableFields(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithRecords(t, tempDir, "readform.dbf", nil)

	ctx := testCtx()
	ctx.Stdin = strings.NewReader("Alice\t25\r")
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	area := ctx.GetActiveArea()
	if err := buildAppendScreen(ctx, area.Table); err != nil {
		t.Fatalf("buildAppendScreen: %v", err)
	}
	if err := newBlankActiveRecord(area, area.Table); err != nil {
		t.Fatalf("newBlankActiveRecord: %v", err)
	}
	if err := runReadForm(ctx); err != nil {
		t.Fatalf("runReadForm: %v", err)
	}

	name, err := area.ActiveRecord.DecodeField(area.Table, 0)
	if err != nil {
		t.Fatalf("decode NAME: %v", err)
	}
	if strings.TrimSpace(name.(string)) != "Alice" {
		t.Fatalf("NAME = %q, want Alice", name)
	}
}

func TestRunAppendScreenAppendsRecord(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithRecords(t, tempDir, "appendscreen.dbf", nil)

	ctx := testCtx()
	ctx.Stdin = strings.NewReader("Alice\t25\r\r")
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	if err := runAppendScreen(ctx); err != nil {
		t.Fatalf("runAppendScreen: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table.Header.RecordCount != 1 {
		t.Fatalf("expected 1 record, got %d", area.Table.Header.RecordCount)
	}

	wseeker := area.Table.Underlying().(io.ReadSeeker)
	rec, err := area.Table.ReadRecordAt(wseeker, 0)
	if err != nil {
		t.Fatalf("read appended record: %v", err)
	}

	name, err := rec.DecodeField(area.Table, 0)
	if err != nil {
		t.Fatalf("decode NAME: %v", err)
	}
	if strings.TrimSpace(name.(string)) != "Alice" {
		t.Fatalf("NAME = %q, want Alice", name)
	}
}

func TestRunAppendScreenBlankRecordExits(t *testing.T) {
	tempDir := t.TempDir()
	dbfPath := createTempDBFWithRecords(t, tempDir, "appendblank.dbf", nil)

	ctx := testCtx()
	ctx.Stdin = strings.NewReader("\r")
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	if err := runAppendScreen(ctx); err != nil {
		t.Fatalf("runAppendScreen: %v", err)
	}

	if ctx.GetActiveArea().Table.Header.RecordCount != 0 {
		t.Fatalf("expected 0 records, got %d", ctx.GetActiveArea().Table.Header.RecordCount)
	}
}

func TestIsBlankAppendRecord(t *testing.T) {
	tbl := &dbf.Table{
		Header: &dbf.Header{RecordLen: 14},
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
			{Name: "AGE", Type: dbf.FieldTypeNumeric, Length: 3},
		},
		Offset: []int{0, 10, 13},
	}

	rec, err := dbf.NewRecord(tbl, false, []interface{}{"", float64(0)})
	if err != nil {
		t.Fatalf("NewRecord: %v", err)
	}
	blank, err := isBlankAppendRecord(tbl, rec)
	if err != nil || !blank {
		t.Fatalf("expected blank record, blank=%v err=%v", blank, err)
	}

	rec, err = dbf.NewRecord(tbl, false, []interface{}{"Ann", float64(0)})
	if err != nil {
		t.Fatalf("NewRecord: %v", err)
	}
	blank, err = isBlankAppendRecord(tbl, rec)
	if err != nil || blank {
		t.Fatalf("expected non-blank record, blank=%v err=%v", blank, err)
	}
}
