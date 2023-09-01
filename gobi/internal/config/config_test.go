package config

import "testing"

func TestNewDefaults(t *testing.T) {
	cfg := New()

	ons := []struct {
		name string
		got  bool
	}{
		{"Talk", cfg.Talk},
		{"Intensity", cfg.Intensity},
		{"Bell", cfg.Bell},
		{"Heading", cfg.Heading},
		{"Safety", cfg.Safety},
		{"Help", cfg.Help},
		{"History", cfg.History},
		{"Scoreboard", cfg.Scoreboard},
		{"Menus", cfg.Menus},
	}
	for _, tt := range ons {
		if !tt.got {
			t.Errorf("%s: got false, want true", tt.name)
		}
	}

	offs := []struct {
		name string
		got  bool
	}{
		{"Exact", cfg.Exact},
		{"Deleted", cfg.Deleted},
		{"DoHistory", cfg.DoHistory},
	}
	for _, tt := range offs {
		if tt.got {
			t.Errorf("%s: got true, want false", tt.name)
		}
	}

	if cfg.Default != "." {
		t.Errorf("Default: got %q, want %q", cfg.Default, ".")
	}
	if cfg.Path != "" {
		t.Errorf("Path: got %q, want empty", cfg.Path)
	}
	if cfg.HistoryTo != 20 {
		t.Errorf("HistoryTo: got %d, want 20", cfg.HistoryTo)
	}
	if cfg.Typeahead != 20 {
		t.Errorf("Typeahead: got %d, want 20", cfg.Typeahead)
	}
	if cfg.MemoWidth != 50 {
		t.Errorf("MemoWidth: got %d, want 50", cfg.MemoWidth)
	}
	if cfg.Screen != ScreenAuto {
		t.Errorf("Screen: got %q, want %q", cfg.Screen, ScreenAuto)
	}
	if cfg.Status != stdoutIsTerminal() {
		t.Errorf("Status: got %v, want %v (stdout terminal)", cfg.Status, stdoutIsTerminal())
	}
}
