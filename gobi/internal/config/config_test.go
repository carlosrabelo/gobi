package config

import (
	"path/filepath"
	"testing"
)

func TestNewDefaults(t *testing.T) {
	clearGOBIEnv(t)
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
	if cfg.NoAltScreen {
		t.Error("NoAltScreen: got true, want false")
	}
}

func TestApplyHostEnv(t *testing.T) {
	clearGOBIEnv(t)
	dir := t.TempDir()
	t.Setenv("GOBI_DEFAULT", dir)
	t.Setenv("GOBI_PATH", dir+string(filepath.ListSeparator)+"extra")
	t.Setenv("GOBI_SCREEN", "DEFAULT")
	t.Setenv("GOBI_STATUS", "OFF")
	t.Setenv("GOBI_SCOREBOARD", "0")
	t.Setenv("GOBI_NO_ALTSCREEN", "ON")
	t.Setenv("GOBI_COLOR", "ON")

	cfg := New()
	if cfg.Default != filepath.Clean(dir) {
		t.Fatalf("Default: got %q want %q", cfg.Default, filepath.Clean(dir))
	}
	if cfg.Path != dir+string(filepath.ListSeparator)+"extra" {
		t.Fatalf("Path: got %q", cfg.Path)
	}
	if cfg.Screen != ScreenDefault {
		t.Fatalf("Screen: got %q", cfg.Screen)
	}
	if cfg.Status {
		t.Fatal("Status: want false")
	}
	if cfg.Scoreboard {
		t.Fatal("Scoreboard: want false")
	}
	if !cfg.NoAltScreen {
		t.Fatal("NoAltScreen: want true")
	}
	if !cfg.ColorOn {
		t.Fatal("ColorOn: want true")
	}
}

func clearGOBIEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"GOBI_DEFAULT", "GOBI_PATH", "GOBI_SCREEN", "GOBI_STATUS",
		"GOBI_SCOREBOARD", "GOBI_NO_ALTSCREEN", "GOBI_COLOR",
	} {
		t.Setenv(k, "")
	}
}
