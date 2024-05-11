package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func testCtx() *context.Context {
	ctx := context.New()
	ctx.Stderr = &bytes.Buffer{}
	return ctx
}

func TestRunClearsScreenOnStart(t *testing.T) {
	var stdin bytes.Buffer
	var stdout bytes.Buffer

	stdin.WriteString("QUIT\n")

	ctx := testCtx()
	ctx.Stdin = &stdin
	ctx.Stdout = &stdout

	err := Run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "\033[2J\033[H") {
		t.Fatalf("expected clear sequence on startup, got %q", output)
	}
	if strings.Contains(output, "\033[24;1H") {
		t.Fatalf("expected no cursor pinning on startup, got %q", output)
	}
}

func TestRunExitsOnQuit(t *testing.T) {
	var stdin bytes.Buffer
	var stdout bytes.Buffer

	stdin.WriteString("QUIT\n")

	ctx := testCtx()
	ctx.Stdin = &stdin
	ctx.Stdout = &stdout

	err := Run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, ". ") {
		t.Fatalf("expected prompt in output, got %q", output)
	}
}

func TestRunExitsOnEOF(t *testing.T) {
	var stdin bytes.Buffer
	var stdout bytes.Buffer

	ctx := testCtx()
	ctx.Stdin = &stdin
	ctx.Stdout = &stdout

	err := Run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunScrollsNaturallyAfterErase(t *testing.T) {
	var stdin bytes.Buffer
	var stdout bytes.Buffer

	stdin.WriteString("ERASE\nQUIT\n")

	ctx := testCtx()
	ctx.Stdin = &stdin
	ctx.Stdout = &stdout

	err := Run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "\033[2J\033[H") {
		t.Fatalf("expected clear sequence after ERASE, got %q", output)
	}
	if strings.Contains(output, "\033[24;1H") {
		t.Fatalf("expected no cursor pinning after ERASE, got %q", output)
	}
}

func TestRunSkipsEmptyLines(t *testing.T) {
	var stdin bytes.Buffer
	var stdout bytes.Buffer

	stdin.WriteString("\n  \nQUIT\n")

	ctx := testCtx()
	ctx.Stdin = &stdin
	ctx.Stdout = &stdout

	err := Run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if strings.Count(output, ". ") > 3 {
		t.Fatalf("expected at most 3 prompts, got %d in %q", strings.Count(output, ". "), output)
	}
}
