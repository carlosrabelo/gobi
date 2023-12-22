package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
)

func (m *Mux) cmdHELP(ctx *context.Context, cmd command.Command) error {
	args := strings.TrimSpace(cmd.Args)
	if args == "" {
		verbs := make([]string, 0, len(m.handlers))
		for v := range m.handlers {
			if v == "*" || v == "??" || v == "?" {
				continue
			}
			verbs = append(verbs, v)
		}
		for i := 0; i < len(verbs); i++ {
			for j := i + 1; j < len(verbs); j++ {
				if verbs[j] < verbs[i] {
					verbs[i], verbs[j] = verbs[j], verbs[i]
				}
			}
		}
		for _, v := range verbs {
			fmt.Fprintln(ctx.Out(), v)
		}
		return nil
	}
	topic := strings.ToUpper(strings.TrimSpace(args))
	text, ok := help.Text(topic)
	if !ok {
		// Fall back to the first word (e.g. HELP USE).
		verb := strings.Fields(topic)[0]
		text, ok = help.Text(verb)
		if !ok {
			return fail(ctx, "no help for %s", topic)
		}
	}
	fmt.Fprintln(ctx.Out(), text)
	return nil
}
