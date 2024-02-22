package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func TestRunInlineMode(t *testing.T) {
	ctx := context.New()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}

	if err := run(ctx, nil, `STORE 99 TO n`); err != nil {
		t.Fatalf("run inline: %v", err)
	}

	val, ok := ctx.Variables.Get("n")
	if !ok {
		t.Fatal("expected n variable")
	}
	if f, ok := val.(float64); !ok || f != 99 {
		t.Fatalf("n = %v (%T), want 99", val, val)
	}
}

func TestRunScriptFile(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "hello.prg")
	if err := os.WriteFile(scriptPath, []byte("STORE 42 TO answer\n"), 0644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	ctx := context.New()
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}

	if err := run(ctx, []string{scriptPath}, ""); err != nil {
		t.Fatalf("run script: %v", err)
	}

	val, ok := ctx.Variables.Get("answer")
	if !ok {
		t.Fatal("expected answer variable")
	}
	if f, ok := val.(float64); !ok || f != 42 {
		t.Fatalf("answer = %v (%T), want 42", val, val)
	}
}

func TestRunRejectsInlineAndScriptTogether(t *testing.T) {
	ctx := context.New()
	err := run(ctx, []string{"demo.prg"}, "STORE 1 TO x")
	if err == nil {
		t.Fatal("expected error when -e and script are both provided")
	}
}

func TestRunRejectsMultipleScriptFiles(t *testing.T) {
	ctx := context.New()
	err := run(ctx, []string{"one.prg", "two.prg"}, "")
	if err == nil {
		t.Fatal("expected error for multiple script files")
	}
}

func TestRunInteractiveMode(t *testing.T) {
	ctx := context.New()
	ctx.Stdin = strings.NewReader("QUIT\n")
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}

	if err := run(ctx, nil, ""); err != nil {
		t.Fatalf("run interactive: %v", err)
	}
}
