package repl

import (
	"strings"
	"testing"
)

func TestParseUseArgs(t *testing.T) {
	filename, indexes, err := parseUseArgs("")
	if err != nil || filename != "" || indexes != nil {
		t.Fatalf("empty args: %q %#v %v", filename, indexes, err)
	}

	filename, indexes, err = parseUseArgs("people")
	if err != nil || filename != "people" || indexes != nil {
		t.Fatalf("filename only: %q %#v %v", filename, indexes, err)
	}

	filename, indexes, err = parseUseArgs("people INDEX byname, bycity")
	if err != nil || filename != "people" {
		t.Fatalf("with indexes: %q %v", filename, err)
	}
	if len(indexes) != 2 || indexes[0] != "byname" || indexes[1] != "bycity" {
		t.Fatalf("unexpected index names: %#v", indexes)
	}

	filename, indexes, err = parseUseArgs("people index byname")
	if err != nil || filename != "people" || len(indexes) != 1 || indexes[0] != "byname" {
		t.Fatalf("lowercase index keyword: %q %#v %v", filename, indexes, err)
	}

	if _, _, err = parseUseArgs("people INDEX"); err == nil {
		t.Fatal("expected error for INDEX without names")
	}

	if _, _, err = parseUseArgs("people extra"); err == nil {
		t.Fatal("expected error for unexpected argument")
	}
}

func TestDispatchUseWithIndexBindsExistingIndex(t *testing.T) {
	tempDir := t.TempDir()
	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 30")...)...)
	createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec1, rec2})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false

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

	// Close everything, then rebind the existing index through USE ... INDEX.
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: ""}); err != nil {
		t.Fatalf("USE close: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people INDEX byname"}); err != nil {
		t.Fatalf("USE with INDEX: %v", err)
	}

	area := ctx.GetActiveArea()
	if len(area.Indexes) != 1 {
		t.Fatalf("expected 1 bound index, got %d", len(area.Indexes))
	}
	if area.Indexes[0].Manager().Header().Expression != "NAME" {
		t.Fatalf("unexpected index expression: %#v", area.Indexes[0].Manager().Header())
	}

	// The bound index must serve FIND immediately.
	if err := commandMux.Dispatch(ctx, Command{Verb: "FIND", Args: "Bob"}); err != nil {
		t.Fatalf("FIND: %v", err)
	}
	if !area.Found || area.RecordNo != 1 {
		t.Fatalf("expected Bob at record index 1, got found=%v recno=%d", area.Found, area.RecordNo)
	}
}

func TestDispatchUseWithMultipleIndexesKeepsOrder(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	for _, idx := range []struct{ expr, file string }{
		{"NAME", "byname"},
		{"AGE", "byage"},
	} {
		if err := commandMux.Dispatch(ctx, Command{
			Verb:     "INDEX",
			Args:     "ON " + idx.expr,
			ToClause: idx.file,
		}); err != nil {
			t.Fatalf("INDEX %s: %v", idx.file, err)
		}
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people INDEX byname, byage"}); err != nil {
		t.Fatalf("USE with INDEX list: %v", err)
	}

	area := ctx.GetActiveArea()
	if len(area.Indexes) != 2 {
		t.Fatalf("expected 2 bound indexes, got %d", len(area.Indexes))
	}
	if area.Indexes[0].Manager().Header().Expression != "NAME" ||
		area.Indexes[1].Manager().Header().Expression != "AGE" {
		t.Fatalf("indexes out of order: %#v, %#v",
			area.Indexes[0].Manager().Header(), area.Indexes[1].Manager().Header())
	}
}

func TestDispatchUseWithMissingIndexClosesTable(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir

	err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people INDEX missing"})
	if err == nil || !strings.Contains(err.Error(), "Index file not found") {
		t.Fatalf("expected missing index error, got %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table != nil {
		t.Fatal("expected table to be closed after index binding failure")
	}
	if len(area.Indexes) != 0 {
		t.Fatalf("expected no bound indexes, got %d", len(area.Indexes))
	}
}
