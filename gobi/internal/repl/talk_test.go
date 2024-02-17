package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestApplySetTalkOff(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	applySetTalk(ctx, []string{"TALK", "OFF"})
	if ctx.Config.Talk {
		t.Fatal("expected Talk to be false")
	}
	if !strings.Contains(ctx.Stdout.(*bytes.Buffer).String(), "Talk: OFF") {
		t.Fatal("expected SET confirmation")
	}
}

func TestApplySetTalkOn(t *testing.T) {
	ctx := testCtx()
	ctx.Config.Talk = false
	ctx.Stdout = &bytes.Buffer{}

	applySetTalk(ctx, []string{"TALK", "ON"})
	if !ctx.Config.Talk {
		t.Fatal("expected Talk to be true")
	}
}

func TestTalkPrintRespectsFlag(t *testing.T) {
	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	ctx.Config.Talk = false

	talkPrint(ctx, "hidden\r\n")
	if stdout.Len() != 0 {
		t.Fatalf("expected no output with TALK OFF, got %q", stdout.String())
	}

	ctx.Config.Talk = true
	talkPrint(ctx, "visible\r\n")
	if stdout.String() != "visible\r\n" {
		t.Fatalf("expected output with TALK ON, got %q", stdout.String())
	}
}

func TestDispatchCountSuppressesTalkOutput(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "TALK OFF"}); err != nil {
		t.Fatalf("SET TALK OFF: %v", err)
	}

	stdout.Reset()
	if err := commandMux.Dispatch(ctx, Command{Verb: "COUNT"}); err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	if strings.Contains(stdout.String(), "COUNT =") {
		t.Fatalf("expected no COUNT talk output, got %q", stdout.String())
	}
}

func TestDispatchStoreSuppressesTalkOutput(t *testing.T) {
	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout

	if err := commandMux.Dispatch(ctx, Command{Verb: "SET", Args: "TALK OFF"}); err != nil {
		t.Fatalf("SET TALK OFF: %v", err)
	}

	stdout.Reset()
	if err := commandMux.Dispatch(ctx, Command{
		Verb:     "STORE",
		Args:     "42",
		ToClause: "n",
	}); err != nil {
		t.Fatalf("STORE: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no STORE talk output, got %q", stdout.String())
	}
}
