package repl

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func (m *Mux) cmdEDIT(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	scope := parseScope(cmd)
	forExpr, err := parseOptionalLogic(cmd.Clauses["FOR"])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	whileExpr, err := parseOptionalLogic(cmd.Clauses["WHILE"])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}

	targets, startIdx, err := editTargetList(ctx, scope, forExpr, whileExpr)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if !a.Landed {
		return fail(ctx, "no current record")
	}
	if len(ctx.FormatLines) > 0 {
		// Format forms: walk targets with READ; bare EDIT keeps one form.
		if targets == nil {
			return m.formatForm(ctx, false)
		}
		for i := startIdx; i < len(targets); i++ {
			if err := ctx.Goto(targets[i]); err != nil {
				return fail(ctx, "%s", err.Error())
			}
			if err := m.formatForm(ctx, false); err != nil {
				return err
			}
			// Esc / Ctrl-Q during READ leaves formatForm; stop the walk.
			if ctx.ReadKey == term.KeyEsc || ctx.ReadKey == term.KeyCtrlQ {
				return nil
			}
		}
		return nil
	}
	return editForm(ctx, false, targets, startIdx)
}

// editTargetList builds the record numbers EDIT will visit.
// nil targets means free navigation from the current record (Ctrl-X / Ctrl-R / PgUp / PgDn).
func editTargetList(ctx *context.Context, scope Scope, forExpr, whileExpr expr.Expression) (targets []int, startIdx int, err error) {
	a := ctx.Active()
	rest := strings.TrimSpace(scope.Rest)

	// EDIT n — single record by number (no scope / filter).
	if scope.Kind == "" && forExpr == nil && whileExpr == nil && rest != "" {
		fields := strings.Fields(rest)
		if len(fields) == 1 {
			n, e := strconv.Atoi(fields[0])
			if e != nil {
				return nil, 0, fmt.Errorf("invalid record number")
			}
			if e := ctx.Goto(n); e != nil {
				return nil, 0, e
			}
			return []int{n}, 0, nil
		}
		return nil, 0, fmt.Errorf("invalid EDIT arguments")
	}

	// Plain EDIT — free nav on current record.
	if scope.Kind == "" && forExpr == nil && whileExpr == nil {
		return nil, 0, nil
	}

	count := a.RecCount()
	start, end := scopeRange(count, a.Cursor, scope)
	if scope.Kind == "" {
		start = 1
		end = count
		if whileExpr != nil {
			start = a.Cursor
			if start < 1 {
				start = 1
			}
		}
	}
	if end < start {
		return []int{}, 0, nil
	}

	env := newEnv(ctx)
	saved := a.Cursor
	for n := start; n <= end; n++ {
		a.Cursor = n
		a.BOF = false
		if e := a.LoadCurrent(); e != nil {
			a.Cursor = saved
			_ = a.LoadCurrent()
			return nil, 0, e
		}
		if ctx.Config.Deleted && a.Record.Deleted {
			continue
		}
		if ok, e := recordPassesFilter(ctx, a); e != nil {
			a.Cursor = saved
			_ = a.LoadCurrent()
			return nil, 0, e
		} else if !ok {
			continue
		}
		if whileExpr != nil {
			ok, e := evalLogic(whileExpr, env)
			if e != nil {
				a.Cursor = saved
				_ = a.LoadCurrent()
				return nil, 0, e
			}
			if !ok {
				break
			}
		}
		if forExpr != nil {
			ok, e := evalLogic(forExpr, env)
			if e != nil {
				a.Cursor = saved
				_ = a.LoadCurrent()
				return nil, 0, e
			}
			if !ok {
				continue
			}
		}
		targets = append(targets, n)
	}
	a.Cursor = saved
	_ = a.LoadCurrent()
	if len(targets) == 0 {
		return targets, 0, nil
	}
	// Prefer starting at the current record when it is in the set.
	startIdx = 0
	for i, n := range targets {
		if n == saved || (saved < 1 && i == 0) {
			startIdx = i
			break
		}
		if n >= saved {
			startIdx = i
			break
		}
	}
	if e := ctx.Goto(targets[startIdx]); e != nil {
		return nil, 0, e
	}
	return targets, startIdx, nil
}

// formatForm runs SET FORMAT TO @ lines then READ (023.17).
func (m *Mux) formatForm(ctx *context.Context, appendMode bool) error {
	ctx.KeyRead = true
	for {
		if appendMode {
			if err := appendBlank(ctx); err != nil {
				return err
			}
		}
		m.gets = nil
		scr := ensureScreen(ctx)
		scr.Clear()
		_ = flushScreenClear(ctx)
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

// editForm is the full-screen APPEND/EDIT form.
// targets nil = free record navigation; non-nil = walk that list (FOR/WHILE/scope).
func editForm(ctx *context.Context, appendMode bool, targets []int, targetIdx int) error {
	ctx.KeyRead = true
	a := ctx.Active()
	fields := a.Table.Header.Fields
	scr := ensureScreen(ctx)
	if appendMode {
		ctx.StatusMode = StatusModeAppend
	} else {
		ctx.StatusMode = StatusModeEdit
	}
	defer func() { _ = leaveFullScreen(ctx) }()
	if ctx.Keys == nil {
		ctx.Keys = term.NewReader(ctx.Stdin)
	}
	keys := ctx.Keys
	keys.DrainPendingLineEnds()

	if !appendMode && targets != nil && len(targets) == 0 {
		return nil
	}

	for {
		if appendMode && uint64(a.RecCount()) >= maxRecords {
			return fail(ctx, "record count limit exceeded")
		}
		if !appendMode && targets != nil {
			if targetIdx < 0 || targetIdx >= len(targets) {
				return nil
			}
			if err := ctx.Goto(targets[targetIdx]); err != nil {
				return fail(ctx, "%s", err.Error())
			}
		}
		values := make([]string, len(fields))
		memoTexts := make([]string, len(fields))
		loadValues := func(rec dbf.Record) {
			for i, f := range fields {
				if f.Type == 'M' {
					memoTexts[i] = memoFieldText(a, rec.Values[i])
					if memoTexts[i] != "" {
						values[i] = "memo"
					} else {
						values[i] = ""
					}
					continue
				}
				values[i] = strings.TrimRight(fieldDisplay(rec.Values[i], f), " ")
			}
		}
		switch {
		case !appendMode:
			loadValues(a.Record)
		case ctx.Config.Carry && a.RecCount() > 0:
			rec := a.Record
			if !a.Landed {
				if r, err := a.Table.Read(a.RecCount()); err == nil {
					rec = r
				}
			}
			loadValues(rec)
		}
		cur := 0
		fieldScroll := 0
		frameReady := false

		workRows := func() int {
			w := statusWorkRows(scr.Rows)
			if w < 3 {
				return 3
			}
			return w
		}
		maxFields := func() int {
			n := workRows() - 2 // title + reminder/status gap
			if rem := menuReminder(ctx); rem != "" {
				n = workRows() - 2
			}
			if n < 1 {
				n = 1
			}
			return n
		}
		ensureFieldVisible := func() {
			mf := maxFields()
			if cur < fieldScroll {
				fieldScroll = cur
			}
			if cur >= fieldScroll+mf {
				fieldScroll = cur - mf + 1
			}
			if fieldScroll < 0 {
				fieldScroll = 0
			}
		}

		paintForm := func() {
			ensureFieldVisible()
			work := workRows()
			blank := strings.Repeat(" ", scr.Cols)
			for r := 0; r < work; r++ {
				scr.WriteString(r, 0, blank, false)
			}
			title := "APPEND"
			if !appendMode {
				title = fmt.Sprintf("RECORD #%5d", a.Cursor)
				if a.Record.Deleted {
					title += " *"
				}
			}
			scr.WriteString(0, 0, padRight(title, scr.Cols), false)
			if rem := menuReminder(ctx); rem != "" && work > 2 {
				scr.WriteString(work-1, 0, padRight(rem, scr.Cols), false)
			}
			dLeft, dRight := fieldDelims(ctx)
			mf := maxFields()
			for vis := 0; vis < mf; vis++ {
				i := fieldScroll + vis
				if i >= len(fields) {
					break
				}
				f := fields[i]
				label := f.Name + ": "
				scr.WriteString(vis+1, 0, padRight(label, scr.Cols), false)
				text := values[i]
				w := int(f.Length)
				if w > scr.Cols-len(label)-2 {
					w = scr.Cols - len(label) - 2
				}
				if w < 1 {
					w = 1
				}
				if len(text) > w {
					text = text[:w]
				}
				text = padRight(text, w)
				writeDelimitedField(scr, vis+1, len(label), text, i == cur && ctx.Config.Intensity, dLeft, dRight)
			}
			paintStatusBand(ctx, ctx.StatusMode, true)
			if !frameReady {
				_ = flushScreenClear(ctx)
				frameReady = true
			} else {
				_ = flushScreen(ctx)
			}
		}
		paintForm()

		saveRecord := func() error {
			rec := dbf.Record{Values: make([]dbf.Value, len(fields))}
			if !appendMode {
				rec.Deleted = a.Record.Deleted
			}
			for i, f := range fields {
				if f.Type == 'M' {
					if err := writeMemoField(a, &rec, i, memoTexts[i]); err != nil {
						return err
					}
					continue
				}
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

		openMemo := func() error {
			f := fields[cur]
			if f.Type != 'M' {
				return nil
			}
			newText, ok, err := editMemoBuffer(ctx, f.Name, memoTexts[cur])
			frameReady = false // force clear after nested surface
			if err != nil {
				return err
			}
			if ok {
				memoTexts[cur] = newText
				if strings.TrimSpace(newText) != "" {
					values[cur] = "memo"
				} else {
					values[cur] = ""
				}
			}
			paintForm()
			return nil
		}

		moveRecord := func(delta int) (leave bool, err error) {
			if appendMode {
				return false, nil
			}
			if err := saveRecord(); err != nil {
				return false, err
			}
			if targets != nil {
				next := targetIdx + delta
				if next < 0 || next >= len(targets) {
					return true, nil
				}
				targetIdx = next
				return false, nil
			}
			if delta > 0 {
				if a.Cursor >= a.RecCount() {
					return true, nil
				}
				if err := ctx.Skip(1); err != nil {
					return false, err
				}
			} else {
				if a.Cursor <= 1 {
					return true, nil
				}
				if err := ctx.Skip(-1); err != nil {
					return false, err
				}
			}
			return false, nil
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
					continue
				}
				return nil
			case term.KeyCtrlQ:
				return nil
			case term.KeyCtrlW:
				if err := saveRecord(); err != nil {
					return fail(ctx, "%s", err.Error())
				}
				if appendMode {
					goto outer
				}
				if targets != nil {
					targetIdx++
					if targetIdx >= len(targets) {
						return nil
					}
					goto outer
				}
				return nil
			case term.KeyCtrlX, term.KeyPgDn:
				leave, err := moveRecord(+1)
				if err != nil {
					return fail(ctx, "%s", err.Error())
				}
				if leave {
					return nil
				}
				goto outer
			case term.KeyCtrlR, term.KeyPgUp:
				leave, err := moveRecord(-1)
				if err != nil {
					return fail(ctx, "%s", err.Error())
				}
				if leave {
					return nil
				}
				goto outer
			case term.KeyCtrlU:
				if !appendMode {
					a.Record.Deleted = !a.Record.Deleted
					_ = a.Table.Write(a.Cursor, a.Record)
					paintForm()
				}
			case term.KeyCtrlY:
				values[cur] = ""
				paintForm()
			case term.KeyDown, term.KeyTab:
				if len(fields) > 0 {
					cur = (cur + 1) % len(fields)
					paintForm()
				}
			case term.KeyEnter:
				if len(fields) > 0 && fields[cur].Type == 'M' {
					if err := openMemo(); err != nil {
						return fail(ctx, "%s", err.Error())
					}
					continue
				}
				if len(fields) > 0 {
					cur = (cur + 1) % len(fields)
					paintForm()
				}
			case term.KeyUp:
				if len(fields) > 0 {
					cur = (cur - 1 + len(fields)) % len(fields)
					paintForm()
				}
			case term.KeyBackspace:
				if fields[cur].Type == 'M' {
					continue
				}
				if len(values[cur]) > 0 {
					values[cur] = values[cur][:len(values[cur])-1]
					paintForm()
				}
			default:
				if term.IsInputByte(k) {
					if fields[cur].Type == 'M' {
						if err := openMemo(); err != nil {
							return fail(ctx, "%s", err.Error())
						}
						continue
					}
					w := int(fields[cur].Length)
					if len(values[cur]) < w {
						values[cur] += string(byte(k))
						paintForm()
						if !ctx.Config.Confirm && len(values[cur]) >= w {
							cur = (cur + 1) % len(fields)
							paintForm()
						}
					}
				}
			}
		}
	outer:
	}
}
