package repl

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func changeTestRecords() [][]byte {
	alice := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	bob := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 35")...)...)
	return [][]byte{alice, bob}
}

func changeTestCtx(t *testing.T, stdin string) *context.Context {
	t.Helper()
	tempDir := t.TempDir()
	createTempDBFWithRecords(t, tempDir, "people.dbf", changeTestRecords())

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader(stdin)

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	return ctx
}

func TestDispatchChangeCurrentRecordField(t *testing.T) {
	ctx := changeTestCtx(t, "Ali\nXyz\n")

	if err := commandMux.Dispatch(ctx, Command{Verb: "CHANGE", Args: "FIELD NAME"}); err != nil {
		t.Fatalf("CHANGE: %v", err)
	}

	area := ctx.GetActiveArea()
	rec, err := area.Table.ReadRecordAt(area.Table.Underlying().(io.ReadSeeker), 0)
	if err != nil {
		t.Fatalf("ReadRecordAt: %v", err)
	}
	name, _ := rec.DecodeField(area.Table, 0)
	if name != "Xyzce" {
		t.Fatalf("NAME = %q, want Xyzce", name)
	}

	out := ctx.Stdout.(*bytes.Buffer).String()
	for _, want := range []string{"RECORD: 00001", "NAME:  Alice", "CHANGE? ", "TO? "} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q, got %q", want, out)
		}
	}
}

func TestDispatchChangeAllSkipsOnEmptyAnswer(t *testing.T) {
	// Record 1: skip NAME (empty). Record 2: replace Bob with Rob.
	ctx := changeTestCtx(t, "\nBob\nRob\n")

	if err := commandMux.Dispatch(ctx, Command{Verb: "CHANGE", Args: "ALL FIELD NAME"}); err != nil {
		t.Fatalf("CHANGE ALL: %v", err)
	}

	area := ctx.GetActiveArea()
	rseeker := area.Table.Underlying().(io.ReadSeeker)

	rec, _ := area.Table.ReadRecordAt(rseeker, 0)
	name, _ := rec.DecodeField(area.Table, 0)
	if name != "Alice" {
		t.Fatalf("record 1 NAME = %q, want Alice (unchanged)", name)
	}

	rec, _ = area.Table.ReadRecordAt(rseeker, 1)
	name, _ = rec.DecodeField(area.Table, 0)
	if name != "Rob" {
		t.Fatalf("record 2 NAME = %q, want Rob", name)
	}
}

func TestDispatchChangeMultipleFields(t *testing.T) {
	// NAME: skip; AGE: change 2 to 3 (25 -> 35).
	ctx := changeTestCtx(t, "\n2\n3\n")

	if err := commandMux.Dispatch(ctx, Command{Verb: "CHANGE", Args: "FIELD NAME,AGE"}); err != nil {
		t.Fatalf("CHANGE: %v", err)
	}

	area := ctx.GetActiveArea()
	rec, _ := area.Table.ReadRecordAt(area.Table.Underlying().(io.ReadSeeker), 0)
	age, _ := rec.DecodeField(area.Table, 1)
	if age != 35.0 {
		t.Fatalf("AGE = %v, want 35", age)
	}
}

func TestDispatchChangeSubstringNotFound(t *testing.T) {
	ctx := changeTestCtx(t, "Zed\nNew\n")

	if err := commandMux.Dispatch(ctx, Command{Verb: "CHANGE", Args: "FIELD NAME"}); err != nil {
		t.Fatalf("CHANGE: %v", err)
	}

	area := ctx.GetActiveArea()
	rec, _ := area.Table.ReadRecordAt(area.Table.Underlying().(io.ReadSeeker), 0)
	name, _ := rec.DecodeField(area.Table, 0)
	if name != "Alice" {
		t.Fatalf("NAME = %q, want Alice (unchanged)", name)
	}

	out := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "Zed not found") {
		t.Fatalf("expected not-found message, got %q", out)
	}
}

func TestDispatchChangeWithForClause(t *testing.T) {
	// Only Bob matches; replace Bob with Rod.
	ctx := changeTestCtx(t, "Bob\nRod\n")

	if err := commandMux.Dispatch(ctx, Command{
		Verb:      "CHANGE",
		Args:      "FIELD NAME",
		ForClause: "AGE > 30",
	}); err != nil {
		t.Fatalf("CHANGE FOR: %v", err)
	}

	area := ctx.GetActiveArea()
	rseeker := area.Table.Underlying().(io.ReadSeeker)

	rec, _ := area.Table.ReadRecordAt(rseeker, 0)
	name, _ := rec.DecodeField(area.Table, 0)
	if name != "Alice" {
		t.Fatalf("record 1 NAME = %q, want Alice (unchanged)", name)
	}

	rec, _ = area.Table.ReadRecordAt(rseeker, 1)
	name, _ = rec.DecodeField(area.Table, 0)
	if name != "Rod" {
		t.Fatalf("record 2 NAME = %q, want Rod", name)
	}
}

func TestDispatchChangeEOFStopsScan(t *testing.T) {
	// Input ends after the first record's NAME prompt.
	ctx := changeTestCtx(t, "")

	if err := commandMux.Dispatch(ctx, Command{Verb: "CHANGE", Args: "ALL FIELD NAME"}); err != nil {
		t.Fatalf("CHANGE: %v", err)
	}

	area := ctx.GetActiveArea()
	rec, _ := area.Table.ReadRecordAt(area.Table.Underlying().(io.ReadSeeker), 0)
	name, _ := rec.DecodeField(area.Table, 0)
	if name != "Alice" {
		t.Fatalf("NAME = %q, want Alice (unchanged)", name)
	}
}

func TestDispatchChangeRequiresFieldList(t *testing.T) {
	ctx := changeTestCtx(t, "")

	err := commandMux.Dispatch(ctx, Command{Verb: "CHANGE"})
	if err == nil || !strings.Contains(err.Error(), "CHANGE requires a FIELD list") {
		t.Fatalf("expected FIELD list error, got %v", err)
	}
}

func TestDispatchChangeUnknownField(t *testing.T) {
	ctx := changeTestCtx(t, "")

	err := commandMux.Dispatch(ctx, Command{Verb: "CHANGE", Args: "FIELD NOPE"})
	if err == nil || !strings.Contains(err.Error(), "Unknown field: NOPE") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestDispatchChangeWithoutDatabaseFails(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "CHANGE", Args: "FIELD NAME"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no-database error, got %v", err)
	}
}
