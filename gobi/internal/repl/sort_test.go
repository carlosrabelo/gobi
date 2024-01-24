package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestParseSortOnExpression(t *testing.T) {
	expr, err := parseSortOnExpression("ON NAME")
	if err != nil || expr != "NAME" {
		t.Fatalf("expr = %q err=%v", expr, err)
	}

	_, err = parseSortOnExpression("NAME")
	if err == nil {
		t.Fatal("expected ON prefix error")
	}
}

func TestDispatchSortOnRequiresDatabase(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{
		Verb:     "SORT",
		Args:     "ON NAME",
		ToClause: "sorted",
	})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchSortOnRequiresToClause(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

	ctx := testCtx()
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "SORT", Args: "ON NAME"})
	if err == nil || !strings.Contains(err.Error(), "TO") {
		t.Fatalf("expected TO clause error, got %v", err)
	}
}

func TestDispatchSortOnCreatesSortedDBF(t *testing.T) {
	tempDir := t.TempDir()
	rec1 := append([]byte{0x20}, append([]byte("Charlie   "), []byte(" 45")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec3 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 30")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec1, rec2, rec3})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdin = strings.NewReader("Y\n")

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{
		Verb:     "SORT",
		Args:     "ON NAME",
		ToClause: "sorted",
	}); err != nil {
		t.Fatalf("SORT: %v", err)
	}

	sortedPath := filepath.Join(tempDir, "sorted.dbf")
	if _, err := os.Stat(sortedPath); err != nil {
		t.Fatalf("expected sorted file: %v", err)
	}

	f, err := os.Open(sortedPath)
	if err != nil {
		t.Fatalf("open sorted: %v", err)
	}
	defer f.Close()

	tbl, err := dbf.Open(f)
	if err != nil {
		t.Fatalf("open sorted table: %v", err)
	}
	if tbl.Header.RecordCount != 3 {
		t.Fatalf("record count = %d, want 3", tbl.Header.RecordCount)
	}

	names := []string{"Alice", "Bob", "Charlie"}
	for i, want := range names {
		rec, err := tbl.ReadRecordAt(f, i)
		if err != nil {
			t.Fatalf("read record %d: %v", i, err)
		}
		val, err := rec.DecodeField(tbl, 0)
		if err != nil {
			t.Fatalf("decode NAME %d: %v", i, err)
		}
		got, ok := val.(string)
		if !ok || strings.TrimSpace(got) != want {
			t.Fatalf("record %d NAME = %q, want %q", i, got, want)
		}
	}
}

func TestDispatchSortOnNumericKey(t *testing.T) {
	tempDir := t.TempDir()
	rec1 := append([]byte{0x20}, append([]byte("Charlie   "), []byte(" 45")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec3 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 30")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec1, rec2, rec3})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdin = strings.NewReader("Y\n")

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{
		Verb:     "SORT",
		Args:     "ON AGE",
		ToClause: "byage",
	}); err != nil {
		t.Fatalf("SORT: %v", err)
	}

	f, err := os.Open(filepath.Join(tempDir, "byage.dbf"))
	if err != nil {
		t.Fatalf("open sorted: %v", err)
	}
	defer f.Close()

	tbl, err := dbf.Open(f)
	if err != nil {
		t.Fatalf("open sorted table: %v", err)
	}

	ages := []string{"25", "30", "45"}
	for i, want := range ages {
		rec, err := tbl.ReadRecordAt(f, i)
		if err != nil {
			t.Fatalf("read record %d: %v", i, err)
		}
		val, err := rec.DecodeField(tbl, 1)
		if err != nil {
			t.Fatalf("decode AGE %d: %v", i, err)
		}
		got, ok := val.(float64)
		if !ok || int(got) != atoi(want) {
			t.Fatalf("record %d AGE = %v, want %s", i, val, want)
		}
	}
}

func atoi(s string) int {
	n := 0
	for _, ch := range s {
		n = n*10 + int(ch-'0')
	}
	return n
}

func TestDispatchSortOnSkipsDeletedRecords(t *testing.T) {
	tempDir := t.TempDir()
	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x2A}, append([]byte("Bob       "), []byte(" 30")...)...)
	rec3 := append([]byte{0x20}, append([]byte("Charlie   "), []byte(" 45")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec1, rec2, rec3})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdin = strings.NewReader("Y\n")

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{
		Verb:     "SORT",
		Args:     "ON NAME",
		ToClause: "active",
	}); err != nil {
		t.Fatalf("SORT: %v", err)
	}

	f, err := os.Open(filepath.Join(tempDir, "active.dbf"))
	if err != nil {
		t.Fatalf("open sorted: %v", err)
	}
	defer f.Close()

	tbl, err := dbf.Open(f)
	if err != nil {
		t.Fatalf("open sorted table: %v", err)
	}
	if tbl.Header.RecordCount != 2 {
		t.Fatalf("record count = %d, want 2", tbl.Header.RecordCount)
	}
}
