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

func TestDispatchReindexRebuildsActiveIndex(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdin = strings.NewReader("Bob\n30\n\n")

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

	idx := ctx.GetActiveArea().Indexes[0]
	if _, found, _ := idx.Manager().SearchExact(ndx.Key("Bob")); found {
		t.Fatal("expected Bob to be missing before append/reindex")
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "APPEND"}); err != nil {
		t.Fatalf("APPEND: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "REINDEX"}); err != nil {
		t.Fatalf("REINDEX: %v", err)
	}

	result, found, err := idx.Manager().SearchExact(ndx.Key("Bob"))
	if err != nil || !found {
		t.Fatalf("expected Bob after REINDEX, found=%v err=%v", found, err)
	}
	if result.RecordNumber != 2 {
		t.Fatalf("record = %d, want 2", result.RecordNumber)
	}
}

func TestDispatchReindexRequiresDatabase(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "REINDEX"})
	if err == nil || !strings.Contains(err.Error(), "No database file is in use") {
		t.Fatalf("expected no database error, got %v", err)
	}
}

func TestDispatchReindexRequiresOpenIndexes(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

	ctx := testCtx()
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{Verb: "REINDEX"})
	if err == nil || !strings.Contains(err.Error(), "No index files are in use") {
		t.Fatalf("expected no index error, got %v", err)
	}
}

func TestDispatchAppendSyncsMultipleIndexes(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdin = strings.NewReader("Bob\n30\n\n")

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "INDEX", Args: "ON NAME", ToClause: "byname"}); err != nil {
		t.Fatalf("INDEX NAME: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "INDEX", Args: "ON AGE", ToClause: "byage"}); err != nil {
		t.Fatalf("INDEX AGE: %v", err)
	}
	if len(ctx.GetActiveArea().Indexes) != 2 {
		t.Fatalf("expected 2 indexes, got %d", len(ctx.GetActiveArea().Indexes))
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "APPEND"}); err != nil {
		t.Fatalf("APPEND: %v", err)
	}

	nameIdx := ctx.GetActiveArea().Indexes[0]
	ageIdx := ctx.GetActiveArea().Indexes[1]

	_, found, err := nameIdx.Manager().SearchExact(ndx.Key("Bob"))
	if err != nil || !found {
		t.Fatalf("expected Bob in name index, found=%v err=%v", found, err)
	}

	_, found, err = ageIdx.Manager().SearchExact(ndx.Key("30"))
	if err != nil || !found {
		t.Fatalf("expected 30 in age index, found=%v err=%v", found, err)
	}
}

func TestDispatchReplaceSyncsMultipleIndexes(t *testing.T) {
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
	if err := commandMux.Dispatch(ctx, Command{Verb: "INDEX", Args: "ON NAME", ToClause: "byname"}); err != nil {
		t.Fatalf("INDEX NAME: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "INDEX", Args: "ON AGE", ToClause: "byage"}); err != nil {
		t.Fatalf("INDEX AGE: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TO 2"}); err != nil {
		t.Fatalf("GO TO 2: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "REPLACE", Args: `NAME WITH "Carol"`}); err != nil {
		t.Fatalf("REPLACE NAME: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "REPLACE", Args: `AGE WITH 40`}); err != nil {
		t.Fatalf("REPLACE AGE: %v", err)
	}

	nameIdx := ctx.GetActiveArea().Indexes[0]
	ageIdx := ctx.GetActiveArea().Indexes[1]

	if _, found, _ := nameIdx.Manager().SearchExact(ndx.Key("Bob")); found {
		t.Fatal("expected Bob to be removed from name index")
	}
	result, found, err := nameIdx.Manager().SearchExact(ndx.Key("Carol"))
	if err != nil || !found {
		t.Fatalf("expected Carol in name index, found=%v err=%v", found, err)
	}
	if result.RecordNumber != 2 {
		t.Fatalf("Carol record = %d, want 2", result.RecordNumber)
	}

	if _, found, _ := ageIdx.Manager().SearchExact(ndx.Key("30")); found {
		t.Fatal("expected 30 to be removed from age index")
	}
	_, found, err = ageIdx.Manager().SearchExact(ndx.Key("40"))
	if err != nil || !found {
		t.Fatalf("expected 40 in age index, found=%v err=%v", found, err)
	}
}
