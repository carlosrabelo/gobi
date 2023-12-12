package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func TestSETPrintAppliesMargin(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split(`STORE 1 TO A`))
	_ = mux.Dispatch(ctx, command.Split("SET MARGIN TO 2"))
	_ = mux.Dispatch(ctx, command.Split("SET PRINT ON"))
	if !ctx.Config.Print {
		t.Fatal("PRINT still off")
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("? A")); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "  1") {
		t.Fatalf("margin missing: %q", out.String())
	}
}

func TestQuestionAndDouble(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()

	if err := mux.Dispatch(ctx, command.Split("? .T.")); err != nil {
		t.Fatal(err)
	}
	if out.String() != ".T.\n" {
		t.Fatalf("?: %q", out.String())
	}

	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("?? 1+1")); err != nil {
		t.Fatal(err)
	}
	if out.String() != "2" {
		t.Fatalf("??: %q", out.String())
	}

	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("? 1, 2")); err != nil {
		t.Fatal(err)
	}
	if out.String() != "1 2\n" {
		t.Fatalf("? commas: %q", out.String())
	}
}
