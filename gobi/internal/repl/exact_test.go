package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func TestDispatchSetExactToggles(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if ctx.Config.Exact {
		t.Fatal("expected EXACT to default to OFF")
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "EXACT ON"}); err != nil {
		t.Fatalf("SET EXACT ON: %v", err)
	}
	if !ctx.Config.Exact {
		t.Fatal("expected EXACT ON")
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "EXACT OFF"}); err != nil {
		t.Fatalf("SET EXACT OFF: %v", err)
	}
	if ctx.Config.Exact {
		t.Fatal("expected EXACT OFF")
	}
}

func evalQuestion(t *testing.T, ctx *context.Context, exprText string) string {
	t.Helper()
	out := &bytes.Buffer{}
	ctx.Stdout = out
	if err := commandMux.Dispatch(ctx, Command{Verb: "?", Args: exprText}); err != nil {
		t.Fatalf("? %s: %v", exprText, err)
	}
	return strings.TrimSpace(out.String())
}

func TestExactOffUsesPrefixComparison(t *testing.T) {
	ctx := testCtx()

	if got := evalQuestion(t, ctx, "'Smith' = 'Sm'"); got != ".T." {
		t.Fatalf("'Smith' = 'Sm' with EXACT OFF = %s, want .T.", got)
	}
	if got := evalQuestion(t, ctx, "'Sm' = 'Smith'"); got != ".F." {
		t.Fatalf("'Sm' = 'Smith' with EXACT OFF = %s, want .F.", got)
	}
	if got := evalQuestion(t, ctx, "'Smith' <> 'Sm'"); got != ".F." {
		t.Fatalf("'Smith' <> 'Sm' with EXACT OFF = %s, want .F.", got)
	}
}

func TestExactOnRequiresFullMatch(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "EXACT ON"}); err != nil {
		t.Fatalf("SET EXACT ON: %v", err)
	}

	if got := evalQuestion(t, ctx, "'Smith' = 'Sm'"); got != ".F." {
		t.Fatalf("'Smith' = 'Sm' with EXACT ON = %s, want .F.", got)
	}
	if got := evalQuestion(t, ctx, "'Smith' = 'Smith'"); got != ".T." {
		t.Fatalf("'Smith' = 'Smith' with EXACT ON = %s, want .T.", got)
	}
}

func TestExactOffCountForPrefix(t *testing.T) {
	tempDir := t.TempDir()
	alice := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	alina := append([]byte{0x20}, append([]byte("Alina     "), []byte(" 30")...)...)
	bob := append([]byte{0x20}, append([]byte("Bob       "), []byte(" 35")...)...)
	createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{alice, alina, bob})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = true
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}

	out := &bytes.Buffer{}
	ctx.Stdout = out
	if err := commandMux.Dispatch(ctx, Command{Verb: "COUNT", ForClause: "NAME = 'Ali'"}); err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	if !strings.Contains(out.String(), "2") {
		t.Fatalf("expected COUNT 2 with EXACT OFF, got %q", out.String())
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "EXACT ON"}); err != nil {
		t.Fatalf("SET EXACT ON: %v", err)
	}
	out = &bytes.Buffer{}
	ctx.Stdout = out
	if err := commandMux.Dispatch(ctx, Command{Verb: "COUNT", ForClause: "NAME = 'Ali'"}); err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	if !strings.Contains(out.String(), "0") {
		t.Fatalf("expected COUNT 0 with EXACT ON, got %q", out.String())
	}
}
