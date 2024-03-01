package repl

import (
	"bytes"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/script"
)

func TestDispatchRemarkEchoesText(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out

	err := commandMux.Dispatch(ctx, Command{Verb: "REMARK", Args: "hello from dBase"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if out.String() != "hello from dBase\r\n" {
		t.Fatalf("REMARK output = %q", out.String())
	}
}

func TestDispatchRemarkEmptyPrintsBlankLine(t *testing.T) {
	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out

	if err := commandMux.Dispatch(ctx, Command{Verb: "REMARK"}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if out.String() != "\r\n" {
		t.Fatalf("REMARK output = %q", out.String())
	}
}

func TestRunProgramEchoesRemark(t *testing.T) {
	source := "REMARK Dept list program\n" +
		"* silent comment\n" +
		"NOTE another silent comment\n" +
		"REMARK done TO everyone\n"

	prog, err := script.ParseSource("remark.prg", source)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctx := testCtx()
	out := &bytes.Buffer{}
	ctx.Stdout = out

	if err := RunProgram(ctx, prog); err != nil {
		t.Fatalf("RunProgram: %v", err)
	}

	want := "Dept list program\r\ndone TO everyone\r\n"
	if out.String() != want {
		t.Fatalf("script output = %q, want %q", out.String(), want)
	}
}
