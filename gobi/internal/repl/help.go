package repl

import (
	"fmt"
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/docs"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
)

// helpDocs holds the embedded language specification parsed once at startup.
var helpDocs = help.Parse(docs.LanguageSpec)

// handleHelp prints the documentation overview, or the syntax and
// description of the topics matching the requested command or function.
func handleHelp(ctx *context.Context, cmd Command) error {
	topic := strings.TrimSpace(cmd.Args)
	if topic == "" {
		writeHelpOverview(ctx.Stdout)
		return nil
	}

	topics := lookupHelpTopics(topic)
	if len(topics) == 0 {
		return fmt.Errorf("*** No help available for %s", strings.ToUpper(topic))
	}
	writeHelpTopics(ctx.Stdout, topics)
	return nil
}

// lookupHelpTopics finds documentation for a topic, expanding dBase II
// verb abbreviations (e.g. DELE -> DELETE) when no direct match exists.
func lookupHelpTopics(topic string) []help.Topic {
	if topics := helpDocs.Lookup(topic); len(topics) > 0 {
		return topics
	}
	if verb, ok := resolveVerb(topic, commandMux.handlers); ok {
		return helpDocs.Lookup(verb)
	}
	return nil
}

func writeHelpOverview(w io.Writer) {
	fmt.Fprint(w, "Gobi - dBase II Clone\r\n")
	fmt.Fprint(w, "Type HELP <command> for details on a specific topic.\r\n")
	for _, category := range helpDocs.Categories() {
		fmt.Fprintf(w, "\r\n%s\r\n", strings.ToUpper(category))
		for _, topic := range helpDocs.ByCategory(category) {
			fmt.Fprintf(w, "  %s\r\n", topic.Syntax)
		}
	}
}

func writeHelpTopics(w io.Writer, topics []help.Topic) {
	for _, topic := range topics {
		fmt.Fprintf(w, "%s\r\n", topic.Syntax)
		fmt.Fprintf(w, "  %s\r\n", topic.Description)
	}
}
