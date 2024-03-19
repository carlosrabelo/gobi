package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// indexedPeopleCtx opens a table whose physical order is Carol, Alice, Bob
// and binds a NAME index, so the index order is Alice(2), Bob(3), Carol(1).
func indexedPeopleCtx(t *testing.T) *context.Context {
	t.Helper()
	tempDir := t.TempDir()
	rec1 := append([]byte{0x20}, append([]byte("Carol     "), []byte(" 31")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec3 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 44")...)...)
	createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec1, rec2, rec3})

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
	return ctx
}

func TestPositionInSequence(t *testing.T) {
	seq := []int{1, 2, 0}
	if pos := positionInSequence(seq, 2, 3); pos != 1 {
		t.Fatalf("pos of record 2 = %d, want 1", pos)
	}
	if pos := positionInSequence(seq, 3, 3); pos != 3 {
		t.Fatalf("pos at EOF = %d, want 3", pos)
	}
	if pos := positionInSequence(seq, 5, 3); pos != 3 {
		t.Fatalf("pos past EOF = %d, want 3", pos)
	}
}

func TestGoTopBottomFollowIndexOrder(t *testing.T) {
	ctx := indexedPeopleCtx(t)
	area := ctx.GetActiveArea()

	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TOP"}); err != nil {
		t.Fatalf("GO TOP: %v", err)
	}
	if area.RecordNo != 1 {
		t.Fatalf("GO TOP record index = %d, want 1 (Alice)", area.RecordNo)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "BOTTOM"}); err != nil {
		t.Fatalf("GO BOTTOM: %v", err)
	}
	if area.RecordNo != 0 {
		t.Fatalf("GO BOTTOM record index = %d, want 0 (Carol)", area.RecordNo)
	}
}

func TestSkipFollowsIndexOrder(t *testing.T) {
	ctx := indexedPeopleCtx(t)
	area := ctx.GetActiveArea()

	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TOP"}); err != nil {
		t.Fatalf("GO TOP: %v", err)
	}

	// Alice -> Bob -> Carol -> EOF
	if err := commandMux.Dispatch(ctx, Command{Verb: "SKIP"}); err != nil {
		t.Fatalf("SKIP: %v", err)
	}
	if area.RecordNo != 2 {
		t.Fatalf("after first SKIP record index = %d, want 2 (Bob)", area.RecordNo)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SKIP"}); err != nil {
		t.Fatalf("SKIP: %v", err)
	}
	if area.RecordNo != 0 {
		t.Fatalf("after second SKIP record index = %d, want 0 (Carol)", area.RecordNo)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SKIP"}); err != nil {
		t.Fatalf("SKIP: %v", err)
	}
	if area.RecordNo != 3 || area.ActiveRecord != nil {
		t.Fatalf("expected EOF after last SKIP, got recno=%d", area.RecordNo)
	}

	// SKIP -1 from EOF returns to the bottom record in index order (Carol).
	if err := commandMux.Dispatch(ctx, Command{Verb: "SKIP", Args: "-1"}); err != nil {
		t.Fatalf("SKIP -1: %v", err)
	}
	if area.RecordNo != 0 {
		t.Fatalf("after SKIP -1 record index = %d, want 0 (Carol)", area.RecordNo)
	}

	// SKIP -1 before the top record fails.
	if err := commandMux.Dispatch(ctx, Command{Verb: "GO", Args: "TOP"}); err != nil {
		t.Fatalf("GO TOP: %v", err)
	}
	err := commandMux.Dispatch(ctx, Command{Verb: "SKIP", Args: "-1"})
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out of range error, got %v", err)
	}
}

func TestListFollowsIndexOrder(t *testing.T) {
	ctx := indexedPeopleCtx(t)
	out := &bytes.Buffer{}
	ctx.Stdout = out

	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST"}); err != nil {
		t.Fatalf("LIST: %v", err)
	}

	output := out.String()
	alice := strings.Index(output, "Alice")
	bob := strings.Index(output, "Bob")
	carol := strings.Index(output, "Carol")
	if alice < 0 || bob < 0 || carol < 0 {
		t.Fatalf("missing names in output: %q", output)
	}
	if !(alice < bob && bob < carol) {
		t.Fatalf("expected index order Alice, Bob, Carol in output: %q", output)
	}

	// Physical record numbers must still be shown.
	if !strings.Contains(output, "    2  Alice") {
		t.Fatalf("expected record number 2 for Alice: %q", output)
	}

	area := ctx.GetActiveArea()
	if area.RecordNo != 3 || area.ActiveRecord != nil {
		t.Fatalf("expected EOF after LIST, got recno=%d", area.RecordNo)
	}
}

func TestListSequentialWithoutIndex(t *testing.T) {
	ctx := indexedPeopleCtx(t)

	// Drop the index; LIST returns to physical order.
	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "INDEX"}); err != nil {
		t.Fatalf("SET INDEX TO: %v", err)
	}

	out := &bytes.Buffer{}
	ctx.Stdout = out
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST"}); err != nil {
		t.Fatalf("LIST: %v", err)
	}

	output := out.String()
	carol := strings.Index(output, "Carol")
	alice := strings.Index(output, "Alice")
	bob := strings.Index(output, "Bob")
	if !(carol < alice && alice < bob) {
		t.Fatalf("expected physical order Carol, Alice, Bob in output: %q", output)
	}
}
