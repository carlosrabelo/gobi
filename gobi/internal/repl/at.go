package repl

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// PendingGET is one registered @ GET.
type PendingGET struct {
	Row, Col int
	Name     string
	Width    int
	Picture  string
	IsField  bool
	FieldIdx int
}

func ensureScreen(ctx *context.Context) *term.Screen {
	if ctx.Screen == nil {
		ctx.Screen = term.New()
	}
	return ctx.Screen
}

func (m *Mux) cmdAt(ctx *context.Context, cmd command.Command) error {
	args := strings.TrimSpace(cmd.Args)
	row, col, rest, err := parseAtCoord(args)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	up := strings.ToUpper(rest)
	switch {
	case strings.HasPrefix(up, "SAY "):
		return atSAY(ctx, row, col, strings.TrimSpace(rest[4:]))
	case strings.HasPrefix(up, "GET "):
		return m.atGET(ctx, row, col, strings.TrimSpace(rest[4:]))
	case strings.HasPrefix(up, "CLEAR"):
		return atClear(ctx, row, col, strings.TrimSpace(rest[5:]))
	case strings.Contains(up, " TO "):
		return atBox(ctx, row, col, rest)
	default:
		return fail(ctx, "@ requires SAY or GET")
	}
}

func atClear(ctx *context.Context, row, col int, rest string) error {
	scr := ensureScreen(ctx)
	r2, c2 := scr.Rows-1, scr.Cols-1
	up := strings.ToUpper(rest)
	if strings.HasPrefix(up, "TO ") {
		var err error
		r2, c2, _, err = parseAtCoord(strings.TrimSpace(rest[3:]))
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
	}
	for r := row; r <= r2 && r < scr.Rows; r++ {
		for c := col; c <= c2 && c < scr.Cols; c++ {
			scr.WriteString(r, c, " ", false)
		}
	}
	return flushScreen(ctx)
}

func atBox(ctx *context.Context, row, col int, rest string) error {
	up := strings.ToUpper(rest)
	double := strings.Contains(up, "DOUBLE")
	i := strings.Index(up, " TO ")
	if i < 0 {
		return fail(ctx, "@ TO required")
	}
	r2, c2, _, err := parseAtCoord(strings.TrimSpace(rest[i+4:]))
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	scr := ensureScreen(ctx)
	h, v, corner := byte('-'), byte('|'), byte('+')
	if double {
		h, v = '=', '|'
	}
	_ = corner
	for c := col; c <= c2 && c < scr.Cols; c++ {
		scr.WriteString(row, c, string(h), false)
		scr.WriteString(r2, c, string(h), false)
	}
	for r := row; r <= r2 && r < scr.Rows; r++ {
		scr.WriteString(r, col, string(v), false)
		scr.WriteString(r, c2, string(v), false)
	}
	scr.WriteString(row, col, "+", false)
	scr.WriteString(row, c2, "+", false)
	scr.WriteString(r2, col, "+", false)
	scr.WriteString(r2, c2, "+", false)
	return flushScreen(ctx)
}

func parseAtCoord(args string) (row, col int, rest string, err error) {
	args = strings.TrimSpace(args)
	i := 0
	for i < len(args) && (unicode.IsDigit(rune(args[i])) || args[i] == '-') {
		i++
	}
	if i == 0 {
		return 0, 0, "", errString("missing row")
	}
	row, err = strconv.Atoi(args[:i])
	if err != nil {
		return 0, 0, "", err
	}
	for i < len(args) && (args[i] == ' ' || args[i] == ',') {
		i++
	}
	j := i
	for j < len(args) && (unicode.IsDigit(rune(args[j])) || args[j] == '-') {
		j++
	}
	if j == i {
		return 0, 0, "", errString("missing column")
	}
	col, err = strconv.Atoi(args[i:j])
	if err != nil {
		return 0, 0, "", err
	}
	rest = strings.TrimSpace(args[j:])
	return row, col, rest, nil
}

func atSAY(ctx *context.Context, row, col int, rest string) error {
	src, pic := splitPicture(rest)
	tree, errs := expr.Parse(src)
	if len(errs) > 0 || tree == nil {
		msg := "invalid expression"
		if len(errs) > 0 {
			msg = errs[0]
		}
		return fail(ctx, "%s", msg)
	}
	v, err := expr.Eval(tree, newEnv(ctx))
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	text := v.String()
	if pic != "" {
		text = applyPicture(text, pic)
	}
	if strings.EqualFold(ctx.Config.Device, "PRINT") {
		// SET DEVICE TO PRINT: @ SAY goes to the print sink (023.15).
		fmt.Fprint(printSink(ctx), text)
		return nil
	}
	scr := ensureScreen(ctx)
	scr.WriteString(row, col, text, false)
	_ = flushScreen(ctx)
	return nil
}

func (m *Mux) atGET(ctx *context.Context, row, col int, rest string) error {
	if len(m.gets) >= 128 {
		return fail(ctx, "too many pending GETs")
	}
	name, pic := splitPicture(rest)
	name = strings.TrimSpace(name)
	if name == "" {
		return fail(ctx, "GET requires a name")
	}
	g := PendingGET{Row: row, Col: col, Name: strings.ToUpper(name), Picture: pic, Width: 10}
	a := ctx.Active()
	if a.HasTable() {
		fi := fieldIndex(a, name)
		if fi >= 0 {
			g.IsField = true
			g.FieldIdx = fi
			g.Width = int(a.Table.Header.Fields[fi].Length)
		}
	}
	m.gets = append(m.gets, g)
	text := getValueText(ctx, g)
	if len(text) > g.Width {
		text = text[:g.Width]
	}
	for len(text) < g.Width {
		text += " "
	}
	scr := ensureScreen(ctx)
	left, right := fieldDelims(ctx)
	writeDelimitedField(scr, row, col, text, ctx.Config.Intensity, left, right)
	_ = flushScreen(ctx)
	return nil
}

// fieldDelims returns the left/right characters around a full-screen field when
// SET DELIMITERS is on (023.7). One stored character is used on both sides.
func fieldDelims(ctx *context.Context) (left, right string) {
	if !ctx.Config.Delimiters {
		return "", ""
	}
	d := ctx.Config.DelimChars
	if d == "" {
		d = ":"
	}
	if len(d) >= 2 {
		return string(d[0]), string(d[1])
	}
	return d, d
}

func writeDelimitedField(scr *term.Screen, row, col int, text string, rev bool, left, right string) {
	c := col
	if left != "" {
		scr.WriteString(row, c, left, false)
		c++
	}
	scr.WriteString(row, c, text, rev)
	c += len(text)
	if right != "" {
		scr.WriteString(row, c, right, false)
	}
}

func splitPicture(rest string) (exprSrc, picture string) {
	up := strings.ToUpper(rest)
	for _, key := range []string{" PICTURE ", " USING "} {
		if i := strings.Index(up, key); i >= 0 {
			exprSrc = strings.TrimSpace(rest[:i])
			pic := strings.TrimSpace(rest[i+len(key):])
			pic = strings.Trim(pic, `"'`)
			return exprSrc, pic
		}
	}
	return strings.TrimSpace(rest), ""
}

func applyPicture(text, pic string) string {
	if pic == "" {
		return text
	}
	if n, err := strconv.ParseFloat(strings.TrimSpace(text), 64); err == nil {
		s, err := expr.Transform(expr.NumberVal(n), pic)
		if err == nil {
			return s
		}
	}
	s, err := expr.Transform(expr.StringVal(text), pic)
	if err != nil {
		return text
	}
	return s
}

func getValueText(ctx *context.Context, g PendingGET) string {
	if g.IsField {
		a := ctx.Active()
		if a.Landed && g.FieldIdx < len(a.Record.Values) {
			return fieldDisplay(a.Record.Values[g.FieldIdx], a.Table.Header.Fields[g.FieldIdx])
		}
		return ""
	}
	if v, ok := ctx.Symbols.Get(g.Name); ok {
		return v.String()
	}
	return ""
}

func fieldDisplay(v dbf.Value, f dbf.Field) string {
	switch f.Type {
	case 'N':
		return strings.TrimSpace(strconv.FormatFloat(v.Num, 'f', int(f.Decimals), 64))
	case 'L':
		if v.Log {
			return "T"
		}
		return "F"
	default:
		return v.Str
	}
}
