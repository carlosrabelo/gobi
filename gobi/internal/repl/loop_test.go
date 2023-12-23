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
	if !strings.Contains(got, "*** Unrecognized command: FOOBAR\n") {
		t.Fatalf("stderr: %q", got)
	}
	if strings.Contains(got, "FOOBARQUIT") {
		t.Fatalf("LF did not end the line: %q", got)
	}
}

func TestRunCtrlCCancelsLineNotExit(t *testing.T) {
	ctx := context.New()
	var out, errBuf bytes.Buffer
	ctx.Stdout = &out
	ctx.Stderr = &errBuf
	// Ctrl-C (0x03) clears the partial line; QUIT still ends the session.
	ctx.Stdin = strings.NewReader("HEL\x03QUIT\n")

	if err := Run(ctx, NewMux()); err != nil {
		t.Fatal(err)
	}
	if errBuf.Len() != 0 {
		t.Fatalf("stderr: %q", errBuf.String())
	}
	if strings.Contains(out.String(), "USE") {
		t.Fatalf("HELP should have been cancelled: %q", out.String())
	}
}

func TestRunHELPWithLFEndsLine(t *testing.T) {
	ctx := context.New()
	var out, errBuf bytes.Buffer
	ctx.Stdout = &out
	ctx.Stderr = &errBuf
	// Unix Enter is LF; must dispatch HELP, not concatenate with QUIT.
	ctx.Stdin = strings.NewReader("HELP\nQUIT\n")

	if err := Run(ctx, NewMux()); err != nil {
		t.Fatal(err)
	}
	if errBuf.Len() != 0 {
		t.Fatalf("stderr: %q", errBuf.String())
	}
	got := out.String()
	if !strings.Contains(got, "USE") {
		t.Fatalf("HELP verb list missing USE: %q", got)
	}
	if strings.Contains(errBuf.String(), "HELPQUIT") || strings.Contains(got, "Unrecognized") {
		t.Fatalf("commands concatenated: out=%q err=%q", got, errBuf.String())
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
