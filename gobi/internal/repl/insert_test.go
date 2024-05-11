package repl

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func insertTestRecords() [][]byte {
	alice := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	bob := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 35")...)...)
	return [][]byte{alice, bob}
}

func TestDispatchInsertBlankAfterCurrent(t *testing.T) {
	tempDir := t.TempDir()
	createTempDBFWithRecords(t, tempDir, "people.dbf", insertTestRecords())

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	// Pointer starts at record 1 (Alice); blank goes between Alice and Bob.
	if err := commandMux.Dispatch(ctx, Command{Verb: "INSERT", Args: "BLANK"}); err != nil {
		t.Fatalf("INSERT BLANK: %v", err)
	}

	area := ctx.GetActiveArea()
	if int(area.Table.Header.RecordCount) != 3 {
		t.Fatalf("record count = %d, want 3", area.Table.Header.RecordCount)
	}
	if area.RecordNo != 1 {
		t.Fatalf("record index = %d, want 1 (inserted record current)", area.RecordNo)
	}

	want := []string{"Alice", "", "Bob"}
	for i, expected := range want {
		rec, err := area.Table.ReadRecordAt(area.Table.Underlying().(io.ReadSeeker), i)
		if err != nil {
			t.Fatalf("ReadRecordAt(%d): %v", i, err)
		}
		name, err := rec.DecodeField(area.Table, 0)
		if err != nil {
			t.Fatalf("DecodeField(%d): %v", i, err)
		}
		if name != expected {
			t.Fatalf("record %d NAME = %q, want %q", i+1, name, expected)
		}
	}
}

func TestDispatchInsertBeforeBlank(t *testing.T) {
	tempDir := t.TempDir()
	createTempDBFWithRecords(t, tempDir, "people.dbf", insertTestRecords())

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "INSERT", Args: "BEFORE BLANK"}); err != nil {
		t.Fatalf("INSERT BEFORE BLANK: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.RecordNo != 0 {
		t.Fatalf("record index = %d, want 0", area.RecordNo)
	}

	rec, err := area.Table.ReadRecordAt(area.Table.Underlying().(io.ReadSeeker), 0)
	if err != nil {
		t.Fatalf("ReadRecordAt(0): %v", err)
	}
	name, _ := rec.DecodeField(area.Table, 0)
	if name != "" {
		t.Fatalf("record 1 NAME = %q, want blank", name)
	}

	rec, err = area.Table.ReadRecordAt(area.Table.Underlying().(io.ReadSeeker), 1)
	if err != nil {
		t.Fatalf("ReadRecordAt(1): %v", err)
	}
	name, _ = rec.DecodeField(area.Table, 0)
	if name != "Alice" {
		t.Fatalf("record 2 NAME = %q, want Alice", name)
	}
}

func TestDispatchInsertPromptsForValues(t *testing.T) {
	tempDir := t.TempDir()
	createTempDBFWithRecords(t, tempDir, "people.dbf", insertTestRecords())

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("Carol\n30\n")

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "INSERT"}); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	area := ctx.GetActiveArea()
	if int(area.Table.Header.RecordCount) != 3 {
		t.Fatalf("record count = %d, want 3", area.Table.Header.RecordCount)
	}

	rec, err := area.Table.ReadRecordAt(area.Table.Underlying().(io.ReadSeeker), 1)
	if err != nil {
		t.Fatalf("ReadRecordAt(1): %v", err)
	}
	name, _ := rec.DecodeField(area.Table, 0)
	if name != "Carol" {
		t.Fatalf("record 2 NAME = %q, want Carol", name)
	}
	age, _ := rec.DecodeField(area.Table, 1)
	if age != 30.0 {
		t.Fatalf("record 2 AGE = %v, want 30", age)
	}

	out := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "NAME ? ") {
		t.Fatalf("expected NAME prompt, got %q", out)
	}
}

func TestDispatchInsertEmptyFirstFieldCancels(t *testing.T) {
	tempDir := t.TempDir()
	createTempDBFWithRecords(t, tempDir, "people.dbf", insertTestRecords())

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("\n")

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "INSERT"}); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	area := ctx.GetActiveArea()
	if int(area.Table.Header.RecordCount) != 2 {
		t.Fatalf("record count = %d, want 2 (insert cancelled)", area.Table.Header.RecordCount)
	}
}

func TestDispatchInsertRebuildsIndexes(t *testing.T) {
	tempDir := t.TempDir()
	createTempDBFWithRecords(t, tempDir, "people.dbf", insertTestRecords())

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

	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TOP"}); err != nil {
		t.Fatalf("GO TOP: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "INSERT", Args: "BEFORE BLANK"}); err != nil {
		t.Fatalf("INSERT BEFORE BLANK: %v", err)
	}

	area := ctx.GetActiveArea()
	records, err := area.Indexes[0].Manager().OrderedRecordNumbers()
	if err != nil {
		t.Fatalf("OrderedRecordNumbers: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("index entries = %d, want 3", len(records))
	}
	// Blank key sorts first; it is now physical record 1.
	if records[0] != 1 {
		t.Fatalf("index order = %v, want blank record (1) first", records)
	}
}

func TestDispatchInsertWithoutDatabaseFails(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "INSERT", Args: "BLANK"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no-database error, got %v", err)
	}
}

func TestDispatchInsertUnknownArgumentFails(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "INSERT", Args: "XYZ"})
	if err == nil || !strings.Contains(err.Error(), "Unexpected argument") {
		t.Fatalf("expected argument error, got %v", err)
	}
}

func TestParseInsertArgs(t *testing.T) {
	tests := []struct {
		args   string
		before bool
		blank  bool
		bad    bool
	}{
		{"", false, false, false},
		{"BLANK", false, true, false},
		{"BEFORE", true, false, false},
		{"BEFORE BLANK", true, true, false},
		{"blank before", true, true, false},
		{"NOPE", false, false, true},
	}
	for _, tt := range tests {
		before, blank, err := parseInsertArgs(tt.args)
		if tt.bad {
			if err == nil {
				t.Errorf("parseInsertArgs(%q): expected error", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseInsertArgs(%q): %v", tt.args, err)
			continue
		}
		if before != tt.before || blank != tt.blank {
			t.Errorf("parseInsertArgs(%q) = (%v, %v), want (%v, %v)",
				tt.args, before, blank, tt.before, tt.blank)
		}
	}
}
