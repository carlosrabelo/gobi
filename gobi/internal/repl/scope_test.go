package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func TestParseScopeClause(t *testing.T) {
	scope, rest, err := parseScopeClause("")
	if err != nil || scope.explicit || rest != "" {
		t.Fatalf("empty: %#v rest=%q err=%v", scope, rest, err)
	}

	scope, rest, err = parseScopeClause("all NAME, AGE")
	if err != nil || !scope.all || rest != "NAME, AGE" {
		t.Fatalf("all: %#v rest=%q err=%v", scope, rest, err)
	}

	scope, rest, err = parseScopeClause("NEXT 3 NAME WITH 'x'")
	if err != nil || scope.next != 3 || rest != "NAME WITH 'x'" {
		t.Fatalf("next: %#v rest=%q err=%v", scope, rest, err)
	}

	if _, _, err = parseScopeClause("NEXT zero"); err == nil {
		t.Fatal("expected error for non-numeric NEXT count")
	}
	if _, _, err = parseScopeClause("NEXT 0"); err == nil {
		t.Fatal("expected error for non-positive NEXT count")
	}

	scope, rest, err = parseScopeClause("NAME, AGE")
	if err != nil || scope.explicit || rest != "NAME, AGE" {
		t.Fatalf("no scope: %#v rest=%q err=%v", scope, rest, err)
	}
}

// scopePeopleCtx opens a table with records Carol/31, Alice/25, Bob/44 and
// positions the cursor at the first record.
func scopePeopleCtx(t *testing.T) *context.Context {
	t.Helper()
	tempDir := t.TempDir()
	rec1 := append([]byte{0x20}, append([]byte("Carol     "), []byte(" 31")...)...)
	rec2 := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	rec3 := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 44")...)...)
	createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec1, rec2, rec3})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	return ctx
}

func TestListNextLimitsScan(t *testing.T) {
	ctx := scopePeopleCtx(t)
	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "2"}); err != nil {
		t.Fatalf("GOTO: %v", err)
	}

	out := &bytes.Buffer{}
	ctx.Stdout = out
	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "NEXT 2 NAME"}); err != nil {
		t.Fatalf("LIST NEXT: %v", err)
	}

	output := out.String()
	if strings.Contains(output, "Carol") {
		t.Fatalf("LIST NEXT 2 from record 2 must not include Carol: %q", output)
	}
	if !strings.Contains(output, "Alice") || !strings.Contains(output, "Bob") {
		t.Fatalf("expected Alice and Bob in output: %q", output)
	}

	// Pointer stays on the last record scanned instead of EOF.
	area := ctx.GetActiveArea()
	if area.RecordNo != 2 {
		t.Fatalf("record index = %d, want 2", area.RecordNo)
	}
}

func TestDisplayAllShowsEveryRecordFromTop(t *testing.T) {
	ctx := scopePeopleCtx(t)
	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "3"}); err != nil {
		t.Fatalf("GOTO: %v", err)
	}

	out := &bytes.Buffer{}
	ctx.Stdout = out
	if err := commandMux.Dispatch(ctx, Command{Verb: "DISPLAY", Args: "ALL"}); err != nil {
		t.Fatalf("DISPLAY ALL: %v", err)
	}

	output := out.String()
	for _, name := range []string{"Carol", "Alice", "Bob"} {
		if !strings.Contains(output, name) {
			t.Fatalf("expected %s in DISPLAY ALL output: %q", name, output)
		}
	}
}

func TestDeleteAllAndNextScopes(t *testing.T) {
	ctx := scopePeopleCtx(t)

	if err := commandMux.Dispatch(ctx, Command{Verb: "DELETE", Args: "ALL"}); err != nil {
		t.Fatalf("DELETE ALL: %v", err)
	}

	out := &bytes.Buffer{}
	ctx.Stdout = out
	ctx.Config.Talk = true
	if err := commandMux.Dispatch(ctx, Command{Verb: "RECALL", Args: "ALL"}); err != nil {
		t.Fatalf("RECALL ALL: %v", err)
	}
	if !strings.Contains(out.String(), "3 record(s) recalled") {
		t.Fatalf("expected 3 records recalled after DELETE ALL: %q", out.String())
	}
	ctx.Config.Talk = false

	// DELETE NEXT 2 from record 1 marks only records 1 and 2.
	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "1"}); err != nil {
		t.Fatalf("GOTO: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "DELETE", Args: "NEXT 2"}); err != nil {
		t.Fatalf("DELETE NEXT: %v", err)
	}
	out.Reset()
	ctx.Config.Talk = true
	if err := commandMux.Dispatch(ctx, Command{Verb: "RECALL", Args: "ALL"}); err != nil {
		t.Fatalf("RECALL ALL: %v", err)
	}
	if !strings.Contains(out.String(), "2 record(s) recalled") {
		t.Fatalf("expected 2 records recalled after DELETE NEXT 2: %q", out.String())
	}
}

func TestReplaceAllAndNextScopes(t *testing.T) {
	ctx := scopePeopleCtx(t)

	if err := commandMux.Dispatch(ctx, Command{Verb: "REPLACE", Args: "ALL AGE WITH 1"}); err != nil {
		t.Fatalf("REPLACE ALL: %v", err)
	}

	// All three ages are 1 now; REPLACE NEXT 2 from record 1 bumps two of them.
	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "1"}); err != nil {
		t.Fatalf("GOTO: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "REPLACE", Args: "NEXT 2 AGE WITH 10"}); err != nil {
		t.Fatalf("REPLACE NEXT: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{
		Verb: "SUM", Args: "AGE", ToClause: "total",
	}); err != nil {
		t.Fatalf("SUM: %v", err)
	}
	total, ok := ctx.Variables.Get("TOTAL")
	if !ok || total.(float64) != 21 {
		t.Fatalf("TOTAL = %#v, want 21 (10+10+1)", total)
	}
}

func TestCountAndSumNextScopes(t *testing.T) {
	ctx := scopePeopleCtx(t)

	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "2"}); err != nil {
		t.Fatalf("GOTO: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{
		Verb: "COUNT", Args: "NEXT 2", ToClause: "hits",
	}); err != nil {
		t.Fatalf("COUNT NEXT: %v", err)
	}
	hits, ok := ctx.Variables.Get("HITS")
	if !ok || hits.(float64) != 2 {
		t.Fatalf("HITS = %#v, want 2", hits)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "2"}); err != nil {
		t.Fatalf("GOTO: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{
		Verb: "SUM", Args: "NEXT 2 AGE", ToClause: "total",
	}); err != nil {
		t.Fatalf("SUM NEXT: %v", err)
	}
	total, ok := ctx.Variables.Get("TOTAL")
	if !ok || total.(float64) != 69 {
		t.Fatalf("TOTAL = %#v, want 69 (25+44)", total)
	}
}

func TestLocateNextLimitsSearch(t *testing.T) {
	ctx := scopePeopleCtx(t)
	area := ctx.GetActiveArea()

	// Bob (record 3) is outside LOCATE NEXT 2 starting at record 1.
	if err := commandMux.Dispatch(ctx, Command{
		Verb: "LOCATE", Args: "NEXT 2", ForClause: "NAME = 'Bob'",
	}); err != nil {
		t.Fatalf("LOCATE NEXT: %v", err)
	}
	if area.Found {
		t.Fatal("Bob must not be found within NEXT 2 scope")
	}

	// Alice (record 2) is inside the scope.
	if err := commandMux.Dispatch(ctx, Command{Verb: "GOTO", Args: "1"}); err != nil {
		t.Fatalf("GOTO: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{
		Verb: "LOCATE", Args: "NEXT 2", ForClause: "NAME = 'Alice'",
	}); err != nil {
		t.Fatalf("LOCATE NEXT: %v", err)
	}
	if !area.Found || area.RecordNo != 1 {
		t.Fatalf("expected Alice found at index 1, got found=%v recno=%d", area.Found, area.RecordNo)
	}

	// CONTINUE respects the original scope bound: no more matches.
	if err := commandMux.Dispatch(ctx, Command{Verb: "CONTINUE"}); err != nil {
		t.Fatalf("CONTINUE: %v", err)
	}
	if area.Found {
		t.Fatal("CONTINUE must not find matches past the NEXT scope")
	}
}
