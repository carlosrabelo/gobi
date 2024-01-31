package repl

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func (m *Mux) cmdLIST(ctx *context.Context, cmd command.Command) error {
	args := strings.TrimSpace(cmd.Args)
	up := strings.ToUpper(args)
	if strings.EqualFold(args, "STRUCTURE") {
		return displayStructure(ctx)
	}
	if strings.EqualFold(args, "USERS") {
		return nil // local session: no-op (038)
	}
	if strings.HasPrefix(up, "FILES") || strings.HasPrefix(up, "STATUS") || strings.EqualFold(args, "MEMORY") || strings.HasPrefix(up, "HISTORY") {
		if strings.EqualFold(args, "MEMORY") {
			return printMemory(ctx)
		}
		if strings.HasPrefix(up, "HISTORY") {
			return cmdDisplayHistory(ctx, m.hist, cmd)
		}
		if strings.HasPrefix(up, "FILES") {
			return cmdDIR(ctx, cmd)
		}
		return displayStatus(ctx)
	}
	return listRecords(ctx, cmd, listOpts{fromOne: true, leaveEOF: true, pageLimit: 0})
}

func (m *Mux) cmdDISPLAY(ctx *context.Context, cmd command.Command) error {
	args := strings.TrimSpace(cmd.Args)
	up := strings.ToUpper(args)
	if strings.EqualFold(args, "STRUCTURE") {
		return displayStructure(ctx)
	}
	if strings.EqualFold(args, "USERS") {
		return nil // local session: no-op (038)
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
	if strings.HasPrefix(up, "HISTORY") {
		return cmdDisplayHistory(ctx, m.hist, cmd)
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
	fmt.Fprintf(ctx.Out(), "Currently Selected Database:\n")
	fmt.Fprintf(ctx.Out(), "Select area: %2d\n", ctx.ActiveArea)
	for i := 1; i <= context.AreaCount; i++ {
		a := ctx.Areas[i]
		if !a.HasTable() {
			continue
		}
		alias := a.Alias
		if alias == "" {
			alias = "(none)"
		}
		fmt.Fprintf(ctx.Out(), "  Area %2d  %-10s  %s\n", i, alias, a.Path)
		fmt.Fprintf(ctx.Out(), "    Recno  %d / %d", a.Cursor, a.RecCount())
		if a.Record.Deleted {
			fmt.Fprint(ctx.Out(), "  Deleted")
		}
		fmt.Fprintln(ctx.Out())
		ndxPrinted := false
		for _, n := range a.NDX {
			if n == "" {
				continue
			}
			if !ndxPrinted {
				fmt.Fprint(ctx.Out(), "    Index ")
				ndxPrinted = true
			} else {
				fmt.Fprint(ctx.Out(), "          ")
			}
			fmt.Fprintln(ctx.Out(), n)
		}
		if a.FilterExpr != "" {
			fmt.Fprintf(ctx.Out(), "    Filter %s\n", a.FilterExpr)
		}
		if a.RelExpr != "" {
			fmt.Fprintf(ctx.Out(), "    Relation %s\n", a.RelExpr)
		}
	}
	c := ctx.Config
	fmt.Fprintln(ctx.Out(), "SET options:")
	fmt.Fprintf(ctx.Out(), "  TALK=%v  INTENSITY=%v  BELL=%v  SAFETY=%v\n", c.Talk, c.Intensity, c.Bell, c.Safety)
	fmt.Fprintf(ctx.Out(), "  EXACT=%v  DELETED=%v  HEADING=%v  CONFIRM=%v\n", c.Exact, c.Deleted, c.Heading, c.Confirm)
	fmt.Fprintf(ctx.Out(), "  ESCAPE=%v  CONSOLE=%v  PRINT=%v  FIXED=%v\n", c.Escape, c.Console, c.Print, c.Fixed)
	fmt.Fprintf(ctx.Out(), "  DEFAULT=%s\n", c.Default)
	if c.Path != "" {
		fmt.Fprintf(ctx.Out(), "  PATH=%s\n", c.Path)
	}
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
	scope := parseScope(cmd)
	cols, err := parseColumns(ctx, scope.Rest)
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
	printHeader(ctx, cols, scope.Off)

	count := a.RecCount()
	seq := make([]int, 0, count)
	if scope.Kind != "" {
		start, end := scopeRange(count, a.Cursor, scope)
		for n := start; n <= end; n++ {
			seq = append(seq, n)
		}
	} else if opts.fromOne && len(a.IndexOrder) > 0 && whileExpr == nil {
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
		if err := printDataLine(ctx, env, useCols, n, scope.Off); err != nil {
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

func printHeader(ctx *context.Context, cols []column, off bool) {
	if !ctx.Config.Heading {
		return
	}
	a := ctx.Active()
	var b strings.Builder
	if !off {
		b.WriteString("Record#")
	}
	for i, c := range cols {
		w := listColWidth(a, c)
		title := c.title
		if len(title) > w {
			title = title[:w]
		}
		if !off || i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(padRight(title, w))
	}
	writeScreenLine(ctx, b.String(), true)
}

func printDataLine(ctx *context.Context, env expr.Env, cols []column, recNo int, off bool) error {
	var b strings.Builder
	if !off {
		fmt.Fprintf(&b, "%7d", recNo)
	}
	a := ctx.Active()
	var memoExtra []string
	for i, c := range cols {
		w := listColWidth(a, c)
		var s string
		if c.expr != nil {
			v, err := expr.Eval(c.expr, env)
			if err != nil {
				return err
			}
			s = padRight(v.String(), w)
		} else {
			f := a.Table.Header.Fields[c.field]
			v := a.Record.Values[c.field]
			if f.Type == 'M' && !v.Empty {
				lines := wrapMemo(v.Str, ctx.Config.MemoWidth)
				if len(lines) > 0 {
					s = padRight(lines[0], w)
					memoExtra = append(memoExtra, lines[1:]...)
				} else {
					s = padRight("", w)
				}
			} else {
				s = formatListField(ctx, f, v, w)
			}
		}
		if !off || i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(s)
	}
	writeScreenLine(ctx, b.String(), true)
	pad := "        "
	if off {
		pad = ""
	}
	for _, ln := range memoExtra {
		writeScreenLine(ctx, pad+ln, true)
	}
	return nil
}

// listColWidth is the III Plus LIST/DISPLAY column width (DBF field length).
func listColWidth(a *context.Area, c column) int {
	if c.expr != nil || c.field < 0 || a.Table == nil {
		// Expression columns: at least the heading width (values pad to match).
		if n := len(c.title); n > 0 {
			return n
		}
		return 10
	}
	w := int(a.Table.Header.Fields[c.field].Length)
	if w < 1 {
		w = 1
	}
	return w
}

// formatListField renders one DBF field into a fixed-width LIST cell.
func formatListField(ctx *context.Context, f dbf.Field, v dbf.Value, w int) string {
	switch f.Type {
	case 'N':
		return formatListNumber(v.Num, w, int(f.Decimals))
	case 'L':
		if v.Log {
			return padRight("T", w)
		}
		return padRight("F", w)
	case 'D':
		if v.Empty {
			return padRight("", w)
		}
		dv := expr.EmptyDateVal()
		if t, err := parseYYYYMMDD(strings.TrimSpace(v.Str)); err == nil {
			dv = expr.DateVal(t)
		}
		order := expr.DateAmerican
		century := false
		if ctx.Config != nil {
			if ctx.Config.DateOrder != "" {
				order = ctx.Config.DateOrder
			}
			century = ctx.Config.Century
		}
		return padRight(expr.FormatDate(dv, order, century), w)
	case 'M':
		return padRight("", w)
	default: // 'C'
		return padRight(v.Str, w)
	}
}

func formatListNumber(n float64, width, decimals int) string {
	if width < 1 {
		width = 1
	}
	if decimals < 0 {
		decimals = 0
	}
	s := strconv.FormatFloat(n, 'f', decimals, 64)
	if len(s) > width {
		return strings.Repeat("*", width)
	}
	return padLeft(s, width)
}

func padLeft(s string, n int) string {
	if len(s) >= n {
		return s[len(s)-n:]
	}
	return strings.Repeat(" ", n-len(s)) + s
}

func parseYYYYMMDD(s string) (time.Time, error) {
	return time.ParseInLocation("20060102", s, time.Local)
}

func wrapMemo(s string, width int) []string {
	if width < 1 {
		width = 50
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	var out []string
	for _, para := range strings.Split(s, "\n") {
		for len(para) > width {
			cut := width
			if i := strings.LastIndex(para[:width], " "); i > 0 {
				cut = i
			}
			out = append(out, para[:cut])
			para = strings.TrimLeft(para[cut:], " ")
		}
		out = append(out, para)
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
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
