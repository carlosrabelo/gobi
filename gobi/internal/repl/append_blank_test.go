package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/ndx"
)

func TestDispatchAppendBlankAddsEmptyRecord(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "APPEND", Args: "BLANK"}); err != nil {
		t.Fatalf("APPEND BLANK: %v", err)
	}

	area := ctx.GetActiveArea()
	if int(area.Table.Header.RecordCount) != 2 {
		t.Fatalf("record count = %d, want 2", area.Table.Header.RecordCount)
	}
	if area.RecordNo != 1 {
		t.Fatalf("record index = %d, want 1 (new record current)", area.RecordNo)
	}

	name, err := area.ActiveRecord.DecodeField(area.Table, 0)
	if err != nil {
		t.Fatalf("DecodeField: %v", err)
	}
	if name != "" {
		t.Fatalf("NAME = %q, want blank", name)
	}
}

func TestDispatchAppendBlankUpdatesIndexes(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{
		Verb:     "INDEX",
		Args:     "ON NAME",
		ToClause: "byname",
	}); err != nil {
		t.Fatalf("INDEX: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "APPEND", Args: "blank"}); err != nil {
		t.Fatalf("APPEND BLANK: %v", err)
	}

	area := ctx.GetActiveArea()
	pm := area.Indexes[0].Manager()
	alice, found, err := pm.SearchExact(ndx.Key("Alice"))
	if err != nil {
		t.Fatalf("SearchExact Alice: %v", err)
	}
	if !found || alice.RecordNumber != 1 {
		t.Fatalf("Alice mapping = %#v found=%v, want record 1", alice, found)
	}
	blank, found, err := pm.SearchExact(ndx.Key(""))
	if err != nil {
		t.Fatalf("SearchExact blank: %v", err)
	}
	if !found || blank.RecordNumber != 2 {
		t.Fatalf("blank mapping = %#v found=%v, want record 2", blank, found)
	}
}

func TestDispatchAppendBlankWithoutDatabaseFails(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "APPEND", Args: "BLANK"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no-database error, got %v", err)
	}
}

func TestDispatchAppendUnknownArgumentStillFails(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "APPEND", Args: "XYZ"})
	if err == nil || !strings.Contains(err.Error(), "APPEND requires FROM") {
		t.Fatalf("expected FROM error, got %v", err)
	}
}
