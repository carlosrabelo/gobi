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
	for _, v := range []string{
		"LOAD", "CALL", "EXPORT", "IMPORT", "LOGOUT", "UNLOCK",
	} {
		m.Register(v, refuse(v))
	}
	for _, topic := range refusedHelpTopics() {
		help.Register(topic, topic+" — not available in Gobi.")
	}
}

func refusedHelpTopics() []string {
	return []string{
		"ASSIST",
		"CREATE LABEL", "MODIFY LABEL",
		"CREATE REPORT", "MODIFY REPORT",
		"CREATE QUERY", "MODIFY QUERY",
		"CREATE SCREEN", "MODIFY SCREEN",
		"CREATE VIEW", "MODIFY VIEW",
		"CREATE VIEW FROM ENVIRONMENT",
		"SET CATALOG", "SET CATALOG TO", "SET TITLE", "SET VIEW TO",
		"SET FILTER TO FILE",
		"LOAD", "CALL", "RELEASE MODULE", "EXPORT", "IMPORT",
		"DISPLAY USERS", "LIST USERS", "LOGOUT", "UNLOCK",
		"SET EXCLUSIVE", "SET ENCRYPTION",
		"FLOCK", "RLOCK", "LOCK",
		"SET PRINTER TO", "SET PRINTER",
		"SET COLOR ON", "SET COLOR OFF",
	}
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
