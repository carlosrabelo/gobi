package config

import "testing"

func TestNewConfig(t *testing.T) {
	cfg := New()
	if !cfg.Talk {
		t.Error("expected default Talk to be true")
	}
	if !cfg.Intensity {
		t.Error("expected default Intensity to be true")
	}
	if !cfg.Bell {
		t.Error("expected default Bell to be true")
	}
	if cfg.DefaultDir != "." {
		t.Errorf("expected default DefaultDir to be '.', got %s", cfg.DefaultDir)
	}
}
