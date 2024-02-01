package term

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestNewScreenDefaults(t *testing.T) {
	s := NewScreen(0, 0)
	if s.Cols() != DefaultCols || s.Rows() != DefaultRows {
		t.Fatalf("expected %dx%d default screen, got %dx%d", DefaultCols, DefaultRows, s.Cols(), s.Rows())
	}
	if s.At(0, 0) != ' ' {
		t.Fatal("expected cleared screen")
	}
}

func TestScreenClearAndWriteAt(t *testing.T) {
	s := NewScreen(10, 3)
	s.WriteAt(1, 2, "HI")
	if s.At(1, 2) != 'H' || s.At(1, 3) != 'I' {
		t.Fatalf("unexpected cells: %q %q", s.At(1, 2), s.At(1, 3))
	}

	s.Clear()
	if s.At(1, 2) != ' ' || s.At(1, 3) != ' ' {
		t.Fatal("expected clear to reset cells")
	}
	if !s.UseCommandLine() {
		t.Fatal("expected clear to enable command-line mode")
	}
	if len(s.GetFields()) != 0 {
		t.Fatal("expected clear to remove GET fields")
	}
}

func TestScreenClearGetsKeepsContents(t *testing.T) {
	s := NewScreen(10, 5)
	s.WriteAt(1, 2, "HI")
	s.RegisterGet(2, 3, "NAME", "XXXX")

	s.ClearGets()

	if len(s.GetFields()) != 0 {
		t.Fatal("expected ClearGets to release GET fields")
	}
	if s.At(1, 2) != 'H' {
		t.Fatal("expected ClearGets to keep screen contents")
	}
}

func TestScreenClearGetsNilSafe(t *testing.T) {
	var s *Screen
	s.ClearGets()
}

func TestScreenRegisterGet(t *testing.T) {
	s := NewScreen(10, 5)
	s.RegisterGet(2, 3, "NAME", "XXXX")
	fields := s.GetFields()
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	if fields[0].Row != 2 || fields[0].Col != 3 || fields[0].Name != "NAME" || fields[0].Picture != "XXXX" {
		t.Fatalf("unexpected field: %+v", fields[0])
	}
}

func TestScreenMoveToCommandLine(t *testing.T) {
	var out bytes.Buffer
	s := NewScreen(80, 24)
	s.Clear()

	if s.CommandLineRow() != 23 {
		t.Fatalf("expected command row 23, got %d", s.CommandLineRow())
	}
	if err := s.MoveToCommandLine(&out); err != nil {
		t.Fatalf("MoveToCommandLine: %v", err)
	}
	if got := out.String(); got != "\033[24;1H" {
		t.Fatalf("expected cursor on last row, got %q", got)
	}
}

func TestScreenWriteAtClipsToWidth(t *testing.T) {
	s := NewScreen(5, 1)
	s.WriteAt(0, 0, "HELLO WORLD")
	if got := string([]byte{s.At(0, 0), s.At(0, 1), s.At(0, 2), s.At(0, 3), s.At(0, 4)}); got != "HELLO" {
		t.Fatalf("expected clipped text HELLO, got %q", got)
	}
}

func TestScreenPresent(t *testing.T) {
	var out bytes.Buffer
	s := NewScreen(4, 2)
	s.WriteAt(0, 0, "AB")
	s.WriteAt(1, 0, "CD")

	if err := s.Present(&out); err != nil {
		t.Fatalf("Present: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "\033[2J\033[H") {
		t.Fatalf("expected clear sequence, got %q", got)
	}
	if !strings.Contains(got, "AB  \r\nCD  \r\n") {
		t.Fatalf("unexpected rendered screen: %q", got)
	}
}

func TestScreenPresentWithHighlights(t *testing.T) {
	var out bytes.Buffer
	s := NewScreen(10, 1)
	s.WriteAt(0, 0, "HELLO     ")

	if err := s.PresentWith(&out, PresentOptions{
		Intensity:  true,
		Highlights: []Highlight{{Row: 0, Col: 0, Length: 5}},
	}); err != nil {
		t.Fatalf("PresentWith: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "\033[7m") {
		t.Fatalf("expected reverse-video sequence, got %q", got)
	}
	if !strings.Contains(got, "HELLO") {
		t.Fatalf("expected cell text, got %q", got)
	}
}

func TestScreenNilSafe(t *testing.T) {
	var s *Screen
	s.Clear()
	s.WriteAt(0, 0, "X")
	s.Resize(100, 30)
	if s.At(0, 0) != ' ' {
		t.Fatal("expected At on nil screen to return space")
	}
	if err := s.Present(io.Discard); err != nil {
		t.Fatalf("Present: %v", err)
	}
}

func TestScreenResizeChangesDimensionsAndClears(t *testing.T) {
	s := NewScreen(80, 24)
	s.WriteAt(0, 0, "DATA")
	s.RegisterGet(1, 1, "FIELD", "")

	s.Resize(120, 40)

	if s.Cols() != 120 || s.Rows() != 40 {
		t.Fatalf("expected 120x40, got %dx%d", s.Cols(), s.Rows())
	}
	if s.At(0, 0) != ' ' {
		t.Fatal("expected buffer cleared after resize")
	}
	if s.GetFields() != nil {
		t.Fatal("expected GET fields cleared after resize")
	}
	if s.CommandLineRow() != 39 {
		t.Fatalf("expected command line row 39, got %d", s.CommandLineRow())
	}
}

func TestScreenResizeSameSizeKeepsContent(t *testing.T) {
	s := NewScreen(80, 24)
	s.WriteAt(0, 0, "KEEP")

	s.Resize(80, 24)

	if s.At(0, 0) != 'K' {
		t.Fatal("expected content preserved when size is unchanged")
	}
}

func TestScreenResizeNonPositiveFallsBackToDefaults(t *testing.T) {
	s := NewScreen(100, 30)
	s.Resize(0, -1)
	if s.Cols() != DefaultCols || s.Rows() != DefaultRows {
		t.Fatalf("expected %dx%d, got %dx%d", DefaultCols, DefaultRows, s.Cols(), s.Rows())
	}
}
