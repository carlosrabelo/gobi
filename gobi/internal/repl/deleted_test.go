package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// deletedTestCtx opens a table with Alice, Bob, Carol and marks Bob deleted.
func deletedTestCtx(t *testing.T) *context.Context {
	t.Helper()
	tempDir := t.TempDir()
	alice := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	bob := append([]byte{0x2A}, append([]byte("Bob       "), []byte(" 35")...)...)
	carol := append([]byte{0x20}, append([]byte("Carol     "), []byte(" 45")...)...)
	createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{alice, bob, carol})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	return ctx
}

func TestDispatchSetDeletedToggles(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if ctx.Config.Deleted {
		t.Fatal("expected DELETED to default to OFF")
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "DELETED ON"}); err != nil {
		t.Fatalf("SET DELETED ON: %v", err)
	}
	if !ctx.Config.Deleted {
		t.Fatal("expected DELETED ON")
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "DELETED OFF"}); err != nil {
		t.Fatalf("SET DELETED OFF: %v", err)
	}
	if ctx.Config.Deleted {
		t.Fatal("expected DELETED OFF")
	}
}

func TestDeletedOnHidesRecordsFromList(t *testing.T) {
	ctx := deletedTestCtx(t)

	out := &bytes.Buffer{}
	ctx.Stdout = out
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST"}); err != nil {
		t.Fatalf("LIST: %v", err)
	}
	if !strings.Contains(out.String(), "Bob") {
		t.Fatalf("expected Bob with DELETED OFF, got %q", out.String())
	}

	ctx.Config.Deleted = true
	out = &bytes.Buffer{}
	ctx.Stdout = out
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST"}); err != nil {
		t.Fatalf("LIST: %v", err)
	}
	if strings.Contains(out.String(), "Bob") {
		t.Fatalf("expected Bob hidden with DELETED ON, got %q", out.String())
	}
	if !strings.Contains(out.String(), "Alice") || !strings.Contains(out.String(), "Carol") {
		t.Fatalf("expected visible records listed, got %q", out.String())
	}
}

func TestDeletedOnExcludesRecordsFromCount(t *testing.T) {
	ctx := deletedTestCtx(t)
	ctx.Config.Talk = true

	out := &bytes.Buffer{}
	ctx.Stdout = out
	if err := commandMux.Dispatch(ctx, Command{Verb: "COUNT", Args: "ALL"}); err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	if !strings.Contains(out.String(), "COUNT = 00003") {
		t.Fatalf("expected 3 with DELETED OFF, got %q", out.String())
	}

	ctx.Config.Deleted = true
	out = &bytes.Buffer{}
	ctx.Stdout = out
	if err := commandMux.Dispatch(ctx, Command{Verb: "COUNT", Args: "ALL"}); err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	if !strings.Contains(out.String(), "COUNT = 00002") {
		t.Fatalf("expected 2 with DELETED ON, got %q", out.String())
	}
}

func TestDeletedOnSkipJumpsOverDeleted(t *testing.T) {
	ctx := deletedTestCtx(t)
	ctx.Config.Deleted = true

	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TOP"}); err != nil {
		t.Fatalf("GO TOP: %v", err)
	}
	area := ctx.GetActiveArea()
	if area.RecordNo != 0 {
		t.Fatalf("GO TOP record = %d, want 0 (Alice)", area.RecordNo)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SKIP"}); err != nil {
		t.Fatalf("SKIP: %v", err)
	}
	if area.RecordNo != 2 {
		t.Fatalf("SKIP record = %d, want 2 (Carol, skipping deleted Bob)", area.RecordNo)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SKIP", Args: "-1"}); err != nil {
		t.Fatalf("SKIP -1: %v", err)
	}
	if area.RecordNo != 0 {
		t.Fatalf("SKIP -1 record = %d, want 0 (Alice)", area.RecordNo)
	}
}

func TestDeletedOnGoTopBottomLandOnVisible(t *testing.T) {
	tempDir := t.TempDir()
	first := append([]byte{0x2A}, append([]byte("First     "), []byte(" 10")...)...)
	mid := append([]byte{0x20}, append([]byte("Mid       "), []byte(" 20")...)...)
	last := append([]byte{0x2A}, append([]byte("Last      "), []byte(" 30")...)...)
	createTempDBFWithRecords(t, tempDir, "edge.dbf", [][]byte{first, mid, last})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Config.Deleted = true
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "edge"}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	area := ctx.GetActiveArea()
	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TOP"}); err != nil {
		t.Fatalf("GO TOP: %v", err)
	}
	if area.RecordNo != 1 {
		t.Fatalf("GO TOP record = %d, want 1 (Mid)", area.RecordNo)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "BOTTOM"}); err != nil {
		t.Fatalf("GO BOTTOM: %v", err)
	}
	if area.RecordNo != 1 {
		t.Fatalf("GO BOTTOM record = %d, want 1 (Mid)", area.RecordNo)
	}
}

func TestDeletedOnLocateSkipsDeleted(t *testing.T) {
	ctx := deletedTestCtx(t)
	ctx.Config.Deleted = true

	if err := commandMux.Dispatch(ctx, Command{Verb: "LOCATE", ForClause: "AGE > 30"}); err != nil {
		t.Fatalf("LOCATE: %v", err)
	}

	area := ctx.GetActiveArea()
	if !area.Found {
		t.Fatal("expected LOCATE to find a record")
	}
	if area.RecordNo != 2 {
		t.Fatalf("LOCATE record = %d, want 2 (Carol, skipping deleted Bob)", area.RecordNo)
	}
}
