package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func TestParseGetArgs(t *testing.T) {
	row, col, target, picture, err := parseGetArgs(`4, 12 GET NAME`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row != "4" || col != "12" || target != "NAME" || picture != "" {
		t.Fatalf("unexpected parse: row=%q col=%q target=%q picture=%q", row, col, target, picture)
	}
}

func TestParseGetArgsWithPicture(t *testing.T) {
	_, _, target, picture, err := parseGetArgs(`2, 5 GET PHONE PICTURE "(999) 999-9999"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target != "PHONE" || picture != "(999) 999-9999" {
		t.Fatalf("unexpected parse: target=%q picture=%q", target, picture)
	}
}

func TestParseGetArgsMissingTarget(t *testing.T) {
	_, _, _, _, err := parseGetArgs(`2, 5 GET`)
	if err == nil || !strings.Contains(err.Error(), "variable or field name") {
		t.Fatalf("expected target error, got %v", err)
	}
}

func TestFormatGetDisplayPadsToPictureWidth(t *testing.T) {
	got := formatGetDisplay("AB", "XXXXX")
	if got != "AB   " {
		t.Fatalf("expected padded value, got %q", got)
	}
}

func TestDispatchAtGetRegistersField(t *testing.T) {
	var stdout bytes.Buffer
	ctx := testCtx()
	ctx.Stdout = &stdout

	if err := ctx.Variables.Set("NAME", "Ann"); err != nil {
		t.Fatalf("set variable: %v", err)
	}

	err := commandMux.Dispatch(ctx, Command{
		Verb: "@",
		Args: `6, 8 GET NAME PICTURE "XXXXXXXXXX"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fields := ctx.Screen.GetFields()
	if len(fields) != 1 {
		t.Fatalf("expected 1 GET field, got %d", len(fields))
	}
	if fields[0] != (term.GetField{Row: 6, Col: 8, Name: "NAME", Picture: "XXXXXXXXXX"}) {
		t.Fatalf("unexpected field: %+v", fields[0])
	}

	got := screenTextAt(ctx, 6, 8, 10)
	if got != "Ann       " {
		t.Fatalf("expected padded display %q, got %q", "Ann       ", got)
	}
}

func TestDispatchAtGetShowsBlankForMissingVariable(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	err := commandMux.Dispatch(ctx, Command{
		Verb: "@",
		Args: `1, 1 GET MISSING`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := screenTextAt(ctx, 1, 1, 7); got != "" && got != "       " {
		// empty string or all spaces depending on clip; missing var shows empty
		if strings.TrimSpace(got) != "" {
			t.Fatalf("expected blank display, got %q", got)
		}
	}

	fields := ctx.Screen.GetFields()
	if len(fields) != 1 || fields[0].Name != "MISSING" {
		t.Fatalf("expected MISSING registered, got %+v", fields)
	}
}

func TestClearRemovesRegisteredGetFields(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	if err := commandMux.Dispatch(ctx, Command{Verb: "@", Args: `0, 0 GET X`}); err != nil {
		t.Fatalf("GET: %v", err)
	}
	if len(ctx.Screen.GetFields()) != 1 {
		t.Fatal("expected registered GET field")
	}

	if err := commandMux.Dispatch(ctx, Command{Verb: "CLEAR"}); err != nil {
		t.Fatalf("CLEAR: %v", err)
	}
	if len(ctx.Screen.GetFields()) != 0 {
		t.Fatal("expected CLEAR to remove GET fields")
	}
}
