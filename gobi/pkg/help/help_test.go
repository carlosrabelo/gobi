package help

import (
	"reflect"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/docs"
)

const fixture = "# Spec\n" +
	"\n" +
	"Intro paragraph.\n" +
	"\n" +
	"## Command Syntaxes\n" +
	"\n" +
	"### Database Operations\n" +
	"- `USE [<filename>]`: Opens a table.\n" +
	"- `DISPLAY STRUCTURE` / `LIST STRUCTURE`: Prints schema details.\n" +
	"\n" +
	"### Program Flow Control\n" +
	"- `IF <expr>` ... `[ELSE]` ... `ENDIF`: Conditional branching.\n" +
	"- `DO WHILE <expr>` ... `ENDDO`: Loop. Supports `LOOP` and `EXIT`.\n" +
	"\n" +
	"---\n" +
	"\n" +
	"## Built-In Functions\n" +
	"\n" +
	"- `EOF()`: Returns logical `.T.` if cursor is past last record.\n" +
	"- `UPPER(<str>)` / `LOWER(<str>)`: Case conversion.\n"

func TestParseCategoriesInSpecOrder(t *testing.T) {
	doc := Parse(fixture)
	want := []string{"Database Operations", "Program Flow Control", "Built-In Functions"}
	if got := doc.Categories(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Categories() = %v, want %v", got, want)
	}
}

func TestParseTopicFields(t *testing.T) {
	doc := Parse(fixture)
	topics := doc.Lookup("USE")
	if len(topics) != 1 {
		t.Fatalf("Lookup(USE) returned %d topics, want 1", len(topics))
	}
	got := topics[0]
	want := Topic{
		Syntax:      "USE [<filename>]",
		Description: "Opens a table.",
		Category:    "Database Operations",
	}
	if got != want {
		t.Fatalf("Lookup(USE)[0] = %+v, want %+v", got, want)
	}
}

func TestParseStripsBackticksFromDescription(t *testing.T) {
	doc := Parse(fixture)
	topics := doc.Lookup("DO WHILE")
	if len(topics) != 1 {
		t.Fatalf("Lookup(DO WHILE) returned %d topics, want 1", len(topics))
	}
	want := "Loop. Supports LOOP and EXIT."
	if topics[0].Description != want {
		t.Fatalf("Description = %q, want %q", topics[0].Description, want)
	}
}

func TestLookupMatchesAlternateTemplates(t *testing.T) {
	doc := Parse(fixture)
	for _, query := range []string{"LIST", "DISPLAY", "LIST STRUCTURE"} {
		topics := doc.Lookup(query)
		if len(topics) != 1 {
			t.Fatalf("Lookup(%s) returned %d topics, want 1", query, len(topics))
		}
		if topics[0].Syntax != "DISPLAY STRUCTURE / LIST STRUCTURE" {
			t.Fatalf("Lookup(%s) matched %q", query, topics[0].Syntax)
		}
	}
}

func TestLookupMatchesEmbeddedKeywords(t *testing.T) {
	doc := Parse(fixture)
	for _, query := range []string{"ELSE", "ENDIF", "endif"} {
		topics := doc.Lookup(query)
		if len(topics) != 1 {
			t.Fatalf("Lookup(%s) returned %d topics, want 1", query, len(topics))
		}
		if topics[0].Category != "Program Flow Control" {
			t.Fatalf("Lookup(%s) matched category %q", query, topics[0].Category)
		}
	}
}

func TestLookupMatchesFunctionNames(t *testing.T) {
	doc := Parse(fixture)
	for _, query := range []string{"EOF", "UPPER", "lower"} {
		if topics := doc.Lookup(query); len(topics) != 1 {
			t.Fatalf("Lookup(%s) returned %d topics, want 1", query, len(topics))
		}
	}
}

func TestLookupUnknownAndEmptyQueries(t *testing.T) {
	doc := Parse(fixture)
	if topics := doc.Lookup("FOO"); topics != nil {
		t.Fatalf("Lookup(FOO) = %v, want nil", topics)
	}
	if topics := doc.Lookup("  "); topics != nil {
		t.Fatalf("Lookup(blank) = %v, want nil", topics)
	}
}

func TestByCategoryPreservesSourceOrder(t *testing.T) {
	doc := Parse(fixture)
	topics := doc.ByCategory("Built-In Functions")
	if len(topics) != 2 {
		t.Fatalf("ByCategory returned %d topics, want 2", len(topics))
	}
	if topics[0].Syntax != "EOF()" || topics[1].Syntax != "UPPER(<str>) / LOWER(<str>)" {
		t.Fatalf("ByCategory order = %q, %q", topics[0].Syntax, topics[1].Syntax)
	}
}

func TestParseEmbeddedLanguageSpec(t *testing.T) {
	doc := Parse(docs.LanguageSpec)

	categories := doc.Categories()
	if len(categories) == 0 {
		t.Fatal("embedded spec produced no categories")
	}
	wantCategories := map[string]bool{
		"Database Operations": false,
		"Built-In Functions":  false,
	}
	for _, c := range categories {
		if _, ok := wantCategories[c]; ok {
			wantCategories[c] = true
		}
	}
	for c, found := range wantCategories {
		if !found {
			t.Fatalf("embedded spec missing category %q", c)
		}
	}

	for _, query := range []string{"USE", "LIST", "GO TOP", "STORE", "EOF", "SUBSTR", "BROWSE"} {
		if topics := doc.Lookup(query); len(topics) == 0 {
			t.Fatalf("embedded spec has no help for %q", query)
		}
	}
}
