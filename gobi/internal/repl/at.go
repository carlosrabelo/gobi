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
	Scroll   int  // @Sn display width; 0 means no scroll
	PicUpper bool // @!
	PicAlpha bool // @A
	PicR     bool // @R literals not stored
	HasRange bool
	RangeLo  expr.Value
	RangeHi  expr.Value
	HasLo    bool
	HasHi    bool
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
		s, err := expr.TransformWithDate(v, pic, ctx.Config.DateOrder, ctx.Config.Century)
		if err == nil {
			text = s
		} else {
			text = applyPicture(text, pic)
		}
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
	name, pic, rangeSrc, err := splitGETRest(rest)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fail(ctx, "GET requires a name")
	}
	g := PendingGET{Row: row, Col: col, Name: strings.ToUpper(name), Picture: pic, Width: 10}
	flags := expr.PictureFuncFlags(pic)
	g.PicUpper = strings.Contains(flags, "!")
	g.PicAlpha = strings.Contains(flags, "A")
	g.PicR = strings.Contains(flags, "R")
	g.Scroll = expr.PictureScrollWidth(pic)
	if rangeSrc != "" {
		lo, hi, hasLo, hasHi, err := parseRANGE(ctx, rangeSrc)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		g.HasRange = true
		g.RangeLo, g.RangeHi = lo, hi
		g.HasLo, g.HasHi = hasLo, hasHi
	}
	a := ctx.Active()
	if a.HasTable() {
		fi := fieldIndex(a, name)
		if fi >= 0 {
			g.IsField = true
			g.FieldIdx = fi
			g.Width = int(a.Table.Header.Fields[fi].Length)
		}
	}
	if g.Scroll > 0 {
		g.Width = g.Scroll
	} else if tmpl := expr.PictureTemplate(pic); tmpl != "" {
		n := 0
		for i := 0; i < len(tmpl); i++ {
			switch tmpl[i] {
			case '9', '#', 'A', 'L', 'N', 'X', 'Y', '!', '$', '*':
				n++
			}
		}
		if n > 0 {
			g.Width = n
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

// splitGETRest pulls name, optional PICTURE/USING, and optional RANGE clause.
func splitGETRest(rest string) (name, pic, rangeSrc string, err error) {
	up := strings.ToUpper(rest)
	body := rest
	if i := strings.Index(up, " RANGE "); i >= 0 {
		rangeSrc = strings.TrimSpace(rest[i+7:])
		body = strings.TrimSpace(rest[:i])
		up = strings.ToUpper(body)
	}
	name, pic = splitPicture(body)
	if name == "" && pic == "" && rangeSrc == "" {
		return "", "", "", errString("GET requires a name")
	}
	return name, pic, rangeSrc, nil
}

func parseRANGE(ctx *context.Context, src string) (lo, hi expr.Value, hasLo, hasHi bool, err error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return lo, hi, false, false, errString("RANGE requires bounds")
	}
	// Split on first comma at top level (expressions may nest later; keep simple).
	comma := strings.IndexByte(src, ',')
	var loSrc, hiSrc string
	if comma < 0 {
		loSrc = strings.TrimSpace(src)
	} else {
		loSrc = strings.TrimSpace(src[:comma])
		hiSrc = strings.TrimSpace(src[comma+1:])
	}
	env := newEnv(ctx)
	if loSrc != "" {
		tree, errs := expr.Parse(loSrc)
		if len(errs) > 0 || tree == nil {
			return lo, hi, false, false, errString("invalid RANGE low")
		}
		lo, err = expr.Eval(tree, env)
		if err != nil {
			return lo, hi, false, false, err
		}
		hasLo = true
	}
	if hiSrc != "" {
		tree, errs := expr.Parse(hiSrc)
		if len(errs) > 0 || tree == nil {
			return lo, hi, false, false, errString("invalid RANGE high")
		}
		hi, err = expr.Eval(tree, env)
		if err != nil {
			return lo, hi, false, false, err
		}
		hasHi = true
	}
	return lo, hi, hasLo, hasHi, nil
}

func valueInRange(ctx *context.Context, g PendingGET, text string) bool {
	if !g.HasRange {
		return true
	}
	v, err := parseGETValue(ctx, g, text)
	if err != nil {
		return false
	}
	if g.HasLo {
		if cmpRange(v, g.RangeLo) < 0 {
			return false
		}
	}
	if g.HasHi {
		if cmpRange(v, g.RangeHi) > 0 {
			return false
		}
	}
	return true
}

func parseGETValue(ctx *context.Context, g PendingGET, text string) (expr.Value, error) {
	text = strings.TrimRight(text, " ")
	if g.IsField {
		a := ctx.Active()
		f := a.Table.Header.Fields[g.FieldIdx]
		switch f.Type {
		case 'N':
			s := strings.TrimSpace(text)
			if s == "" {
				return expr.NumberVal(0), nil
			}
			n, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return expr.Value{}, err
			}
			return expr.NumberVal(n), nil
		case 'D':
			return expr.ParseDateString(text, ctx.Config.DateOrder), nil
		case 'L':
			return expr.LogicalVal(text == "T" || text == "t" || text == "Y" || text == "y" || text == ".T."), nil
		default:
			return expr.StringVal(text), nil
		}
	}
	if old, ok := ctx.Symbols.Get(g.Name); ok {
		switch old.Kind {
		case expr.KindNumber:
			n, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
			if err != nil {
				if strings.TrimSpace(text) == "" {
					return expr.NumberVal(0), nil
				}
				return expr.Value{}, err
			}
			return expr.NumberVal(n), nil
		case expr.KindDate:
			return expr.ParseDateString(text, ctx.Config.DateOrder), nil
		case expr.KindLogical:
			return expr.LogicalVal(text == "T" || text == "t" || text == "Y" || text == "y" || text == ".T."), nil
		default:
			return expr.StringVal(text), nil
		}
	}
	if n, err := strconv.ParseFloat(strings.TrimSpace(text), 64); err == nil {
		return expr.NumberVal(n), nil
	}
	return expr.StringVal(text), nil
}

func cmpRange(a, b expr.Value) int {
	if a.Kind == expr.KindDate && b.Kind == expr.KindDate {
		if a.EmptyDate || b.EmptyDate {
			return 0
		}
		if a.Date.Before(b.Date) {
			return -1
		}
		if a.Date.After(b.Date) {
			return 1
		}
		return 0
	}
	an := a.Num
	bn := b.Num
	if a.Kind != expr.KindNumber {
		an, _ = strconv.ParseFloat(strings.TrimSpace(a.String()), 64)
	}
	if b.Kind != expr.KindNumber {
		bn, _ = strconv.ParseFloat(strings.TrimSpace(b.String()), 64)
	}
	if an < bn {
		return -1
	}
	if an > bn {
		return 1
	}
	return 0
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
