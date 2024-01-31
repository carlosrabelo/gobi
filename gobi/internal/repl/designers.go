package repl

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
)

func init() {
	help.Register("CREATE REPORT", "CREATE REPORT <name> — design a .frm report form.")
	help.Register("MODIFY REPORT", "MODIFY REPORT <name> — edit a .frm report form.")
	help.Register("CREATE LABEL", "CREATE LABEL <name> — design a .lbl label form.")
	help.Register("MODIFY LABEL", "MODIFY LABEL <name> — edit a .lbl label form.")
	help.Register("CREATE QUERY", "CREATE QUERY <name> — design a filter query file.")
	help.Register("MODIFY QUERY", "MODIFY QUERY <name> — edit a query file.")
	help.Register("CREATE SCREEN", "CREATE SCREEN <name> — design a .fmt screen file.")
	help.Register("MODIFY SCREEN", "MODIFY SCREEN <name> — edit a .fmt screen file.")
	help.Register("CREATE VIEW", "CREATE VIEW <name> [FROM ENVIRONMENT] — design or snapshot a view.")
	help.Register("MODIFY VIEW", "MODIFY VIEW <name> — edit a view file.")
}

// designerKind is REPORT, LABEL, QUERY, SCREEN, or VIEW.
func openDesigner(ctx *context.Context, kind, name string, create bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fail(ctx, "%s requires a name", kind)
	}
	ext := designerExt(kind)
	if filepath.Ext(name) == "" {
		name += ext
	}
	path, err := ctx.Resolve(name)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	var initial []string
	if create {
		if fileExists(path) && ctx.Config.Safety {
			if _, err := confirmSafety(ctx, path+" already exists. Overwrite it? (Y/N)"); err != nil {
				return err
			}
		}
		if !fileExists(path) {
			initial = designerTemplate(kind)
		}
	} else if !fileExists(path) {
		return fail(ctx, "file not found: %s", filepath.Base(path))
	}
	return editTextFile(ctx, path, kind, initial)
}

func designerExt(kind string) string {
	switch strings.ToUpper(kind) {
	case "REPORT":
		return ".frm"
	case "LABEL":
		return ".lbl"
	case "QUERY":
		return ".qry"
	case "SCREEN":
		return ".fmt"
	case "VIEW":
		return ".vue"
	default:
		return ".txt"
	}
}

func designerTemplate(kind string) []string {
	switch strings.ToUpper(kind) {
	case "REPORT":
		return []string{"Untitled", ""}
	case "LABEL":
		return []string{"2 20 0 1 0", `""`, `""`}
	case "QUERY":
		return []string{".T."}
	case "SCREEN":
		return []string{"@ 0,0 SAY \"\""}
	case "VIEW":
		return []string{"* view"}
	default:
		return []string{""}
	}
}

func parseDesignerArgs(args string) (kind, name string, fromEnv bool, ok bool) {
	fields := strings.Fields(args)
	if len(fields) == 0 {
		return "", "", false, false
	}
	kind = strings.ToUpper(fields[0])
	switch kind {
	case "LABEL", "REPORT", "QUERY", "SCREEN", "VIEW":
	default:
		return "", "", false, false
	}
	rest := fields[1:]
	if kind == "VIEW" && len(rest) >= 2 &&
		strings.EqualFold(rest[0], "FROM") && strings.EqualFold(rest[1], "ENVIRONMENT") {
		// CREATE VIEW FROM ENVIRONMENT [name]
		name = ""
		if len(rest) >= 3 {
			name = rest[2]
		}
		return kind, name, true, true
	}
	if len(rest) >= 1 {
		name = rest[0]
	}
	if kind == "VIEW" && len(rest) >= 3 &&
		strings.EqualFold(rest[1], "FROM") && strings.EqualFold(rest[2], "ENVIRONMENT") {
		return kind, name, true, true
	}
	return kind, name, false, true
}

func createDesignerCmd(ctx *context.Context, cmd command.Command) (handled bool, err error) {
	kind, name, fromEnv, ok := parseDesignerArgs(cmd.Args)
	fromClause := strings.ToUpper(strings.TrimSpace(cmd.Clauses["FROM"]))
	if strings.HasPrefix(fromClause, "ENVIRONMENT") {
		fromEnv = true
		fields := strings.Fields(cmd.Args)
		if len(fields) >= 1 && strings.EqualFold(fields[0], "VIEW") {
			ok = true
			kind = "VIEW"
			if name == "" && len(fields) >= 2 {
				name = fields[1]
			}
		}
	}
	if !ok {
		return false, nil
	}
	if fromEnv {
		if name == "" {
			name = strings.TrimSpace(cmd.Clauses["TO"])
		}
		if name == "" {
			return true, fail(ctx, "CREATE VIEW FROM ENVIRONMENT requires a name")
		}
		return true, createViewFromEnvironment(ctx, name)
	}
	return true, openDesigner(ctx, kind, name, true)
}

func modifyDesignerCmd(ctx *context.Context, cmd command.Command) (handled bool, err error) {
	kind, name, _, ok := parseDesignerArgs(cmd.Args)
	if !ok {
		return false, nil
	}
	return true, openDesigner(ctx, kind, name, false)
}

// createViewFromEnvironment writes open areas, indexes, relations, and filter.
func createViewFromEnvironment(ctx *context.Context, name string) error {
	if filepath.Ext(name) == "" {
		name += ".vue"
	}
	path, err := ctx.Resolve(name)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if fileExists(path) && ctx.Config.Safety {
		if _, err := confirmSafety(ctx, path+" already exists. Overwrite it? (Y/N)"); err != nil {
			return err
		}
	}
	var b strings.Builder
	for i := 1; i <= context.AreaCount; i++ {
		a := ctx.Areas[i]
		if a == nil || !a.HasTable() {
			continue
		}
		b.WriteString("SELECT ")
		b.WriteString(string(rune('A' + i - 1)))
		b.WriteByte('\n')
		use := "USE " + a.Path
		if a.Alias != "" {
			use += " ALIAS " + a.Alias
		}
		b.WriteString(use)
		b.WriteByte('\n')
		var ndx []string
		for _, p := range a.NDX {
			if p != "" {
				ndx = append(ndx, p)
			}
		}
		if len(ndx) > 0 {
			b.WriteString("SET INDEX TO ")
			b.WriteString(strings.Join(ndx, ","))
			b.WriteByte('\n')
		}
		if a.FilterExpr != "" {
			b.WriteString("SET FILTER TO ")
			b.WriteString(a.FilterExpr)
			b.WriteByte('\n')
		}
		if a.RelExpr != "" && a.RelAlias != "" {
			b.WriteString("SET RELATION TO ")
			b.WriteString(a.RelExpr)
			b.WriteString(" INTO ")
			b.WriteString(a.RelAlias)
			b.WriteByte('\n')
		}
	}
	b.WriteString("SELECT ")
	b.WriteString(string(rune('A' + ctx.ActiveArea - 1)))
	b.WriteByte('\n')
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
