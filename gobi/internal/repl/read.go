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
	drawGET := func(i int, active bool) {
		g := m.gets[i]
		text := values[i]
		if len(text) > g.Width {
			text = text[:g.Width]
		}
		for len(text) < g.Width {
			text += " "
		}
		rev := active && ctx.Config.Intensity
		scr.WriteString(g.Row, g.Col, text, rev)
	}
	for i := range m.gets {
		drawGET(i, i == cur)
	}
	_ = scr.Present(ctx.Stdout)

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
			ctx.ReadKey = 12
			clearGets()
			return nil
		case term.KeyTab, term.KeyEnter:
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
			_ = scr.Present(ctx.Stdout)
		case term.KeyBackspace:
			if len(values[cur]) > 0 {
				values[cur] = values[cur][:len(values[cur])-1]
				drawGET(cur, true)
				_ = scr.Present(ctx.Stdout)
			}
		default:
			if k >= 32 && k < 127 {
				g := m.gets[cur]
				ch := byte(k)
				if g.Picture != "" && !pictureAllows(g.Picture, len(values[cur]), ch) {
					if ctx.Config.Bell {
						_, _ = ctx.Stdout.Write([]byte{7})
					}
					continue
				}
				if len(values[cur]) < g.Width {
					values[cur] += string(ch)
					drawGET(cur, true)
					_ = scr.Present(ctx.Stdout)
				}
			}
		}
	}
}

func pictureAllows(pic string, pos int, ch byte) bool {
	if pos >= len(pic) {
		return false
	}
	switch pic[pos] {
	case '9':
		return ch >= '0' && ch <= '9' || ch == '-' || ch == '+'
	case 'A':
		return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
	case 'X', '$', '.':
		return true
	default:
		return ch == pic[pos]
	}
}

func storeGET(ctx *context.Context, g PendingGET, text string) error {
	text = strings.TrimRight(text, " ")
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
