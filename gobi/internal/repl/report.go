package repl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/frm"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
)

func init() {
	help.Register("REPORT", "REPORT FORM name — print a text .frm report.")
	help.Register("EJECT", "EJECT — write a form feed to the print sink.")
}

func cmdREPORT(ctx *context.Context, cmd command.Command) error {
	args := strings.Fields(cmd.Args)
	if len(args) < 2 || !strings.EqualFold(args[0], "FORM") {
		return fail(ctx, "REPORT FORM required")
	}
	name := args[1]
	opts := reportOpts{}
	for i := 2; i < len(args); i++ {
		u := strings.ToUpper(args[i])
		switch u {
		case "PLAIN":
			opts.plain = true
		case "SUMMARY":
			opts.summary = true
		case "NOEJECT":
			opts.noeject = true
		case "HEADING":
			i++
			if i < len(args) {
				opts.heading = args[i]
			}
		}
	}
	to := strings.TrimSpace(cmd.Clauses["TO"])
	upTo := strings.ToUpper(to)
	opts.toPrint = upTo == "PRINT"
	if strings.HasPrefix(upTo, "FILE") {
		rest := strings.TrimSpace(to[4:])
		opts.toFile = strings.TrimSpace(rest)
	}

	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	path := name
	if filepath.Ext(path) == "" {
		path += ".frm"
	}
	resolved, err := ctx.Resolve(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	f, err := os.Open(resolved)
	if err != nil {
		return fail(ctx, "form not found: %s", path)
	}
	defer f.Close()
	form, err := frm.Parse(f)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}

	var file *os.File
	var out io.Writer = ctx.Out()
	if opts.toFile != "" {
		fp := opts.toFile
		if filepath.Ext(fp) == "" {
			fp += ".txt"
		}
		fp, err = ctx.Resolve(fp)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if fileExists(fp) && ctx.Config.Safety {
			ok, err := confirmSafety(ctx, "file exists, overwrite? (Y/N)")
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
		}
		file, err = os.Create(fp)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		defer file.Close()
		out = file
	}

	w := out
	if (opts.toPrint || ctx.Config.Print) && !opts.noeject && !opts.plain {
		fmt.Fprint(printSink(ctx), "\x0c")
	}

	forExpr, err := parseOptionalLogic(cmd.Clauses["FOR"])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	whileExpr, err := parseOptionalLogic(cmd.Clauses["WHILE"])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	scope := parseScope(cmd)
	env := newEnv(ctx)

	if !opts.plain {
		heading := opts.heading
		if heading == "" {
			heading = time.Now().Format("01/02/06") + "   Page 1"
		}
		writePrintLine(ctx, w, heading)
	}
	writePrintLine(ctx, w, form.Title)
	var hdr strings.Builder
	for i, c := range form.Columns {
		if i > 0 {
			hdr.WriteByte(' ')
		}
		hdr.WriteString(padRight(strings.ToUpper(c.Field), c.Width))
	}
	writePrintLine(ctx, w, hdr.String())

	count := a.RecCount()
	start, end := 1, count
	if scope.Kind != "" {
		start, end = scopeRange(count, a.Cursor, scope)
	}

	numeric := make([]bool, len(form.Columns))
	for i, c := range form.Columns {
		for _, fld := range a.Table.Header.Fields {
			if strings.EqualFold(fld.Name, c.Field) && fld.Type == 'N' {
				numeric[i] = true
			}
		}
	}
	groupTotals := make([]float64, len(form.Columns))
	reportTotals := make([]float64, len(form.Columns))
	var prevGroup string
	haveGroup := false

	for n := start; n <= end; n++ {
		if err := ctx.Goto(n); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if ctx.Config.Deleted && a.Record.Deleted {
			continue
		}
		if whileExpr != nil {
			ok, err := evalLogic(whileExpr, env)
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			if !ok {
				break
			}
		}
		if forExpr != nil {
			ok, err := evalLogic(forExpr, env)
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			if !ok {
				continue
			}
		}

		if form.Group != "" {
			gv, _ := ctx.FieldValue("", form.Group)
			gs := gv.String()
			if haveGroup && gs != prevGroup {
				writeTotals(ctx, w, form, groupTotals, numeric, "Group")
				for i := range groupTotals {
					groupTotals[i] = 0
				}
			}
			prevGroup = gs
			haveGroup = true
		}

		var row strings.Builder
		for i, c := range form.Columns {
			if i > 0 {
				row.WriteByte(' ')
			}
			v, _ := ctx.FieldValue("", c.Field)
			if numeric[i] && v.Kind == expr.KindNumber {
				groupTotals[i] += v.Num
				reportTotals[i] += v.Num
			}
			row.WriteString(padRight(v.String(), c.Width))
		}
		if !opts.summary {
			writePrintLine(ctx, w, row.String())
		}
	}
	if form.Group != "" && haveGroup {
		writeTotals(ctx, w, form, groupTotals, numeric, "Group")
	}
	writeTotals(ctx, w, form, reportTotals, numeric, "Total")
	return nil
}

type reportOpts struct {
	plain   bool
	summary bool
	noeject bool
	heading string
	toPrint bool
	toFile  string
}

func writeTotals(ctx *context.Context, w io.Writer, form frm.Form, totals []float64, numeric []bool, label string) {
	var row strings.Builder
	for i, c := range form.Columns {
		if i > 0 {
			row.WriteByte(' ')
		}
		if i == 0 {
			row.WriteString(padRight(label, c.Width))
			continue
		}
		if numeric[i] {
			row.WriteString(padRight(fmt.Sprintf("%g", totals[i]), c.Width))
		} else {
			row.WriteString(padRight("", c.Width))
		}
	}
	writePrintLine(ctx, w, row.String())
}

func writePrintLine(ctx *context.Context, w io.Writer, s string) {
	margin := ""
	if ctx.Config.Print && ctx.Config.Margin > 0 {
		margin = strings.Repeat(" ", ctx.Config.Margin)
	}
	fmt.Fprintln(w, margin+s)
}

func printSink(ctx *context.Context) io.Writer {
	return ctx.Out()
}

func cmdEJECT(ctx *context.Context, _ command.Command) error {
	fmt.Fprint(printSink(ctx), "\x0c")
	return nil
}
