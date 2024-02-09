// Package help stores HELP topics (spec 022).
package help

import (
	"fmt"
	"sort"
	"strings"
)

// Category order for the HELP catalog.
var categoryOrder = []string{
	"Session", "Data", "Records", "Index", "Screen", "Program", "Files", "Set", "Help",
}

// Topic is one HELP entry.
type Topic struct {
	Name     string
	Category string
	Synopsis string
	Summary  string
	Body     string
	Examples []string
	SeeAlso  []string
}

var topics = map[string]*Topic{}

func init() {
	RegisterTopic(Topic{
		Name:     "INTRODUCTION",
		Category: "Help",
		Synopsis: "INTRODUCTION",
		Summary:  "What Gobi is and how to leave the session.",
		Body:     "Gobi is an interactive database session compatible with the dBase III Plus language. At the dot prompt type commands and press Enter. QUIT leaves. HELP lists topics by category; HELP <topic> shows usage and details. VERSION() returns GOBI 1.0.",
		SeeAlso:  []string{"HELP", "QUIT"},
	})
	RegisterTopic(Topic{
		Name:     "EXPRESSIONS",
		Category: "Help",
		Synopsis: "EXPRESSIONS",
		Summary:  "Values and where expressions appear.",
		Body:     "Expressions combine fields, memory variables, literals, operators, and functions. They appear in ?, LIST fields, FOR/WHILE, STORE, REPLACE, and index keys.",
		SeeAlso:  []string{"?", "STORE", "LIST"},
	})
	// Seed common verbs with at least synopsis+summary; Register overwrites.
	for _, t := range []Topic{
		{Name: "USE", Category: "Data", Synopsis: "USE [<file>] [INDEX <ndx…>] [ALIAS <name>]", Summary: "Open a table in the active work area.", Body: "USE with a file name opens that .dbf. USE alone closes the active table. INDEX opens .ndx files. ALIAS sets the area name."},
		{Name: "QUIT", Category: "Session", Synopsis: "QUIT", Summary: "Close tables and leave the session.", Body: "QUIT closes open tables and ends the process with status 0."},
		{Name: "HELP", Category: "Help", Synopsis: "HELP [<topic>]", Summary: "Show the topic catalog or one topic.", Body: "HELP with no arguments prints topics by category. HELP <topic> prints usage, summary, body, examples, and see-also."},
		{Name: "SET", Category: "Set", Synopsis: "SET <option> ON|OFF | SET <option> TO <value>", Summary: "Change a session flag or path.", Body: "Common options include TALK, DEFAULT, EXACT, DELETED, SCREEN AUTO|DEFAULT. Unknown options are errors."},
		{Name: "DIR", Category: "Files", Synopsis: "DIR", Summary: "List .dbf files in the default directory.", Body: "Each line shows the file name, size in bytes, and record count."},
		{Name: "RUN", Category: "Session", Synopsis: "RUN <text>", Summary: "Run a host shell command.", Body: "Passes text to the host shell and waits. Open tables stay open. Non-zero status prints a *** line."},
		{Name: "LIST", Category: "Records", Synopsis: "LIST [<fields>] [FOR <expr>] [WHILE <expr>]", Summary: "List records from the active table.", Body: "Walks records and prints fields. FOR skips non-matching rows; WHILE stops when false."},
		{Name: "DISPLAY", Category: "Records", Synopsis: "DISPLAY [<fields>] [FOR <expr>]", Summary: "Page through records.", Body: "Shows one screen page from the current record."},
		{Name: "APPEND", Category: "Records", Synopsis: "APPEND [BLANK] | APPEND FROM <file>", Summary: "Add records.", Body: "Interactive APPEND edits a form. APPEND BLANK adds an empty record. APPEND FROM copies from another table."},
		{Name: "EDIT", Category: "Records", Synopsis: "EDIT [<n>] [FOR <expr>]", Summary: "Full-screen edit of a record.", Body: "Opens a form for the current or numbered record. Ctrl-W saves; Esc or Ctrl-Q aborts."},
		{Name: "BROWSE", Category: "Records", Synopsis: "BROWSE [FIELDS <list>] …", Summary: "Spreadsheet view of the table.", Body: "Arrow keys move; Enter edits a cell; Esc leaves."},
		{Name: "INDEX", Category: "Index", Synopsis: "INDEX ON <expr> TO <name> [UNIQUE]", Summary: "Build an .ndx index.", Body: "Scans the table and writes keys to name.ndx."},
		{Name: "DO", Category: "Program", Synopsis: "DO <name>", Summary: "Run a .prg command file.", Body: "Loads and runs commands from the file until RETURN, CANCEL, or end."},
		{Name: "STORE", Category: "Program", Synopsis: "STORE <expr> TO <name>", Summary: "Set a memory variable.", Body: "Evaluates expr and stores the value under name (at most 10 characters)."},
		{Name: "ASSIST", Category: "Screen", Synopsis: "ASSIST", Summary: "Full-screen menu of session commands.", Body: "Opens the Assist shell. Esc leaves. Menu actions run ordinary verbs such as USE, LIST, and BROWSE.", SeeAlso: []string{"HELP", "BROWSE", "EDIT"}},
	} {
		RegisterTopic(t)
	}
}

// RegisterTopic adds or replaces a topic. Name is stored upper-case.
func RegisterTopic(t Topic) {
	t.Name = strings.ToUpper(strings.TrimSpace(t.Name))
	if t.Category == "" {
		t.Category = "Help"
	}
	topics[t.Name] = &t
}

// Register adds or replaces a short topic from a single help line (legacy callers).
func Register(verb, text string) {
	verb = strings.ToUpper(strings.TrimSpace(verb))
	summary := text
	if i := strings.Index(text, "—"); i >= 0 {
		summary = strings.TrimSpace(text[i+len("—"):])
	} else if i := strings.Index(text, "-"); i >= 0 && i < 40 {
		summary = strings.TrimSpace(text[i+1:])
	}
	if existing, ok := topics[verb]; ok {
		existing.Summary = summary
		if existing.Body == "" {
			existing.Body = text
		}
		if existing.Synopsis == "" {
			existing.Synopsis = verb
		}
		return
	}
	RegisterTopic(Topic{
		Name:     verb,
		Category: guessCategory(verb),
		Synopsis: verb,
		Summary:  summary,
		Body:     text,
	})
}

func guessCategory(verb string) string {
	switch {
	case strings.HasPrefix(verb, "SET"):
		return "Set"
	case verb == "HELP" || verb == "INTRODUCTION" || verb == "EXPRESSIONS":
		return "Help"
	case verb == "USE" || verb == "CREATE" || verb == "CLOSE" || verb == "SELECT":
		return "Data"
	case verb == "INDEX" || verb == "FIND" || verb == "SEEK" || verb == "REINDEX":
		return "Index"
	case verb == "DO" || verb == "STORE" || verb == "PRIVATE" || verb == "PUBLIC" || verb == "PARAMETERS":
		return "Program"
	case verb == "DIR" || verb == "COPY" || verb == "RENAME" || verb == "ERASE" || verb == "TYPE":
		return "Files"
	case verb == "ASSIST" || verb == "@" || verb == "READ" || verb == "CLEAR":
		return "Screen"
	case verb == "QUIT" || verb == "RUN" || verb == "!" || verb == "CANCEL" || verb == "RESUME" || verb == "SUSPEND":
		return "Session"
	default:
		return "Records"
	}
}

// Lookup returns a topic by name (case-insensitive).
func Lookup(name string) (*Topic, bool) {
	t, ok := topics[strings.ToUpper(strings.TrimSpace(name))]
	return t, ok
}

// Text returns a single-line legacy help string for verb.
func Text(verb string) (string, bool) {
	t, ok := Lookup(verb)
	if !ok {
		return "", false
	}
	if t.Synopsis != "" && t.Summary != "" {
		return t.Synopsis + " — " + t.Summary, true
	}
	return t.Body, t.Body != ""
}

// Verbs returns sorted topic names.
func Verbs() []string {
	out := make([]string, 0, len(topics))
	for k := range topics {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// FormatCatalog writes the categorized HELP index to a string for width columns.
func FormatCatalog(width int) string {
	if width < 40 {
		width = 80
	}
	var b strings.Builder
	b.WriteString("Gobi help — type HELP <topic> for details\n")
	byCat := map[string][]*Topic{}
	for _, t := range topics {
		byCat[t.Category] = append(byCat[t.Category], t)
	}
	for _, cat := range categoryOrder {
		list := byCat[cat]
		if len(list) == 0 {
			continue
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
		b.WriteByte('\n')
		b.WriteString(cat)
		b.WriteByte('\n')
		for _, t := range list {
			line := fmt.Sprintf("  %-12s %s", t.Name, t.Summary)
			if len(line) > width {
				line = line[:width]
			}
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// FormatTopic writes one topic in the 022 layout, wrapped to width.
func FormatTopic(t *Topic, width int) string {
	if width < 40 {
		width = 80
	}
	var b strings.Builder
	b.WriteString(t.Name)
	b.WriteByte('\n')
	if t.Synopsis != "" {
		b.WriteString("Usage: ")
		b.WriteString(t.Synopsis)
		b.WriteByte('\n')
	}
	if t.Summary != "" {
		b.WriteByte('\n')
		b.WriteString(wrap(t.Summary, width))
		b.WriteByte('\n')
	}
	if t.Body != "" {
		b.WriteByte('\n')
		b.WriteString(wrap(t.Body, width))
		b.WriteByte('\n')
	}
	if len(t.Examples) > 0 {
		b.WriteString("\nExamples:\n")
		for _, ex := range t.Examples {
			b.WriteString("  ")
			b.WriteString(ex)
			b.WriteByte('\n')
		}
	}
	if len(t.SeeAlso) > 0 {
		b.WriteString("\nSee also: ")
		b.WriteString(strings.Join(t.SeeAlso, ", "))
		b.WriteByte('\n')
	}
	return b.String()
}

func wrap(s string, width int) string {
	s = strings.TrimSpace(s)
	if len(s) <= width {
		return s
	}
	var b strings.Builder
	for len(s) > width {
		cut := strings.LastIndex(s[:width], " ")
		if cut <= 0 {
			cut = width
		}
		b.WriteString(s[:cut])
		b.WriteByte('\n')
		s = strings.TrimSpace(s[cut:])
	}
	b.WriteString(s)
	return b.String()
}
