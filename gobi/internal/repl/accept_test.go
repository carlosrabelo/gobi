package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestDispatchAcceptStoresStringVariable(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out
	ctx.Stdin = strings.NewReader("John Doe\n")

	cmd := Command{Verb: "ACCEPT", Args: "'What is your name?'", ToClause: "name"}
	if err := commandMux.Dispatch(ctx, cmd); err != nil {
		t.Fatalf("ACCEPT failed: %v", err)
	}

	if !strings.Contains(out.String(), "What is your name?: ") {
		t.Fatalf("expected prompt in output, got %q", out.String())
	}

	val, ok := ctx.Variables.Get("NAME")
	if !ok {
		t.Fatal("expected variable NAME to be stored")
	}
	if s, isString := val.(string); !isString || s != "John Doe" {
		t.Fatalf("expected NAME = %q as string, got %v (%T)", "John Doe", val, val)
	}
}

func TestDispatchAcceptWithoutPromptShowsMarker(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out
	ctx.Stdin = strings.NewReader("answer\n")

	if err := commandMux.Dispatch(ctx, Command{Verb: "ACCEPT", ToClause: "REPLY"}); err != nil {
		t.Fatalf("ACCEPT failed: %v", err)
	}

	if !strings.Contains(out.String(), ": ") {
		t.Fatalf("expected bare input marker, got %q", out.String())
	}
	if val, _ := ctx.Variables.Get("REPLY"); val != "answer" {
		t.Fatalf("expected REPLY = %q, got %v", "answer", val)
	}
}

func TestDispatchAcceptStoresEmptyLine(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("\n")

	if err := commandMux.Dispatch(ctx, Command{Verb: "ACCEPT", ToClause: "BLANK"}); err != nil {
		t.Fatalf("ACCEPT failed: %v", err)
	}

	val, ok := ctx.Variables.Get("BLANK")
	if !ok {
		t.Fatal("expected variable BLANK to be stored")
	}
	if val != "" {
		t.Fatalf("expected empty string, got %v", val)
	}
}

func TestDispatchAcceptRequiresToClause(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("ignored\n")

	err := commandMux.Dispatch(ctx, Command{Verb: "ACCEPT", Args: "'Name?'"})
	if err == nil || !strings.Contains(err.Error(), "ACCEPT requires TO") {
		t.Fatalf("expected missing TO error, got %v", err)
	}
}

func TestDispatchAcceptRejectsUnquotedPrompt(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("ignored\n")

	err := commandMux.Dispatch(ctx, Command{Verb: "ACCEPT", Args: "Name?", ToClause: "X"})
	if err == nil || !strings.Contains(err.Error(), "prompt must be a quoted string") {
		t.Fatalf("expected unquoted prompt error, got %v", err)
	}
}

func TestDispatchAcceptRejectsInvalidVariableName(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("ignored\n")

	err := commandMux.Dispatch(ctx, Command{Verb: "ACCEPT", ToClause: "9BAD"})
	if err == nil || !strings.Contains(err.Error(), "Invalid ACCEPT target") {
		t.Fatalf("expected invalid target error, got %v", err)
	}
}

func TestDispatchAcceptEOFLeavesVariableUnset(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("")

	if err := commandMux.Dispatch(ctx, Command{Verb: "ACCEPT", ToClause: "NOPE"}); err != nil {
		t.Fatalf("expected silent EOF, got %v", err)
	}
	if _, ok := ctx.Variables.Get("NOPE"); ok {
		t.Fatal("expected variable to remain unset on EOF")
	}
}

func TestParseAcceptCommandLine(t *testing.T) {
	cmd := ParseCommand("ACCEPT 'What is your name?' TO NAME")
	if cmd.Verb != "ACCEPT" {
		t.Fatalf("expected verb ACCEPT, got %q", cmd.Verb)
	}
	if cmd.Args != "'What is your name?'" {
		t.Fatalf("expected quoted prompt in args, got %q", cmd.Args)
	}
	if cmd.ToClause != "NAME" {
		t.Fatalf("expected TO clause NAME, got %q", cmd.ToClause)
	}
}
