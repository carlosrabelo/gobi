package repl

import (
	"strings"
	"testing"
)

func TestChrReturnsASCIICharacter(t *testing.T) {
	ctx := testCtx()

	if got := evalQuestion(t, ctx, "CHR(65)"); got != "A" {
		t.Fatalf("CHR(65) = %q, want A", got)
	}
	if got := evalQuestion(t, ctx, "CHR(42)"); got != "*" {
		t.Fatalf("CHR(42) = %q, want *", got)
	}
	if got := evalQuestion(t, ctx, "LEN(CHR(7))"); got != "1" {
		t.Fatalf("LEN(CHR(7)) = %q, want 1 (single byte)", got)
	}
}

func TestChrRejectsOutOfRange(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "?", Args: "CHR(300)"})
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out of range error, got %v", err)
	}

	err = commandMux.Dispatch(ctx, Command{Verb: "?", Args: "CHR(0 - 1)"})
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out of range error, got %v", err)
	}
}

func TestChrRejectsNonNumeric(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "?", Args: "CHR('A')"})
	if err == nil || !strings.Contains(err.Error(), "numeric") {
		t.Fatalf("expected numeric argument error, got %v", err)
	}
}
