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
	args := strings.TrimSpace(cmd.Args)
	up := strings.ToUpper(args)
	if strings.EqualFold(args, "STRUCTURE") {
		return displayStructure(ctx)
	}
	if strings.HasPrefix(up, "FILES") || strings.HasPrefix(up, "STATUS") || strings.EqualFold(args, "MEMORY") || strings.HasPrefix(up, "HISTORY") {
		if strings.EqualFold(args, "MEMORY") {
			return printMemory(ctx)
		}
		if strings.HasPrefix(up, "HISTORY") {
			return fail(ctx, "LIST HISTORY requires session history")
		}
		if strings.HasPrefix(up, "FILES") {
			return cmdDIR(ctx, cmd)
		}
		return displayStatus(ctx)
	}
	return listRecords(ctx, cmd, listOpts{fromOne: true, leaveEOF: true, pageLimit: 0})
}

func cmdDISPLAY(ctx *context.Context, cmd command.Command) error {
	args := strings.TrimSpace(cmd.Args)
	up := strings.ToUpper(args)
	if strings.EqualFold(args, "STRUCTURE") {
		return displayStructure(ctx)
	}
	if strings.EqualFold(args, "MEMORY") {
		return printMemory(ctx)
	}
	if strings.HasPrefix(up, "FILES") {
		return cmdDIR(ctx, cmd)
	}
	if strings.HasPrefix(up, "STATUS") {
		return displayStatus(ctx)
	}
	if _, ok := cmd.Clauses["TO"]; ok {
		return fail(ctx, "DISPLAY does not accept TO")
	}
	return listRecords(ctx, cmd, listOpts{
		fromOne:   false,
		leaveEOF:  false,
		pageLimit: pageSize(ctx),
	})
}

func displayStatus(ctx *context.Context) error {
	fmt.Fprintf(ctx.Stdout, "Area: %d\n", ctx.ActiveArea)
	for i := 1; i <= context.AreaCount; i++ {
		a := ctx.Areas[i]
		if !a.HasTable() {
			continue
		}
		fmt.Fprintf(ctx.Stdout, "  %d: %s", i, a.Path)
		for _, n := range a.NDX {
			if n != "" {
				fmt.Fprintf(ctx.Stdout, " %s", n)
			}
		}
		fmt.Fprintln(ctx.Stdout)
	}
	c := ctx.Config
	fmt.Fprintf(ctx.Stdout, "TALK=%v INTENSITY=%v BELL=%v SAFETY=%v EXACT=%v DELETED=%v\n",
		c.Talk, c.Intensity, c.Bell, c.Safety, c.Exact, c.Deleted)
	return nil
}

type listOpts struct {
	fromOne   bool // LIST starts at record 1
	leaveEOF  bool // LIST leaves cursor past last
	pageLimit int  // 0 = no page cap; DISPLAY uses rows-4
}

func pageSize(ctx *context.Context) int {
	rows := 24
	if ctx.Screen != nil {
		rows = ctx.Screen.Rows
	}
	n := rows - 4
	if n < 1 {
		n = 1
	}
	return n
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
	forExpr, err := parseOptionalLogic(cmd.Clauses["FOR"])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	whileExpr, err := parseOptionalLogic(cmd.Clauses["WHILE"])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	env := newEnv(ctx)
	printHeader(ctx, cols)

	count := a.RecCount()
	seq := make([]int, 0, count)
	if opts.fromOne && len(a.IndexOrder) > 0 && whileExpr == nil {
		seq = append(seq, a.IndexOrder...)
	} else {
		start := 1
		if whileExpr != nil || !opts.fromOne {
			start = a.Cursor
			if start < 1 {
				start = 1
			}
		}
		for n := start; n <= count; n++ {
			seq = append(seq, n)
		}
	}
	if count == 0 || len(seq) == 0 {
		if opts.leaveEOF {
			a.Cursor = count + 1
			a.BOF = false
			_ = a.LoadCurrent()
		}
		return nil
	}

	printed := 0
	lastPrinted := 0
	for _, n := range seq {
		a.Cursor = n
		a.BOF = false
		if err := a.LoadCurrent(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if ctx.Config.Deleted && a.Record.Deleted {
			continue
		}
		if ok, err := recordPassesFilter(ctx, a); err != nil {
			return fail(ctx, "%s", err.Error())
		} else if !ok {
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
		useCols := cols
		if a.FieldsOn && len(a.FieldList) > 0 && len(cols) == 0 {
			useCols = fieldListColumns(a)
		}
		if err := printDataLine(ctx, env, useCols, n); err != nil {
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

func parseOptionalLogic(src string) (expr.Expression, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return nil, nil
	}
	tree, errs := expr.Parse(src)
	if len(errs) > 0 || tree == nil {
		msg := "invalid expression"
		if len(errs) > 0 {
			msg = errs[0]
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return tree, nil
}

func evalLogic(tree expr.Expression, env expr.Env) (bool, error) {
	v, err := expr.Eval(tree, env)
	if err != nil {
		return false, err
	}
	if v.Kind != expr.KindLogical {
		return false, fmt.Errorf("expected logical expression")
	}
	return v.Log, nil
}

type column struct {
	title string
	expr  expr.Expression // nil means use field index
	field int             // 0-based; -1 when expr set
}

func fieldListColumns(a *context.Area) []column {
	var cols []column
	for _, name := range a.FieldList {
		for i, f := range a.Table.Header.Fields {
			if strings.EqualFold(f.Name, name) {
				cols = append(cols, column{title: f.Name, field: i})
				break
			}
		}
	}
	return cols
}

func parseColumns(ctx *context.Context, args string) ([]column, error) {
	args = strings.TrimSpace(args)
	a := ctx.Active()
	if args == "" {
		if a.FieldsOn && len(a.FieldList) > 0 {
			return fieldListColumns(a), nil
		}
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
	if !ctx.Config.Heading {
		return
	}
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
