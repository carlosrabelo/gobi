package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
)

func refuse(verb string) Handler {
	msg := fmt.Sprintf("%s is not available in Gobi", verb)
	help.Register(verb, verb+" — not available in Gobi.")
	return func(ctx *context.Context, cmd command.Command) error {
		return failCode(ctx, 36, "%s", msg)
	}
}

func registerRefused(m *Mux) {
	verbs := []string{
		"ASSIST", "CREATE LABEL", "MODIFY LABEL", "CREATE REPORT", "MODIFY REPORT",
		"CREATE QUERY", "MODIFY QUERY", "CREATE SCREEN", "MODIFY SCREEN",
		"CREATE VIEW", "MODIFY VIEW", "LOAD", "CALL", "EXPORT", "IMPORT",
		"LOGOUT", "UNLOCK",
	}
	for _, v := range verbs {
		// multi-word: register first token only when single; multi handled specially
		parts := strings.Fields(v)
		if len(parts) == 1 {
			m.Register(parts[0], refuse(parts[0]))
		}
	}
	// multi-word via CREATE/MODIFY already registered — wrap
	m.Register("ASSIST", refuse("ASSIST"))
	m.Register("LOAD", refuse("LOAD"))
	m.Register("CALL", refuse("CALL"))
	m.Register("EXPORT", refuse("EXPORT"))
	m.Register("IMPORT", refuse("IMPORT"))
	m.Register("LOGOUT", refuse("LOGOUT"))
	m.Register("UNLOCK", refuse("UNLOCK"))
	help.Register("ASSIST", "ASSIST — not available in Gobi.")
	help.Register("LOAD", "LOAD — not available in Gobi.")
	help.Register("CALL", "CALL — not available in Gobi.")
	help.Register("EXPORT", "EXPORT — not available in Gobi.")
	help.Register("IMPORT", "IMPORT — not available in Gobi.")
	help.Register("RELEASE MODULE", "RELEASE MODULE — not available in Gobi.")
	help.Register("FLOCK", "FLOCK() — not available in Gobi.")
	help.Register("RLOCK", "RLOCK() — not available in Gobi.")
	help.Register("LOCK", "LOCK() — not available in Gobi.")
}

func refuseCreateModify(ctx *context.Context, cmd command.Command, kind string) error {
	args := strings.ToUpper(strings.TrimSpace(cmd.Args))
	for _, w := range []string{"LABEL", "REPORT", "QUERY", "SCREEN", "VIEW"} {
		if strings.HasPrefix(args, w) {
			return failCode(ctx, 36, "%s %s is not available in Gobi", kind, w)
		}
	}
	if kind == "CREATE" && strings.HasPrefix(args, "VIEW FROM") {
		return failCode(ctx, 36, "CREATE VIEW FROM ENVIRONMENT is not available in Gobi")
	}
	return nil
}
