package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
	"github.com/carlosrabelo/gobi/gobi/pkg/lbl"
)

func init() {
	help.Register("LABEL", "LABEL FORM name — print mailing labels from a text .lbl.")
}

func cmdLABEL(ctx *context.Context, cmd command.Command) error {
	args := strings.Fields(cmd.Args)
	if len(args) < 2 || !strings.EqualFold(args[0], "FORM") {
		return fail(ctx, "LABEL FORM required")
	}
	name := args[1]
	sample := false
	for _, a := range args[2:] {
		if strings.EqualFold(a, "SAMPLE") {
			sample = true
		}
	}
	path := name
	if filepath.Ext(path) == "" {
		path += ".lbl"
	}
	resolved, err := ctx.Resolve(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	f, err := os.Open(resolved)
	if err != nil {
		return fail(ctx, "form not found")
	}
	defer f.Close()
	form, err := lbl.Parse(f)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}

	var out strings.Builder
	if sample {
		for row := 0; row < form.Height; row++ {
			fmt.Fprintln(&out, strings.Repeat("X", form.Width))
		}
		fmt.Fprint(ctx.Stdout, out.String())
		return nil
	}
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	env := newEnv(ctx)
	count := a.RecCount()
	var batch [][]string
	for n := 1; n <= count; n++ {
		if err := ctx.Goto(n); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if ctx.Config.Deleted && a.Record.Deleted {
			continue
		}
		lines := make([]string, form.Height)
		for i, src := range form.Body {
			if strings.TrimSpace(src) == "" {
				lines[i] = padRight("", form.Width)
				continue
			}
			tree, errs := expr.Parse(src)
			if len(errs) > 0 || tree == nil {
				return fail(ctx, "invalid label expression")
			}
			v, err := expr.Eval(tree, env)
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			lines[i] = padRight(v.String(), form.Width)
		}
		batch = append(batch, lines)
		if len(batch) == form.Across {
			writeLabelRow(&out, form, batch)
			batch = nil
		}
	}
	if len(batch) > 0 {
		writeLabelRow(&out, form, batch)
	}
	text := out.String()
	fmt.Fprint(ctx.Stdout, text)
	to := strings.TrimSpace(cmd.Clauses["TO"])
	up := strings.ToUpper(to)
	if strings.HasPrefix(up, "FILE") {
		fp := strings.TrimSpace(to[4:])
		if filepath.Ext(fp) == "" {
			fp += ".txt"
		}
		fp, err = ctx.Resolve(fp)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if err := os.WriteFile(fp, []byte(text), 0o644); err != nil {
			return fail(ctx, "%s", err.Error())
		}
	}
	if up == "PRINT" {
		fmt.Fprint(printSink(ctx), text)
	}
	return nil
}

func writeLabelRow(out *strings.Builder, form lbl.Form, batch [][]string) {
	for row := 0; row < form.Height; row++ {
		var line strings.Builder
		for i, lab := range batch {
			if i > 0 {
				line.WriteString(strings.Repeat(" ", form.Gap))
			}
			line.WriteString(lab[row])
		}
		fmt.Fprintln(out, line.String())
	}
}
