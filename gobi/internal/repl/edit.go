package repl

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func (m *Mux) cmdEDIT(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	args := strings.TrimSpace(cmd.Args)
	if args != "" {
		n, err := strconv.Atoi(strings.Fields(args)[0])
		if err != nil {
			return fail(ctx, "invalid record number")
		}
		if err := ctx.Goto(n); err != nil {
			return fail(ctx, "%s", err.Error())
		}
	}
	if !a.Landed {
		return fail(ctx, "no current record")
	}
	if len(ctx.FormatLines) > 0 {
		return m.formatForm(ctx, false)
	}
	return editForm(ctx, false)
}

// formatForm runs SET FORMAT TO @ lines then READ (023.17).
func (m *Mux) formatForm(ctx *context.Context, appendMode bool) error {
	for {
		if appendMode {
			if err := appendBlank(ctx); err != nil {
				return err
			}
		}
		m.gets = nil
		scr := ensureScreen(ctx)
		scr.Clear()
		_ = scr.PresentClear(ctx.Out())
		for _, line := range ctx.FormatLines {
			if isCommentLine(line) {
				continue
			}
			trim := strings.TrimSpace(line)
			cmd := command.Split(trim)
			if cmd.Verb != "@" {
				continue
			}
			if err := m.cmdAt(ctx, cmd); err != nil {
				return err
			}
		}
		if err := m.cmdREAD(ctx, command.Command{Verb: "READ"}); err != nil {
			return err
		}
		if !appendMode {
			return nil
		}
		if ctx.ReadKey == 12 || ctx.ReadKey == 268 {
			return nil
		}
	}
}

func (m *Mux) cmdREADWithFormat(ctx *context.Context, cmd command.Command) error {
	if len(ctx.FormatLines) > 0 && len(m.gets) == 0 {
		for _, line := range ctx.FormatLines {
			if isCommentLine(line) {
				continue
			}
			c := command.Split(strings.TrimSpace(line))
			if c.Verb != "@" {
				continue
			}
			if err := m.cmdAt(ctx, c); err != nil {
				return err
			}
		}
	}
	return m.cmdREAD(ctx, cmd)
}

func editForm(ctx *context.Context, appendMode bool) error {
	a := ctx.Active()
	fields := a.Table.Header.Fields
	scr := ensureScreen(ctx)
	keys := term.NewReader(ctx.Stdin)

	for {
		if appendMode && uint64(a.RecCount()) >= maxRecords {
			return fail(ctx, "record count limit exceeded")
		}
		values := make([]string, len(fields))
		switch {
		case !appendMode:
			for i, f := range fields {
				values[i] = strings.TrimRight(fieldDisplay(a.Record.Values[i], f), " ")
			}
		case ctx.Config.Carry && a.RecCount() > 0:
			// SET CARRY ON: seed the new APPEND form from the previous record (023.5).
			rec := a.Record
			if !a.Landed {
				if r, err := a.Table.Read(a.RecCount()); err == nil {
					rec = r
				}
			}
			for i, f := range fields {
				values[i] = strings.TrimRight(fieldDisplay(rec.Values[i], f), " ")
			}
		}
		cur := 0
		redraw := func() {
			scr.Clear()
			title := "APPEND"
			if !appendMode {
				title = fmt.Sprintf("RECORD #%5d", a.Cursor)
			}
			scr.WriteString(0, 0, title, false)
			dLeft, dRight := fieldDelims(ctx)
			for i, f := range fields {
				label := f.Name + ": "
				scr.WriteString(i+1, 0, label, false)
				text := values[i]
				w := int(f.Length)
				if len(text) > w {
					text = text[:w]
				}
				for len(text) < w {
					text += " "
				}
				writeDelimitedField(scr, i+1, len(label), text, i == cur && ctx.Config.Intensity, dLeft, dRight)
			}
			_ = scr.PresentClear(ctx.Out())
		}
		redraw()

		saveRecord := func() error {
			rec := dbf.Record{Values: make([]dbf.Value, len(fields))}
			if !appendMode {
				rec.Deleted = a.Record.Deleted
			}
			for i, f := range fields {
				v, err := parseFieldInput(f, values[i])
				if err != nil {
					return err
				}
				rec.Values[i] = v
			}
			if appendMode {
				if err := a.Table.Append(rec); err != nil {
					return err
				}
				a.Cursor = a.RecCount()
				a.BOF = false
				if err := a.LoadCurrent(); err != nil {
					return err
				}
				return appendIndexKeys(ctx, a.Cursor)
			}
			if err := a.Table.Write(a.Cursor, rec); err != nil {
				return err
			}
			a.Record = rec
			return syncAllIndexes(ctx)
		}

		for {
			k, err := keys.ReadKey()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return fail(ctx, "%s", err.Error())
			}
			switch k {
			case term.KeyEsc:
				if !ctx.Config.Escape {
					continue // SET ESCAPE OFF: Esc does not leave the form (023.9)
				}
				return nil
			case term.KeyCtrlQ:
				return nil
			case term.KeyCtrlW:
				if err := saveRecord(); err != nil {
					return fail(ctx, "%s", err.Error())
				}
				if appendMode {
					break // outer: new blank
				}
				return nil
			case term.KeyCtrlX:
				if appendMode {
					continue
				}
				if err := saveRecord(); err != nil {
					return fail(ctx, "%s", err.Error())
				}
				if a.Cursor >= a.RecCount() {
					return nil
				}
				if err := ctx.Skip(1); err != nil {
					return fail(ctx, "%s", err.Error())
				}
				goto outer
			case term.KeyCtrlR:
				if appendMode {
					continue
				}
				if err := saveRecord(); err != nil {
					return fail(ctx, "%s", err.Error())
				}
				if a.Cursor <= 1 {
					return nil
				}
				if err := ctx.Skip(-1); err != nil {
					return fail(ctx, "%s", err.Error())
				}
				goto outer
			case term.KeyCtrlU:
				if !appendMode {
					a.Record.Deleted = !a.Record.Deleted
					_ = a.Table.Write(a.Cursor, a.Record)
					redraw()
				}
			case term.KeyCtrlY:
				values[cur] = ""
				redraw()
			case term.KeyTab, term.KeyEnter:
				cur = (cur + 1) % len(fields)
				redraw()
			case term.KeyBackspace:
				if len(values[cur]) > 0 {
					values[cur] = values[cur][:len(values[cur])-1]
					redraw()
				}
			default:
				if k >= 32 && k < 127 {
					w := int(fields[cur].Length)
					if len(values[cur]) < w {
						values[cur] += string(byte(k))
						redraw()
						// SET CONFIRM OFF: leaving a full field advances without Enter (023.6).
						if !ctx.Config.Confirm && len(values[cur]) >= w {
							cur = (cur + 1) % len(fields)
							redraw()
						}
					}
				}
			}
			if k == term.KeyCtrlW && appendMode {
				goto outer
			}
		}
	outer:
	}
}
