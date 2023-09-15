package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func TestRunEmptyLineThenQuit(t *testing.T) {
	ctx := context.New()
	var out, errBuf bytes.Buffer
	ctx.Stdout = &out
	ctx.Stderr = &errBuf
	ctx.Stdin = strings.NewReader("\nQUIT\n")

	if err := Run(ctx, NewMux()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), ". ") {
		t.Fatalf("prompt missing: %q", out.String())
	}
	if errBuf.Len() != 0 {
		t.Fatalf("stderr: %q", errBuf.String())
	}
}

func TestRunUnknownVerb(t *testing.T) {
	ctx := context.New()
	var out, errBuf bytes.Buffer
	ctx.Stdout = &out
	ctx.Stderr = &errBuf
	ctx.Stdin = strings.NewReader("FOOBAR\nQUIT\n")

	if err := Run(ctx, NewMux()); err != nil {
		t.Fatal(err)
	}
	got := errBuf.String()
	if !strings.Contains(got, "*** Unrecognized command: FOOBAR") {
		t.Fatalf("stderr: %q", got)
	}
}

func TestRunQuitClean(t *testing.T) {
	ctx := context.New()
	var out bytes.Buffer
	ctx.Stdout = &out
	ctx.Stderr = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("QUIT\n")

	if err := Run(ctx, NewMux()); err != nil {
		t.Fatal(err)
	}
}
