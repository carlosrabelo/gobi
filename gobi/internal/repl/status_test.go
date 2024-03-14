package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestDispatchDisplayStatusWithoutDatabases(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "DISPLAY", Args: "STATUS"}); err != nil {
		t.Fatalf("DISPLAY STATUS: %v", err)
	}

	out := ctx.Stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"CURRENTLY SELECTED DATABASE: PRIMARY",
		"PRIMARY DATABASE IN USE: NONE",
		"SECONDARY DATABASE IN USE: NONE",
		"SET TALK       ON",
		"SET INTENSITY  ON",
		"SET BELL       ON",
		"SET EXACT      OFF",
		"SET DELETED    OFF",
		"SET DEFAULT TO",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q, got %q", want, out)
		}
	}
}

func TestDispatchDisplayStatusShowsOpenDatabaseAndIndex(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec})

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Config.Talk = false
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: "people"}); err != nil {
		t.Fatalf("USE: %v", err)
	}
	if err := commandMux.Dispatch(ctx, Command{
		Verb:     "INDEX",
		Args:     "ON NAME",
		ToClause: "byname",
	}); err != nil {
		t.Fatalf("INDEX: %v", err)
	}

	ctx.Stdout = &bytes.Buffer{}
	if err := commandMux.Dispatch(ctx, Command{Verb: "DISPLAY", Args: "STATUS"}); err != nil {
		t.Fatalf("DISPLAY STATUS: %v", err)
	}

	out := ctx.Stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"PRIMARY DATABASE IN USE:",
		"people.dbf",
		"ALIAS: PEOPLE",
		"INDEX FILE:",
		"byname.ndx",
		"KEY: NAME",
		"SET TALK       OFF",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q, got %q", want, out)
		}
	}
}

func TestDispatchListStatusMatchesDisplay(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "STATUS"}); err != nil {
		t.Fatalf("LIST STATUS: %v", err)
	}

	out := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "CURRENTLY SELECTED DATABASE: PRIMARY") {
		t.Fatalf("expected status output, got %q", out)
	}
}
