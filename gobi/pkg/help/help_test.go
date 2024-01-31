package help

import (
	"strings"
	"testing"
)

func TestFormatCatalogHasCategories(t *testing.T) {
	s := FormatCatalog(80)
	if !strings.Contains(s, "Gobi help") {
		t.Fatalf("title: %q", s)
	}
	if !strings.Contains(s, "Data") || !strings.Contains(s, "USE") {
		t.Fatalf("catalog missing Data/USE: %q", s)
	}
}

func TestFormatTopicUSE(t *testing.T) {
	topic, ok := Lookup("USE")
	if !ok {
		t.Fatal("USE topic")
	}
	s := FormatTopic(topic, 80)
	if !strings.Contains(s, "Usage:") || !strings.Contains(s, "USE") {
		t.Fatalf("topic: %q", s)
	}
}

func TestApplyAutoPassthroughInHosttermIsSeparate(t *testing.T) {
	// smoke: Register replaces summary
	Register("USE", "USE — open a table.")
	text, ok := Text("USE")
	if !ok || !strings.Contains(text, "open") {
		t.Fatalf("%q ok=%v", text, ok)
	}
}
