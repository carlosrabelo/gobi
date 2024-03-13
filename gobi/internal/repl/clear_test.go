package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestDispatchClearClosesDatabasesAndReleasesVariables(t *testing.T) {
	tempDir := t.TempDir()
	createTempDBFWithRecords(t, tempDir, "people.dbf", nil)

	var stdout bytes.Buffer
	ctx := testCtx()
	ctx.Stdout = &stdout
	ctx.Config.DefaultDir = tempDir

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{Verb: "STORE", Args: "42", ToClause: "X"}); err != nil {
		t.Fatalf("STORE: %v", err)
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "CLEAR"}); err != nil {
		t.Fatalf("CLEAR: %v", err)
	}

	area := ctx.GetActiveArea()
	if area.Table != nil {
		t.Fatal("expected CLEAR to close the open database")
	}
	if _, found := ctx.Variables.Get("X"); found {
		t.Fatal("expected CLEAR to release memory variables")
	}
	if strings.Contains(stdout.String(), "\033[2J") {
		t.Fatal("expected CLEAR not to clear the terminal (dBase II semantics)")
	}
}

func TestDispatchClearRejectsUnknownOption(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "CLEAR", Args: "XYZ"})
	if err == nil || !strings.Contains(err.Error(), "Unrecognized CLEAR option") {
		t.Fatalf("expected unrecognized option error, got %v", err)
	}
}

func TestDispatchClearGetsReleasesPendingGets(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	ctx.Screen.WriteAt(2, 4, "KEEP!")
	ctx.Screen.RegisterGet(1, 1, "NAME", "")

	if err := commandMux.Dispatch(ctx, Command{Verb: "CLEAR", Args: "GETS"}); err != nil {
		t.Fatalf("CLEAR GETS: %v", err)
	}

	if len(ctx.Screen.GetFields()) != 0 {
		t.Fatal("expected CLEAR GETS to release GET fields")
	}
	if ctx.Screen.At(2, 4) != 'K' {
		t.Fatal("expected CLEAR GETS to keep screen contents")
	}
}
