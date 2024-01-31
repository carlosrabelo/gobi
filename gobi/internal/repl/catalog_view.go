package repl

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
)

func init() {
	help.Register("SET CATALOG", "SET CATALOG ON|OFF | SET CATALOG TO <name> — catalog of databases.")
	help.Register("SET TITLE", "SET TITLE TO <text> — catalog title.")
	help.Register("SET VIEW TO", "SET VIEW TO <name> — restore a saved view.")
	help.Register("SET FILTER TO FILE", "SET FILTER TO FILE <name> — load a query file as the filter.")
}

func setCatalog(ctx *context.Context, args []string, to string) error {
	if len(args) >= 2 && (strings.EqualFold(args[1], "ON") || strings.EqualFold(args[1], "OFF")) {
		on, err := parseOnOff(args[1])
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		ctx.Config.CatalogOn = on
		return nil
	}
	name := to
	if name == "" && len(args) >= 2 && strings.EqualFold(args[1], "TO") {
		name = strings.Join(args[2:], " ")
	}
	if name == "" && len(args) >= 2 {
		// SET CATALOG name
		name = args[1]
	}
	if name == "" {
		ctx.CatalogPath = ""
		return nil
	}
	if filepath.Ext(name) == "" {
		name += ".cat"
	}
	path, err := ctx.Resolve(name)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if !fileExists(path) {
		title := ctx.CatalogTitle
		if title == "" {
			title = "Catalog"
		}
		body := "TITLE " + title + "\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return fail(ctx, "%s", err.Error())
		}
	}
	ctx.CatalogPath = path
	ctx.Config.CatalogOn = true
	return nil
}

func setTitle(ctx *context.Context, to string, args []string) error {
	text := to
	if text == "" && len(args) >= 2 && strings.EqualFold(args[1], "TO") {
		text = strings.Join(args[2:], " ")
	}
	ctx.CatalogTitle = text
	if ctx.CatalogPath != "" {
		return rewriteCatalogTitle(ctx)
	}
	return nil
}

func rewriteCatalogTitle(ctx *context.Context) error {
	path := ctx.CatalogPath
	data, err := os.ReadFile(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, ln := range lines {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(ln)), "TITLE ") {
			lines[i] = "TITLE " + ctx.CatalogTitle
			found = true
			break
		}
	}
	if !found {
		lines = append([]string{"TITLE " + ctx.CatalogTitle}, lines...)
	}
	body := strings.Join(lines, "\n")
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func catalogAddDBF(ctx *context.Context, path, alias string) {
	if !ctx.Config.CatalogOn || ctx.CatalogPath == "" {
		return
	}
	line := "DBF " + path
	if alias != "" {
		line += " " + alias
	}
	f, err := os.OpenFile(ctx.CatalogPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line + "\n")
}

func setViewTO(ctx *context.Context, m *Mux, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fail(ctx, "SET VIEW TO requires a name")
	}
	if filepath.Ext(name) == "" {
		name += ".vue"
	}
	path, err := ctx.Resolve(name)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	f, err := os.Open(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "*") {
			continue
		}
		if err := m.Dispatch(ctx, command.Split(line)); err != nil {
			return err
		}
	}
	return sc.Err()
}

func setFilterToFile(ctx *context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fail(ctx, "SET FILTER TO FILE requires a name")
	}
	if filepath.Ext(name) == "" {
		name += ".qry"
	}
	path, err := ctx.Resolve(name)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	text := strings.TrimSpace(string(data))
	text = strings.ReplaceAll(text, "\r\n", "\n")
	// Use first non-empty, non-comment line as the filter expression.
	exprSrc := ""
	for _, ln := range strings.Split(text, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "*") {
			continue
		}
		exprSrc = ln
		break
	}
	ctx.Active().FilterExpr = exprSrc
	return nil
}
