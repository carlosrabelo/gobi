package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func cmdLIST(ctx *context.Context, cmd command.Command) error {
	if strings.EqualFold(strings.TrimSpace(cmd.Args), "STRUCTURE") {
		return displayStructure(ctx)
	}
	return listRecords(ctx, cmd, listOpts{fromOne: true, leaveEOF: true, pageLimit: 0})
}

func cmdDISPLAY(ctx *context.Context, cmd command.Command) error {
	if strings.EqualFold(strings.TrimSpace(cmd.Args), "STRUCTURE") {
		return displayStructure(ctx)
	}
	// Record DISPLAY arrives in 010.3.
	return fail(ctx, "DISPLAY STRUCTURE required")
}

type listOpts struct {
	fromOne   bool // LIST starts at record 1
	leaveEOF  bool // LIST leaves cursor past last
	pageLimit int  // 0 = no page cap; DISPLAY uses rows-4
}

func pageSize(*context.Context) int {
	// Classic 24-row screen minus 4 until spec 018.
	return 20
}

func listRecords(ctx *context.Context, cmd command.Command, opts listOpts) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	cols, err := parseColumns(ctx, cmd.Args)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	env := newEnv(ctx)
	printHeader(ctx, cols)

	count := a.RecCount()
	start := 1
	if !opts.fromOne {
		start = a.Cursor
		if start < 1 {
			start = 1
		}
		if count == 0 || start > count {
			return nil
		}
	}

	printed := 0
	lastPrinted := 0
	for n := start; n <= count; n++ {
		a.Cursor = n
		a.BOF = false
		if err := a.LoadCurrent(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if ctx.Config.Deleted && a.Record.Deleted {
			continue
		}
		if err := printDataLine(ctx, env, cols, n); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		printed++
		lastPrinted = n
		if opts.pageLimit > 0 && printed >= opts.pageLimit {
			break
		}
	}

	if opts.leaveEOF {
		a.Cursor = count + 1
		a.BOF = false
		_ = a.LoadCurrent()
	} else if lastPrinted > 0 {
		a.Cursor = lastPrinted
		a.BOF = false
		_ = a.LoadCurrent()
	}
	return nil
}

type column struct {
	title string
	expr  expr.Expression // nil means use field index
	field int             // 0-based; -1 when expr set
}

func parseColumns(ctx *context.Context, args string) ([]column, error) {
	args = strings.TrimSpace(args)
	a := ctx.Active()
	if args == "" {
		cols := make([]column, len(a.Table.Header.Fields))
		for i, f := range a.Table.Header.Fields {
			cols[i] = column{title: f.Name, field: i}
		}
		return cols, nil
	}
	parts := splitCommaExprs(args)
	cols := make([]column, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		tree, errs := expr.Parse(p)
		if len(errs) > 0 || tree == nil {
			msg := "invalid expression"
			if len(errs) > 0 {
				msg = errs[0]
			}
			return nil, fmt.Errorf("%s", msg)
		}
		cols = append(cols, column{title: strings.ToUpper(p), expr: tree, field: -1})
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("empty field list")
	}
	return cols, nil
}

func printHeader(ctx *context.Context, cols []column) {
	fmt.Fprint(ctx.Stdout, "Record#")
	for _, c := range cols {
		fmt.Fprintf(ctx.Stdout, " %s", c.title)
	}
	fmt.Fprintln(ctx.Stdout)
}

func printDataLine(ctx *context.Context, env expr.Env, cols []column, recNo int) error {
	fmt.Fprintf(ctx.Stdout, "%7d", recNo)
	a := ctx.Active()
	for _, c := range cols {
		var s string
		if c.expr != nil {
			v, err := expr.Eval(c.expr, env)
			if err != nil {
				return err
			}
			s = v.String()
		} else {
			f := a.Table.Header.Fields[c.field]
			s = fieldString(f, a.Record.Values[c.field])
		}
		fmt.Fprintf(ctx.Stdout, " %s", s)
	}
	fmt.Fprintln(ctx.Stdout)
	return nil
}

func fieldString(f dbf.Field, v dbf.Value) string {
	switch f.Type {
	case 'N':
		return expr.NumberVal(v.Num).String()
	case 'L':
		return expr.LogicalVal(v.Log).String()
	case 'D', 'M':
		if v.Empty {
			return ""
		}
		return v.Str
	default:
		return v.Str
	}
}

func splitCommaExprs(s string) []string {
	var parts []string
	var b strings.Builder
	depth := 0
	inSingle, inDouble := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
			b.WriteByte(c)
		case c == '"' && !inSingle:
			inDouble = !inDouble
			b.WriteByte(c)
		case !inSingle && !inDouble && c == '(':
			depth++
			b.WriteByte(c)
		case !inSingle && !inDouble && c == ')':
			if depth > 0 {
				depth--
			}
			b.WriteByte(c)
		case !inSingle && !inDouble && depth == 0 && c == ',':
			parts = append(parts, strings.TrimSpace(b.String()))
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	if t := strings.TrimSpace(b.String()); t != "" || len(parts) > 0 {
		parts = append(parts, strings.TrimSpace(b.String()))
	}
	return parts
}
