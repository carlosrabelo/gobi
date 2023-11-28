package repl

import (
	"strings"
	"testing"
)

func TestParseLocateClauses(t *testing.T) {
	forExp, whileExp, err := parseLocateClauses("AGE > 30", "RECNO() <= 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if forExp == nil || whileExp == nil {
		t.Fatal("expected parsed FOR and WHILE expressions")
	}
}

func TestParseLocateClausesInvalidFor(t *testing.T) {
	_, _, err := parseLocateClauses("AGE >", "")
	if err == nil || !strings.Contains(err.Error(), "FOR clause") {
		t.Fatalf("expected FOR syntax error, got %v", err)
	}
}
