package repl

import (
	"strings"
	"testing"
)

func TestParseFindString(t *testing.T) {
	text, err := parseFindString(`"Al"`)
	if err != nil || text != "Al" {
		t.Fatalf("text = %q err=%v", text, err)
	}

	text, err = parseFindString("Bob")
	if err != nil || text != "Bob" {
		t.Fatalf("text = %q err=%v", text, err)
	}

	_, err = parseFindString("")
	if err == nil {
		t.Fatal("expected empty string error")
	}
}

func TestDispatchSeekExactMatch(t *testing.T) {
	tempDir := t.TempDir()
	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 30")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec1, rec2})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{
		Verb:     "INDEX",
		Args:     "ON NAME",
		ToClause: "people",
	}); err != nil {
		t.Fatalf("INDEX: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SEEK", Args: `"Bob"`}); err != nil {
		t.Fatalf("SEEK: %v", err)
	}

	area := ctx.GetActiveArea()
	if !area.Found {
		t.Fatal("expected FOUND true after SEEK")
	}
	if area.RecordNo != 1 {
		t.Fatalf("record index = %d, want 1", area.RecordNo)
	}
	if area.ActiveRecord == nil {
		t.Fatal("expected active record loaded")
	}
}

func TestDispatchSeekNotFound(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{
		Verb:     "INDEX",
		Args:     "ON NAME",
		ToClause: "people",
	}); err != nil {
		t.Fatalf("INDEX: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TOP"}); err != nil {
		t.Fatalf("GO TOP: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SEEK", Args: `"Zed"`}); err != nil {
		t.Fatalf("SEEK: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Found {
		t.Fatal("expected FOUND false after failed SEEK")
	}
	if area.ActiveRecord != nil {
		t.Fatal("expected no active record after failed SEEK")
	}
	if area.RecordNo != 1 {
		t.Fatalf("record index = %d, want EOF at 1", area.RecordNo)
	}
}

func TestDispatchFindPrefixMatch(t *testing.T) {
	tempDir := t.TempDir()
	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Amy       "), []byte(" 28")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec1, rec2})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{
		Verb:     "INDEX",
		Args:     "ON NAME",
		ToClause: "people",
	}); err != nil {
		t.Fatalf("INDEX: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "FIND", Args: `"Am"`}); err != nil {
		t.Fatalf("FIND: %v", err)
	}

	area := ctx.GetActiveArea()
	if !area.Found || area.RecordNo != 1 {
		t.Fatalf("expected Amy at record index 1, found=%v recordNo=%d", area.Found, area.RecordNo)
	}
}

func TestDispatchFindRequiresIndex(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

	ctx := testCtx()
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "FIND", Args: `"A"`})
	if err == nil || !strings.Contains(err.Error(), "No index files are in use") {
		t.Fatalf("expected no index error, got %v", err)
	}
}

func TestDispatchSeekRequiresIndex(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

	ctx := testCtx()
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "SEEK", Args: `"Alice"`})
	if err == nil || !strings.Contains(err.Error(), "No index files are in use") {
		t.Fatalf("expected no index error, got %v", err)
	}
}
