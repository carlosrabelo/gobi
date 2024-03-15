package repl

import (
	"strings"
	"testing"
)

func TestSplitIndexNames(t *testing.T) {
	names := splitIndexNames("byname, bycity,, byage ")
	if len(names) != 3 || names[0] != "byname" || names[1] != "bycity" || names[2] != "byage" {
		t.Fatalf("unexpected names: %#v", names)
	}
	if names = splitIndexNames(""); names != nil {
		t.Fatalf("expected nil for empty list, got %#v", names)
	}
}

func TestDispatchSetIndexToBindsIndexes(t *testing.T) {
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

	// Rebind in reverse order; AGE becomes the controlling index.
	if err := commandMux.Dispatch(ctx, Command{
		Verb:     "SET",
		Args:     "INDEX",
		ToClause: "byage, byname",
	}); err != nil {
		t.Fatalf("SET INDEX TO: %v", err)
	}

	area := ctx.GetActiveArea()
	if len(area.Indexes) != 2 {
		t.Fatalf("expected 2 bound indexes, got %d", len(area.Indexes))
	}
	if area.Indexes[0].Manager().Header().Expression != "AGE" ||
		area.Indexes[1].Manager().Header().Expression != "NAME" {
		t.Fatalf("indexes out of order: %#v, %#v",
			area.Indexes[0].Manager().Header(), area.Indexes[1].Manager().Header())
	}

	// FIND must use the new controlling NAME index after another rebind.
	if err := commandMux.Dispatch(ctx, Command{
		Verb:     "SET",
		Args:     "INDEX",
		ToClause: "byname",
	}); err != nil {
		t.Fatalf("SET INDEX TO byname: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "FIND", Args: "Bob"}); err != nil {
		t.Fatalf("FIND: %v", err)
	}
	if !area.Found || area.RecordNo != 1 {
		t.Fatalf("expected Bob at record index 1, got found=%v recno=%d", area.Found, area.RecordNo)
	}
}

func TestDispatchSetIndexToEmptyClosesIndexes(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

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

	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "INDEX"}); err != nil {
		t.Fatalf("SET INDEX TO: %v", err)
	}

	area := ctx.GetActiveArea()
	if len(area.Indexes) != 0 {
		t.Fatalf("expected no bound indexes, got %d", len(area.Indexes))
	}
}

func TestDispatchSetIndexWithoutDatabaseFails(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "INDEX", ToClause: "byname"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no-database error, got %v", err)
	}
}

func TestDispatchSetIndexMissingFileClearsIndexes(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

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

	err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "INDEX", ToClause: "missing"})
	if err == nil || !strings.Contains(err.Error(), "Index file not found") {
		t.Fatalf("expected missing index error, got %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table == nil {
		t.Fatal("expected table to stay open after SET INDEX failure")
	}
	if len(area.Indexes) != 0 {
		t.Fatalf("expected no bound indexes, got %d", len(area.Indexes))
	}
}

func TestDispatchSetDefaultWithToClause(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &strings.Builder{}

	err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "DEFAULT", ToClause: "/tmp/data"})
	if err != nil {
		t.Fatalf("SET DEFAULT: %v", err)
	}
	if ctx.Config.DefaultDir != "/tmp/data" {
		t.Fatalf("DefaultDir = %q, want /tmp/data", ctx.Config.DefaultDir)
	}
}
