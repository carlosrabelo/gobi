package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatMemvarValue(t *testing.T) {
	if got := formatMemvarValue(float64(10)); got != "10" {
		t.Fatalf("expected 10, got %q", got)
	}
	if got := formatMemvarValue(float64(17.35)); got != "17.35" {
		t.Fatalf("expected 17.35, got %q", got)
	}
	if got := formatMemvarValue("hello"); got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
	if got := formatMemvarValue(true); got != ".T." {
		t.Fatalf("expected .T., got %q", got)
	}
}

func TestMemvarTypeChar(t *testing.T) {
	if memvarTypeChar(float64(1)) != 'N' {
		t.Fatal("expected numeric type")
	}
	if memvarTypeChar("x") != 'C' {
		t.Fatal("expected character type")
	}
	if memvarTypeChar(false) != 'L' {
		t.Fatal("expected logical type")
	}
}

func TestMemvarByteSize(t *testing.T) {
	if memvarByteSize("abcd") != 4 {
		t.Fatalf("expected 4 bytes for string")
	}
	if memvarByteSize(float64(1)) != 8 {
		t.Fatalf("expected 8 bytes for numeric")
	}
}

func TestWriteMemoryTable(t *testing.T) {
	ctx := testCtx()
	if err := ctx.Variables.Set("MESSAGE", "How's it going so far?"); err != nil {
		t.Fatalf("set MESSAGE: %v", err)
	}
	if err := ctx.Variables.Set("HOURS", float64(10)); err != nil {
		t.Fatalf("set HOURS: %v", err)
	}
	if err := ctx.Variables.Set("PAYRATE", 17.35); err != nil {
		t.Fatalf("set PAYRATE: %v", err)
	}

	var buf bytes.Buffer
	if err := writeMemoryTable(ctx, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "MESSAGE     (C)  How's it going so far?") &&
		!strings.Contains(output, "MESSAGE       (C)  How's it going so far?") {
		t.Fatalf("expected formatted MESSAGE row, got %q", output)
	}
	if !strings.Contains(output, "(N)  10") {
		t.Fatalf("expected formatted HOURS row, got %q", output)
	}
	if !strings.Contains(output, "(N)  17.35") {
		t.Fatalf("expected formatted PAYRATE row, got %q", output)
	}
	if !strings.Contains(output, "** TOTAL ** 03 VARIABLES USED") {
		t.Fatalf("expected total footer, got %q", output)
	}
}

func TestWriteMemoryTableEmpty(t *testing.T) {
	ctx := testCtx()
	var buf bytes.Buffer

	if err := writeMemoryTable(ctx, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if output != "** TOTAL ** 00 VARIABLES USED 00000 BYTES USED\r\n" {
		t.Fatalf("unexpected empty output: %q", output)
	}
}
