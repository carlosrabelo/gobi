package repl

import (
	"strings"
	"testing"
)

func TestDispatchUnknownVerb(t *testing.T) {
	ctx := testCtx()

	err := commandMux.Dispatch(ctx, Command{Verb: "XYZZY"})
	if err == nil {
		t.Fatal("expected error for unknown verb")
	}
	if !strings.Contains(err.Error(), "Unrecognized command") {
		t.Fatalf("expected unrecognized command error, got %v", err)
	}
}

func TestDispatchAbbreviation(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "QUI"})
	if err != errQuit {
		t.Fatalf("expected errQuit for QUI abbreviation, got %v", err)
	}
}

func TestDispatchExactVerb(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "QUIT"})
	if err != errQuit {
		t.Fatalf("expected errQuit, got %v", err)
	}
}
