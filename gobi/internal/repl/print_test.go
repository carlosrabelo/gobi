package repl

import (
	"bytes"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

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
