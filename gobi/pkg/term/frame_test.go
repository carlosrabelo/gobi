package term

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestFrameWriterPresentEmpty(t *testing.T) {
	var out bytes.Buffer
	fw := NewFrameWriter(&out)

	fw.Begin()
	if err := fw.Present(); err != nil {
		t.Fatalf("Present: %v", err)
	}
	if out.String() != "\033[2J\033[H" {
		t.Fatalf("expected clear-only frame, got %q", out.String())
	}

	fw.Begin()
	if err := fw.Present(); err != nil {
		t.Fatalf("second Present: %v", err)
	}
	if out.Len() != len("\033[2J\033[H") {
		t.Fatalf("expected unchanged empty frame to skip output")
	}
}

func TestFrameWriterPresent(t *testing.T) {
	var out bytes.Buffer
	fw := NewFrameWriter(&out)

	fw.Begin()
	fmt.Fprint(fw.Back(), "EDIT screen\r\n")
	if err := fw.Present(); err != nil {
		t.Fatalf("Present: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "\033[2J\033[H") {
		t.Fatalf("expected clear screen sequence, got %q", got)
	}
	if !strings.Contains(got, "EDIT screen") {
		t.Fatalf("expected frame content, got %q", got)
	}
}

func TestFrameWriterSkipsUnchangedFrame(t *testing.T) {
	var out bytes.Buffer
	fw := NewFrameWriter(&out)

	fw.Begin()
	fmt.Fprint(fw.Back(), "same frame")
	if err := fw.Present(); err != nil {
		t.Fatalf("Present: %v", err)
	}
	firstLen := out.Len()

	fw.Begin()
	fmt.Fprint(fw.Back(), "same frame")
	if err := fw.Present(); err != nil {
		t.Fatalf("Present: %v", err)
	}
	if out.Len() != firstLen {
		t.Fatalf("expected unchanged frame to skip output, got %d then %d bytes", firstLen, out.Len())
	}
}

func TestFrameWriterPresentsAfterChange(t *testing.T) {
	var out bytes.Buffer
	fw := NewFrameWriter(&out)

	fw.Begin()
	fmt.Fprint(fw.Back(), "v1")
	if err := fw.Present(); err != nil {
		t.Fatalf("Present: %v", err)
	}
	firstLen := out.Len()

	fw.Begin()
	fmt.Fprint(fw.Back(), "v2")
	if err := fw.Present(); err != nil {
		t.Fatalf("Present: %v", err)
	}
	if out.Len() <= firstLen {
		t.Fatal("expected second present to write updated frame")
	}
	if !strings.Contains(out.String(), "v2") {
		t.Fatal("expected updated frame content")
	}
}

func TestFrameWriterBeginClearsBackBuffer(t *testing.T) {
	fw := NewFrameWriter(&bytes.Buffer{})
	fw.Begin()
	fmt.Fprint(fw.Back(), "draft")
	fw.Begin()
	if got := fw.back.Len(); got != 0 {
		t.Fatalf("expected empty back buffer after Begin, got %d bytes", got)
	}
}

func TestFrameWriterReset(t *testing.T) {
	var out bytes.Buffer
	fw := NewFrameWriter(&out)

	fw.Begin()
	fmt.Fprint(fw.Back(), "frame")
	if err := fw.Present(); err != nil {
		t.Fatalf("Present: %v", err)
	}

	fw.Reset()
	fw.Begin()
	fmt.Fprint(fw.Back(), "frame")
	if err := fw.Present(); err != nil {
		t.Fatalf("Present after reset: %v", err)
	}
	if strings.Count(out.String(), "frame") != 2 {
		t.Fatalf("expected frame to be presented again after reset, got %q", out.String())
	}
}

func TestFrameWriterNilSafe(t *testing.T) {
	var fw *FrameWriter
	fw.Begin()
	if _, err := fw.Back().Write([]byte("x")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := fw.Present(); err != nil {
		t.Fatalf("Present: %v", err)
	}
	fw.Reset()
}
