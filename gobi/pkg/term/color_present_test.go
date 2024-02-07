package term

import (
	"bytes"
	"strings"
	"testing"
)

func TestPresentEmitsColor(t *testing.T) {
	s := NewSize(40, 10)
	s.ColorActive = true
	s.StandardFG, s.StandardBG = 7, 0
	s.EnhancedFG, s.EnhancedBG = 0, 7
	s.Put(0, 0, 'A', false)
	var out bytes.Buffer
	if err := s.Present(&out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "\x1b[37m") || !strings.Contains(got, "\x1b[40m") {
		t.Fatalf("expected white on black: %q", got)
	}
	if !strings.Contains(got, "\x1b[0m") {
		t.Fatalf("expected reset: %q", got)
	}
}
