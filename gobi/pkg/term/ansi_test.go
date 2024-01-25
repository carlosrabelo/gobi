package term

import (
	"bytes"
	"strings"
	"testing"
)

func colorPtr(c Color) *Color {
	return &c
}

func TestMoveToUsesOneBasedANSI(t *testing.T) {
	var buf bytes.Buffer
	if err := MoveTo(&buf, 0, 0); err != nil {
		t.Fatalf("MoveTo: %v", err)
	}
	if got := buf.String(); got != "\033[1;1H" {
		t.Fatalf("expected \\033[1;1H, got %q", got)
	}
}

func TestMoveToPosition(t *testing.T) {
	var buf bytes.Buffer
	if err := MoveTo(&buf, 4, 9); err != nil {
		t.Fatalf("MoveTo: %v", err)
	}
	if got := buf.String(); got != "\033[5;10H" {
		t.Fatalf("expected \\033[5;10H, got %q", got)
	}
}

func TestClearScreen(t *testing.T) {
	var buf bytes.Buffer
	if err := ClearScreen(&buf); err != nil {
		t.Fatalf("ClearScreen: %v", err)
	}
	if got := buf.String(); got != "\033[2J\033[H" {
		t.Fatalf("unexpected clear sequence: %q", got)
	}
}

func TestEraseDisplayAndLine(t *testing.T) {
	var display bytes.Buffer
	if err := EraseDisplay(&display); err != nil {
		t.Fatalf("EraseDisplay: %v", err)
	}
	if display.String() != "\033[2J" {
		t.Fatalf("unexpected erase display: %q", display.String())
	}

	var line bytes.Buffer
	if err := EraseLine(&line); err != nil {
		t.Fatalf("EraseLine: %v", err)
	}
	if line.String() != "\033[K" {
		t.Fatalf("unexpected erase line: %q", line.String())
	}
}

func TestHomeAndCursorVisibility(t *testing.T) {
	var home bytes.Buffer
	if err := Home(&home); err != nil {
		t.Fatalf("Home: %v", err)
	}
	if home.String() != "\033[H" {
		t.Fatalf("unexpected home sequence: %q", home.String())
	}

	var hidden bytes.Buffer
	if err := HideCursor(&hidden); err != nil {
		t.Fatalf("HideCursor: %v", err)
	}
	if hidden.String() != "\033[?25l" {
		t.Fatalf("unexpected hide cursor sequence: %q", hidden.String())
	}

	var shown bytes.Buffer
	if err := ShowCursor(&shown); err != nil {
		t.Fatalf("ShowCursor: %v", err)
	}
	if shown.String() != "\033[?25h" {
		t.Fatalf("unexpected show cursor sequence: %q", shown.String())
	}
}

func TestStyleSequence(t *testing.T) {
	tests := []struct {
		name  string
		style Style
		want  string
	}{
		{
			name:  "empty resets",
			style: Style{},
			want:  "\033[0m",
		},
		{
			name:  "foreground",
			style: Style{Foreground: colorPtr(ColorCyan)},
			want:  "\033[36m",
		},
		{
			name:  "background",
			style: Style{Background: colorPtr(ColorBlue)},
			want:  "\033[44m",
		},
		{
			name: "bold reverse colors",
			style: Style{
				Foreground: colorPtr(ColorWhite),
				Background: colorPtr(ColorRed),
				Bold:       true,
				Reverse:    true,
			},
			want: "\033[1;7;37;41m",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.style.Sequence(); got != tc.want {
				t.Fatalf("Sequence() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSetStyleAndColorHelpers(t *testing.T) {
	var buf bytes.Buffer
	if err := SetStyle(&buf, Style{Foreground: colorPtr(ColorGreen)}); err != nil {
		t.Fatalf("SetStyle: %v", err)
	}
	if err := SetForeground(&buf, ColorYellow); err != nil {
		t.Fatalf("SetForeground: %v", err)
	}
	if err := SetBackground(&buf, ColorMagenta); err != nil {
		t.Fatalf("SetBackground: %v", err)
	}
	if err := SetBold(&buf); err != nil {
		t.Fatalf("SetBold: %v", err)
	}
	if err := SetReverse(&buf); err != nil {
		t.Fatalf("SetReverse: %v", err)
	}
	if err := RingBell(&buf); err != nil {
		t.Fatalf("RingBell: %v", err)
	}
	if err := Reset(&buf); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	got := buf.String()
	for _, seq := range []string{"\033[32m", "\033[33m", "\033[45m", "\033[1m", "\033[7m", "\a", "\033[0m"} {
		if !strings.Contains(got, seq) {
			t.Fatalf("expected %q in output %q", seq, got)
		}
	}
}

func TestSaveAndRestoreCursor(t *testing.T) {
	var buf bytes.Buffer
	if err := SaveCursor(&buf); err != nil {
		t.Fatalf("SaveCursor: %v", err)
	}
	if err := RestoreCursor(&buf); err != nil {
		t.Fatalf("RestoreCursor: %v", err)
	}
	if got := buf.String(); got != "\0337\0338" {
		t.Fatalf("expected DECSC/DECRC sequences, got %q", got)
	}
}

func TestJoinInts(t *testing.T) {
	if got := joinInts(nil); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
	if got := joinInts([]int{1, 7, 37}); got != "1;7;37" {
		t.Fatalf("unexpected joined ints: %q", got)
	}
}
