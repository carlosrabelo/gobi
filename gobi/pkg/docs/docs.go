// Package docs embeds the Gobi specification documents so they can be
// shipped inside the binary and consumed at runtime (e.g. by the HELP
// command).
package docs

import _ "embed"

// LanguageSpec holds the dBase II language and syntax specification used
// as the documentation source for the HELP command.
//
//go:embed language_spec.md
var LanguageSpec string
