package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestDispatchHelpOverviewListsCategoriesAndSyntaxes(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out

	if err := commandMux.Dispatch(ctx, Command{Verb: "HELP"}); err != nil {
		t.Fatalf("HELP failed: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"Gobi - dBase II Clone",
		"DATABASE OPERATIONS",
		"BUILT-IN FUNCTIONS",
		"USE [<filename>]",
		"EOF()",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("HELP overview missing %q in:\n%s", want, output)
		}
	}
}

func TestDispatchHelpTopicShowsSyntaxAndDescription(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out

	if err := commandMux.Dispatch(ctx, Command{Verb: "HELP", Args: "LIST"}); err != nil {
		t.Fatalf("HELP LIST failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "LIST [ALL / NEXT <n>] [<fields>] [FOR <expr>] [WHILE <expr>]") {
		t.Fatalf("HELP LIST missing syntax in:\n%s", output)
	}
	if !strings.Contains(output, "Print matching records.") {
		t.Fatalf("HELP LIST missing description in:\n%s", output)
	}
	if !strings.Contains(output, "LIST STRUCTURE") {
		t.Fatalf("HELP LIST missing LIST STRUCTURE variant in:\n%s", output)
	}
}

func TestDispatchHelpTopicIsCaseInsensitive(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out

	if err := commandMux.Dispatch(ctx, Command{Verb: "HELP", Args: "pack"}); err != nil {
		t.Fatalf("HELP pack failed: %v", err)
	}
	if !strings.Contains(out.String(), "PACK") {
		t.Fatalf("HELP pack output missing PACK:\n%s", out.String())
	}
}

func TestDispatchHelpResolvesVerbAbbreviation(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out

	if err := commandMux.Dispatch(ctx, Command{Verb: "HELP", Args: "DELE"}); err != nil {
		t.Fatalf("HELP DELE failed: %v", err)
	}
	if !strings.Contains(out.String(), "DELETE [ALL / NEXT <n>] [FOR <expr>]") {
		t.Fatalf("HELP DELE did not expand to DELETE:\n%s", out.String())
	}
}

func TestDispatchHelpFunctionTopic(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out

	if err := commandMux.Dispatch(ctx, Command{Verb: "HELP", Args: "SUBSTR"}); err != nil {
		t.Fatalf("HELP SUBSTR failed: %v", err)
	}
	if !strings.Contains(out.String(), "SUBSTR(<str>, <start>, <len>)") {
		t.Fatalf("HELP SUBSTR missing function syntax:\n%s", out.String())
	}
}

func TestDispatchHelpUnknownTopicFails(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out

	err := commandMux.Dispatch(ctx, Command{Verb: "HELP", Args: "FROBNICATE"})
	if err == nil {
		t.Fatal("HELP FROBNICATE should fail")
	}
	if !strings.Contains(err.Error(), "No help available for FROBNICATE") {
		t.Fatalf("unexpected error: %v", err)
	}
}
