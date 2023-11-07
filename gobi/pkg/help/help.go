package help

import "strings"

var texts = map[string]string{
	"USE":     "USE [file] [ALIAS name] — open a table in the active work area.",
	"QUIT":    "QUIT — close tables and leave the session.",
	"HELP":    "HELP [verb] — list verbs or show one verb's help.",
	"SET":     "SET option ON|OFF or SET DEFAULT TO path.",
	"DIR":     "DIR — list .dbf files in the default directory.",
	"RUN":     "RUN text — run a host shell command.",
	"LIST":    "LIST [fields] [FOR|WHILE] — list records.",
	"DISPLAY": "DISPLAY — page through records.",
	"APPEND":  "APPEND — add records; APPEND FROM file.",
	"EDIT":    "EDIT [n] — full-screen edit.",
	"BROWSE":  "BROWSE — spreadsheet view of the table.",
	"INDEX":   "INDEX ON expr TO name — build an .ndx.",
	"DO":      "DO name — run a .prg command file.",
	"STORE":   "STORE expr TO name — set a memory variable.",
}

// Text returns help for verb (case-insensitive). ok is false if unknown.
func Text(verb string) (string, bool) {
	t, ok := texts[strings.ToUpper(verb)]
	return t, ok
}

// Verbs returns sorted verb names that have help text.
func Verbs() []string {
	out := make([]string, 0, len(texts))
	for k := range texts {
		out = append(out, k)
	}
	// insertion sort for stability without importing sort in tiny package — use sort
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// Register adds or replaces help text for a verb.
func Register(verb, text string) {
	texts[strings.ToUpper(verb)] = text
}
