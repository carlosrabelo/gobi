package repl

import (
	"io"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func (m *Mux) cmdREAD(ctx *context.Context, cmd command.Command) error {
	ctx.KeyRead = true
	if len(m.gets) == 0 {
		return nil
	}
	keep := strings.Contains(strings.ToUpper(cmd.Args), "SAVE")
	keys := term.NewReader(ctx.Stdin)
	values := make([]string, len(m.gets))
	for i, g := range m.gets {
		values[i] = strings.TrimRight(getValueText(ctx, g), " ")
	}
	cur := 0
	scr := ensureScreen(ctx)
	left, right := fieldDelims(ctx)
	drawGET := func(i int, active bool) {
		g := m.gets[i]
		text := values[i]
		if g.Scroll > 0 && len(text) > g.Width {
			text = text[len(text)-g.Width:]
		} else if len(text) > g.Width {
			text = text[:g.Width]
		}
		for len(text) < g.Width {
			text += " "
		}
		rev := active && ctx.Config.Intensity
		writeDelimitedField(scr, g.Row, g.Col, text, rev, left, right)
	}
	for i := range m.gets {
		drawGET(i, i == cur)
	}
	_ = flushScreen(ctx)

	clearGets := func() {
		if !keep {
			m.gets = nil
		}
	}
	save := func() error {
		changed := false
		for i, g := range m.gets {
			old := strings.TrimRight(getValueText(ctx, g), " ")
			if values[i] != old {
				changed = true
			}
			if err := storeGET(ctx, g, values[i]); err != nil {
				return err
			}
		}
		ctx.Updated = changed
		return nil
	}

	for {
		k, err := keys.ReadKey()
		if err != nil {
			if err == io.EOF {
				if err := save(); err != nil {
					return err
				}
				ctx.ReadKey = 15
				if ctx.Updated {
					ctx.ReadKey = 271
				}
				clearGets()
				return nil
			}
			return fail(ctx, "%s", err.Error())
		}
		switch k {
		case term.KeyEsc:
			if !ctx.Config.Escape {
				continue // SET ESCAPE OFF: Esc does not cancel READ (023.9)
			}
			ctx.ReadKey = 12
			clearGets()
			return nil
		case term.KeyTab, term.KeyEnter:
			g := m.gets[cur]
			if g.HasRange && !valueInRange(ctx, g, values[cur]) {
				if ctx.Config.Bell {
					_, _ = ctx.Out().Write([]byte{7})
				}
				values[cur] = strings.TrimRight(getValueText(ctx, g), " ")
				drawGET(cur, true)
				_ = flushScreen(ctx)
				continue
			}
			if k == term.KeyEnter && cur == len(m.gets)-1 {
				if err := save(); err != nil {
					return err
				}
				ctx.ReadKey = 15
				if ctx.Updated {
					ctx.ReadKey = 271
				}
				clearGets()
				return nil
			}
			drawGET(cur, false)
			cur++
			if cur >= len(m.gets) {
				cur = 0
			}
			drawGET(cur, true)
			_ = flushScreen(ctx)
		case term.KeyBackspace:
			if len(values[cur]) > 0 {
				values[cur] = values[cur][:len(values[cur])-1]
				drawGET(cur, true)
				_ = flushScreen(ctx)
			}
		default:
			if term.IsInputByte(k) {
				g := m.gets[cur]
				ch := byte(k)
				if g.PicUpper {
					if ch >= 'a' && ch <= 'z' {
						ch = ch - 'a' + 'A'
					}
				}
				if g.PicAlpha && !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')) {
					if ctx.Config.Bell {
						_, _ = ctx.Out().Write([]byte{7})
					}
					continue
				}
				if g.Picture != "" && !pictureAllows(g.Picture, len(values[cur]), ch) {
					if ctx.Config.Bell {
						_, _ = ctx.Out().Write([]byte{7})
					}
					continue
				}
				maxLen := g.Width
				if g.Scroll > 0 {
					maxLen = 254
				}
				if len(values[cur]) < maxLen {
					values[cur] += string(ch)
					drawGET(cur, true)
					_ = flushScreen(ctx)
					// SET CONFIRM OFF: a full GET advances without Enter (023.6).
					if !ctx.Config.Confirm && g.Scroll == 0 && len(values[cur]) >= g.Width {
						if g.HasRange && !valueInRange(ctx, g, values[cur]) {
							if ctx.Config.Bell {
								_, _ = ctx.Out().Write([]byte{7})
							}
							values[cur] = strings.TrimRight(getValueText(ctx, g), " ")
							drawGET(cur, true)
							_ = flushScreen(ctx)
							continue
						}
						if cur == len(m.gets)-1 {
							if err := save(); err != nil {
								return err
							}
							ctx.ReadKey = 15
							if ctx.Updated {
								ctx.ReadKey = 271
							}
							clearGets()
							return nil
						}
						drawGET(cur, false)
						cur++
						drawGET(cur, true)
						_ = flushScreen(ctx)
					}
				}
			}
		}
	}
}

func pictureAllows(pic string, pos int, ch byte) bool {
	template := pic
	if strings.HasPrefix(pic, "@") {
		if i := strings.IndexByte(pic, ' '); i >= 0 {
			template = pic[i+1:]
		} else {
			return true
		}
	}
	// Walk template slots (skip literals) to the pos-th editable cell.
	slot := 0
	for i := 0; i < len(template); i++ {
		c := template[i]
		switch c {
		case '9', '#', 'A', 'L', 'N', 'X', 'Y', '!', '$', '*':
			if slot == pos {
				return pictureSlotAllows(c, ch)
			}
			slot++
		case '.', ',':
			// display-only in template for input walk
		default:
			// literal
		}
	}
	return false
}

func pictureSlotAllows(slot, ch byte) bool {
	switch slot {
	case '9':
		return ch >= '0' && ch <= '9' || ch == '-' || ch == '+'
	case '#':
		return ch >= '0' && ch <= '9' || ch == ' ' || ch == '-' || ch == '+' || ch == '.'
	case 'A':
		return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
	case 'L':
		return ch == 'T' || ch == 't' || ch == 'F' || ch == 'f' || ch == 'Y' || ch == 'y' || ch == 'N' || ch == 'n'
	case 'N':
		return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
	case 'Y':
		return ch == 'Y' || ch == 'y' || ch == 'N' || ch == 'n'
	case '!', 'X', '$', '*':
		return true
	default:
		return false
	}
}

func storeGET(ctx *context.Context, g PendingGET, text string) error {
	text = strings.TrimRight(text, " ")
	if g.PicR {
		text = expr.StripPictureLiterals(text, g.Picture)
	}
	if g.IsField {
		a := ctx.Active()
		if !a.Landed {
			return fail(ctx, "no current record")
		}
		f := a.Table.Header.Fields[g.FieldIdx]
		v, err := parseFieldInput(f, text)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		rec := a.Record
		rec.Values[g.FieldIdx] = v
		if err := a.Table.Write(a.Cursor, rec); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		a.Record = rec
		return syncAllIndexes(ctx)
	}
	var val expr.Value
	if old, ok := ctx.Symbols.Get(g.Name); ok {
		switch old.Kind {
		case expr.KindNumber:
			n, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
			if err != nil {
				if strings.TrimSpace(text) == "" {
					n = 0
				} else {
					return fail(ctx, "%s", err.Error())
				}
			}
			val = expr.NumberVal(n)
		case expr.KindLogical:
			val = expr.LogicalVal(text == "T" || text == "t" || text == "Y" || text == "y" || text == ".T.")
		default:
			val = expr.StringVal(text)
		}
	} else {
		val = expr.StringVal(text)
	}
	if err := ctx.Symbols.Set(g.Name, val); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return nil
}
