package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestApplySetBellOff(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	applySetBell(ctx, []string{"BELL", "OFF"})
	if ctx.Config.Bell {
		t.Fatal("expected Bell to be false")
	}
	if !strings.Contains(ctx.Stdout.(*bytes.Buffer).String(), "Bell: OFF") {
		t.Fatal("expected SET confirmation")
	}
}

func TestApplySetBellOn(t *testing.T) {
	ctx := testCtx()
	ctx.Config.Bell = false
	ctx.Stdout = &bytes.Buffer{}

	applySetBell(ctx, []string{"BELL", "ON"})
	if !ctx.Config.Bell {
		t.Fatal("expected Bell to be true")
	}
}

func TestBellAlertRespectsFlag(t *testing.T) {
	ctx := testCtx()
	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	ctx.Config.Bell = false

	bellAlert(ctx)
	if stdout.Len() != 0 {
		t.Fatalf("expected no bell with BELL OFF, got %q", stdout.String())
	}

	ctx.Config.Bell = true
	bellAlert(ctx)
	if stdout.String() != "\a" {
		t.Fatalf("expected bell character, got %q", stdout.String())
	}
}

func TestReportValidationErrorRingsBellAndPrints(t *testing.T) {
	ctx := testCtx()
	var stdout, stderr bytes.Buffer
	ctx.Stdout = &stdout
	ctx.Stderr = &stderr
	ctx.Config.Bell = true

	reportValidationError(ctx, errTestValidation)

	if stdout.String() != "\a" {
		t.Fatalf("expected bell, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "invalid value") {
		t.Fatalf("expected stderr message, got %q", stderr.String())
	}
}

func TestReportValidationErrorSkipsBellWhenOff(t *testing.T) {
	ctx := testCtx()
	var stdout, stderr bytes.Buffer
	ctx.Stdout = &stdout
	ctx.Stderr = &stderr
	ctx.Config.Bell = false

	reportValidationError(ctx, errTestValidation)

	if stdout.Len() != 0 {
		t.Fatalf("expected no bell, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "invalid value") {
		t.Fatalf("expected stderr message, got %q", stderr.String())
	}
}

var errTestValidation = testValidationError("invalid value")

type testValidationError string

func (e testValidationError) Error() string { return string(e) }
