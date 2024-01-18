package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/ndx"
)

func TestParseIndexOnExpression(t *testing.T) {
	expr, err := parseIndexOnExpression("ON NAME")
	if err != nil || expr != "NAME" {
		t.Fatalf("expr = %q err=%v", expr, err)
	}

	_, err = parseIndexOnExpression("NAME")
	if err == nil {
		t.Fatal("expected ON prefix error")
	}
}

func TestDispatchIndexOnBuildsNDXFile(t *testing.T) {
	tempDir := t.TempDir()

	rec1 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 30")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec1, rec2})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdin = strings.NewReader("Y\n")

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{
		Verb:     "INDEX",
		Args:     "ON NAME",
		ToClause: "people",
	})
	if err != nil {
		t.Fatalf("INDEX: %v", err)
	}

	indexPath := filepath.Join(tempDir, "people.ndx")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("expected index file: %v", err)
	}
	if len(ctx.GetActiveArea().Indexes) != 1 {
		t.Fatalf("expected one open index, got %d", len(ctx.GetActiveArea().Indexes))
	}

	idx := ctx.GetActiveArea().Indexes[0]
	result, found, err := idx.Manager().SearchExact(ndx.Key("Bob"))
	if err != nil || !found {
		t.Fatalf("expected Bob in index, found=%v err=%v", found, err)
	}
	if result.RecordNumber != 2 {
		t.Fatalf("record = %d, want 2", result.RecordNumber)
	}
}

func TestDispatchIndexOnRequiresDatabase(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{
		Verb:     "INDEX",
		Args:     "ON NAME",
		ToClause: "people",
	})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchIndexOnRequiresToClause(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

	ctx := testCtx()
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "INDEX", Args: "ON NAME"})
	if err == nil || !strings.Contains(err.Error(), "TO") {
		t.Fatalf("expected TO clause error, got %v", err)
	}
}
