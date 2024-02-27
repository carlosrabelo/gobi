// Package help parses the Gobi language specification markdown into
// searchable documentation topics consumed by the HELP command.
package help

import "strings"

// Topic is a single documented command or built-in function entry.
type Topic struct {
	Syntax      string // syntax template, e.g. "LIST [<fields>] [FOR <expr>]"
	Description string // plain-text explanation of the topic
	Category    string // specification heading the topic belongs to
}

// entry pairs a Topic with the uppercased syntax templates used to match
// HELP queries against the topic.
type entry struct {
	topic Topic
	keys  []string
}

// Documentation holds help topics parsed from the language specification.
type Documentation struct {
	entries    []entry
	categories []string
}

// Parse extracts help topics from a markdown specification source.
// "###" headings define topic categories ("##" headings act as the category
// when no "###" heading follows, as with Built-In Functions) and bullets in
// the form "- `SYNTAX`: description" define the topics themselves.
func Parse(src string) *Documentation {
	doc := &Documentation{}
	category := ""
	seen := make(map[string]bool)
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "### "):
			category = strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
		case strings.HasPrefix(trimmed, "## "):
			category = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
		case strings.HasPrefix(trimmed, "- `"):
			e, ok := parseBullet(trimmed, category)
			if !ok {
				continue
			}
			doc.entries = append(doc.entries, e)
			if !seen[category] {
				seen[category] = true
				doc.categories = append(doc.categories, category)
			}
		}
	}
	return doc
}

// Lookup returns every topic whose syntax templates start with the given
// query (case-insensitive). Multi-word queries such as "DISPLAY MEMORY"
// match their exact template; single verbs match every template variant.
func (d *Documentation) Lookup(query string) []Topic {
	query = strings.ToUpper(strings.TrimSpace(query))
	if query == "" {
		return nil
	}
	var topics []Topic
	for _, e := range d.entries {
		if e.matches(query) {
			topics = append(topics, e.topic)
		}
	}
	return topics
}

// Categories returns the category names in specification order.
func (d *Documentation) Categories() []string {
	return append([]string(nil), d.categories...)
}

// ByCategory returns the topics belonging to a category, in source order.
func (d *Documentation) ByCategory(category string) []Topic {
	var topics []Topic
	for _, e := range d.entries {
		if e.topic.Category == category {
			topics = append(topics, e.topic)
		}
	}
	return topics
}

// parseBullet converts a "- `SYNTAX`: description" markdown bullet into an
// entry. The syntax part ends at the first backtick immediately followed by
// a colon, so descriptions may freely contain backticked words.
func parseBullet(line, category string) (entry, bool) {
	body := strings.TrimPrefix(line, "- ")
	idx := strings.Index(body, "`:")
	if idx < 0 {
		return entry{}, false
	}
	rawSyntax := body[:idx+1]
	description := strings.TrimSpace(body[idx+2:])
	topic := Topic{
		Syntax:      strings.ReplaceAll(rawSyntax, "`", ""),
		Description: strings.ReplaceAll(description, "`", ""),
		Category:    category,
	}
	return entry{topic: topic, keys: matchKeys(rawSyntax)}, true
}

// matchKeys returns the uppercased backtick-delimited syntax templates of a
// bullet, stripped of optional brackets, used to match HELP queries. A
// bullet like "`DISPLAY STRUCTURE` / `LIST STRUCTURE`" yields two keys so
// both verbs find the topic.
func matchKeys(rawSyntax string) []string {
	parts := strings.Split(rawSyntax, "`")
	var keys []string
	for i := 1; i < len(parts); i += 2 {
		key := strings.ToUpper(strings.TrimSpace(parts[i]))
		key = strings.ReplaceAll(key, "[", "")
		key = strings.ReplaceAll(key, "]", "")
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

// matches reports whether any syntax template starts with the query as a
// whole word (or function name, for templates like "EOF()").
func (e entry) matches(query string) bool {
	for _, key := range e.keys {
		if key == query ||
			strings.HasPrefix(key, query+" ") ||
			strings.HasPrefix(key, query+"(") {
			return true
		}
	}
	return false
}
