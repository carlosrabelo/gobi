package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func TestValidateReadPictureDigits(t *testing.T) {
	if err := validateReadValue("25", "99"); err != nil {
		t.Fatalf("expected valid digits, got %v", err)
	}
	if err := validateReadValue("2A", "99"); err == nil {
		t.Fatal("expected invalid digit validation")
	}
}

func TestValidateReadPictureLetters(t *testing.T) {
	if err := validateReadValue("AB", "!!"); err != nil {
		t.Fatalf("expected valid letters, got %v", err)
	}
	if err := validateReadValue("A1", "!!"); err == nil {
		t.Fatal("expected invalid letter validation")
	}
}

func TestOverlayPictureValueWithLiterals(t *testing.T) {
	spec := buildPictureSpec("(99)")
	got := overlayPictureValue(spec, "5")
	if got != "(5 )" {
		t.Fatalf("expected overlay with literal, got %q", got)
	}
}

func TestReadSessionTabNavigation(t *testing.T) {
	s := &readSession{
		fields: []readFieldState{
			{def: term.GetField{Row: 0, Col: 0, Name: "A", Picture: "XX"}, value: "AB", spec: buildPictureSpec("XX")},
			{def: term.GetField{Row: 1, Col: 0, Name: "B", Picture: "XX"}, value: "", spec: buildPictureSpec("XX")},
		},
	}

	done, err := s.handleKey(editKeyTab)
	if err != nil || done {
		t.Fatalf("tab advance: done=%v err=%v", done, err)
	}
	if s.fieldIdx != 1 {
		t.Fatalf("expected field 1, got %d", s.fieldIdx)
	}
}

func TestReadSessionValidateBlocksTab(t *testing.T) {
	s := &readSession{
		ctx: testCtx(),
		fields: []readFieldState{
			{def: term.GetField{Row: 0, Col: 0, Name: "AGE", Picture: "99"}, value: "2A", spec: buildPictureSpec("99")},
		},
	}

	_, err := s.handleKey(editKeyTab)
	if err == nil || !strings.Contains(err.Error(), "AGE") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestDispatchReadCommitsMemvars(t *testing.T) {
	var stdout bytes.Buffer
	ctx := testCtx()
	ctx.Stdout = &stdout
	ctx.Stdin = strings.NewReader("Ann\t25\r")

	if err := commandMux.Dispatch(ctx, Command{Verb: "CLEAR"}); err != nil {
		t.Fatalf("CLEAR: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "@", Args: `5, 10 GET NAME PICTURE "XXXXX"`}); err != nil {
		t.Fatalf("GET NAME: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "@", Args: `6, 10 GET AGE PICTURE "99"`}); err != nil {
		t.Fatalf("GET AGE: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "READ"}); err != nil {
		t.Fatalf("READ: %v", err)
	}

	name, ok := ctx.Variables.Get("NAME")
	if !ok || name != "Ann" {
		t.Fatalf("NAME = %v (%v), want Ann", name, ok)
	}
	age, ok := ctx.Variables.Get("AGE")
	if !ok || age != "25" {
		t.Fatalf("AGE = %v (%v), want 25", age, ok)
	}
}

func TestDispatchReadRequiresGetFields(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "READ"})
	if err == nil || !strings.Contains(err.Error(), "GET fields") {
		t.Fatalf("expected GET fields error, got %v", err)
	}
}
