package repl

import (
	"io"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func cmdBROWSE(ctx *context.Context, cmd command.Command) error {
	ctx.KeyRead = true
	drainTypeaheadIntoKeys(ctx)
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	opts := parseBrowseOpts(cmd.Args)
	fields := a.Table.Header.Fields
	cols := make([]int, 0, len(fields))
	if len(opts.fields) > 0 {
		for _, name := range opts.fields {
			fi := fieldIndex(a, name)
			if fi < 0 {
				return fail(ctx, "unknown field: %s", name)
			}
			cols = append(cols, fi)
		}
	} else {
		for i := range fields {
			cols = append(cols, i)
		}
	}
	if len(cols) == 0 {
		return fail(ctx, "no fields to browse")
	}
	lockN := opts.lock
	if lockN < 0 {
		lockN = 0
	}
	if lockN > len(cols) {
		lockN = len(cols)
	}

	scr := ensureScreen(ctx)
	ctx.StatusMode = StatusModeBrowse
	defer func() { _ = leaveFullScreen(ctx) }()
	if ctx.Keys == nil {
		ctx.Keys = term.NewReader(ctx.Stdin)
	}
	keys := ctx.Keys
	keys.DrainPendingLineEnds()

	browseSeq := func() []int {
		if len(a.IndexOrder) > 0 {
			out := make([]int, len(a.IndexOrder))
			copy(out, a.IndexOrder)
			return out
		}
		n := a.RecCount()
		out := make([]int, n)
		for i := 0; i < n; i++ {
			out[i] = i + 1
		}
		return out
	}
	seq := browseSeq()
	pos := 0
	if a.Cursor >= 1 {
		for i, rn := range seq {
			if rn == a.Cursor {
				pos = i
				break
			}
		}
	}
	col := 0
	hScroll := 0 // offset into unlocked columns (cols[lockN:])
	editing := false
	editBuf := ""
	firstPaint := true

	physical := func() int {
		if len(seq) == 0 {
			return 1
		}
		if pos < 0 {
			pos = 0
		}
		if pos >= len(seq) {
			pos = len(seq) - 1
		}
		return seq[pos]
	}

	redraw := func() {
		blank := strings.Repeat(" ", scr.Cols)
		work := statusWorkRows(scr.Rows)
		for r := 0; r < work; r++ {
			scr.WriteString(r, 0, blank, false)
		}
		paintCols := func(row, rn, x int, header bool) int {
			paintOne := func(ci int) {
				if x >= scr.Cols {
					return
				}
				fi := cols[ci]
				w := browseColWidth(fields[fi], opts.width)
				var text string
				if header {
					text = fields[fi].Name
				} else {
					r, err := a.Table.Read(rn)
					if err != nil {
						text = ""
					} else {
						text = fieldDisplay(r.Values[fi], fields[fi])
						if fields[fi].Type == 'M' {
							text = memoBrowseLabel(a, r.Values[fi])
						}
						if rn == physical() && ci == col && editing {
							text = editBuf
						}
					}
				}
				if len(text) > w {
					text = text[:w]
				}
				rev := !header && rn == physical() && ci == col
				scr.WriteString(row, x, padRight(text, w), rev && ctx.Config.Intensity)
				x += w + 1
			}
			for ci := 0; ci < lockN; ci++ {
				paintOne(ci)
			}
			for ci := lockN + hScroll; ci < len(cols); ci++ {
				paintOne(ci)
			}
			return x
		}
		_ = paintCols(0, 0, 0, true)
		maxRows := work - 1
		if !opts.noMenu {
			if rem := menuReminder(ctx); rem != "" && work > 2 {
				scr.WriteString(work-1, 0, padRight(rem, scr.Cols), false)
				maxRows = work - 2
			}
		}
		if maxRows < 1 {
			maxRows = 1
		}
		for row := 0; row < maxRows; row++ {
			si := pos - maxRows/2 + row
			if si < 0 || si >= len(seq) {
				continue
			}
			_ = paintCols(row+1, seq[si], 0, false)
		}
		name := fields[cols[col]].Name
		if editing {
			ctx.Message = name + " *"
		} else {
			ctx.Message = name
		}
		paintStatusBand(ctx, StatusModeBrowse, true)
		if firstPaint {
			_ = flushScreenClear(ctx)
			firstPaint = false
		} else {
			_ = flushScreen(ctx)
		}
	}

	ensureColVisible := func() {
		if col < lockN {
			return
		}
		u := col - lockN
		if u < hScroll {
			hScroll = u
		}
		// Keep a few unlocked columns in view.
		if u >= hScroll+3 {
			hScroll = u - 2
		}
		maxH := len(cols) - lockN - 1
		if maxH < 0 {
			maxH = 0
		}
		if hScroll > maxH {
			hScroll = maxH
		}
		if hScroll < 0 {
			hScroll = 0
		}
	}

	startCellEdit := func() error {
		fi := cols[col]
		if opts.freeze != "" && !strings.EqualFold(fields[fi].Name, opts.freeze) {
			return nil
		}
		rec := physical()
		r, err := a.Table.Read(rec)
		if err != nil {
			return err
		}
		if fields[fi].Type == 'M' {
			text := memoFieldText(a, r.Values[fi])
			newText, ok, err := editMemoBuffer(ctx, fields[fi].Name, text)
			firstPaint = true
			if err != nil {
				return err
			}
			if ok {
				if err := writeMemoField(a, &r, fi, newText); err != nil {
					return err
				}
				_ = a.Table.Write(rec, r)
				_ = syncAllIndexes(ctx)
				if len(a.NDX) > 0 && a.NDX[0] != "" {
					_ = rebuildIndexOrder(ctx)
					oldPhys := rec
					seq = browseSeq()
					if opts.noFollow {
						if pos >= len(seq) && len(seq) > 0 {
							pos = len(seq) - 1
						}
					} else {
						for i, rn := range seq {
							if rn == oldPhys {
								pos = i
								break
							}
						}
					}
				}
			}
			redraw()
			return nil
		}
		editBuf = strings.TrimRight(fieldDisplay(r.Values[fi], fields[fi]), " ")
		editing = true
		redraw()
		return nil
	}

	commitCell := func() {
		fi := cols[col]
		rec := physical()
		v, err := parseFieldInput(fields[fi], editBuf)
		if err == nil {
			r, _ := a.Table.Read(rec)
			r.Values[fi] = v
			_ = a.Table.Write(rec, r)
			_ = syncAllIndexes(ctx)
			if len(a.NDX) > 0 && a.NDX[0] != "" {
				_ = rebuildIndexOrder(ctx)
				oldPhys := rec
				seq = browseSeq()
				if opts.noFollow {
					if pos >= len(seq) && len(seq) > 0 {
						pos = len(seq) - 1
					}
				} else {
					for i, rn := range seq {
						if rn == oldPhys {
							pos = i
							break
						}
					}
				}
			}
		}
		editing = false
		redraw()
	}

	appendBottom := func() error {
		if opts.noAppend {
			return nil
		}
		if uint64(a.RecCount()) >= maxRecords {
			return fail(ctx, "record count limit exceeded")
		}
		blank := dbf.Record{Values: make([]dbf.Value, len(fields))}
		if err := a.Table.Append(blank); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		_ = appendIndexKeys(ctx, a.RecCount())
		if len(a.NDX) > 0 && a.NDX[0] != "" {
			_ = rebuildIndexOrder(ctx)
		}
		seq = browseSeq()
		pos = len(seq) - 1
		redraw()
		return nil
	}

	redraw()
	for {
		k, err := keys.ReadKey()
		if err != nil {
			if err == io.EOF {
				a.Cursor = physical()
				_ = a.LoadCurrent()
				return nil
			}
			return fail(ctx, "%s", err.Error())
		}
		if k == term.KeyIns {
			ctx.InsertMode = !ctx.InsertMode
			redraw()
			continue
		}
		if editing {
			switch k {
			case term.KeyEsc:
				editing = false
				redraw()
			case term.KeyEnter:
				commitCell()
			case term.KeyBackspace:
				if len(editBuf) > 0 {
					editBuf = editBuf[:len(editBuf)-1]
					redraw()
				}
			default:
				if term.IsInputByte(k) {
					w := browseColWidth(fields[cols[col]], opts.width)
					editBuf = applyInputByte(editBuf, byte(k), w, ctx.InsertMode)
					redraw()
				}
			}
			continue
		}
		switch k {
		case term.KeyEsc, term.KeyCtrlQ:
			a.Cursor = physical()
			_ = a.LoadCurrent()
			return nil
		case term.KeyRight:
			if col < len(cols)-1 {
				col++
				ensureColVisible()
				redraw()
			}
		case term.KeyLeft:
			if col > 0 {
				col--
				ensureColVisible()
				redraw()
			}
		case term.KeyDown:
			if pos < len(seq)-1 {
				pos++
				redraw()
			} else if err := appendBottom(); err != nil {
				return err
			}
		case term.KeyUp:
			if pos > 0 {
				pos--
				redraw()
			}
		case term.KeyCtrlU:
			rec := physical()
			r, err := a.Table.Read(rec)
			if err == nil {
				r.Deleted = !r.Deleted
				_ = a.Table.Write(rec, r)
				redraw()
			}
		case term.KeyEnter:
			if err := startCellEdit(); err != nil {
				return fail(ctx, "%s", err.Error())
			}
		default:
			if term.IsInputByte(k) {
				fi := cols[col]
				if fields[fi].Type == 'M' {
					if err := startCellEdit(); err != nil {
						return fail(ctx, "%s", err.Error())
					}
					continue
				}
				if opts.freeze != "" && !strings.EqualFold(fields[fi].Name, opts.freeze) {
					continue
				}
				editBuf = string(byte(k))
				editing = true
				redraw()
			}
		}
	}
}

func browseColWidth(f dbf.Field, widthCap int) int {
	w := int(f.Length)
	if f.Type == 'M' && w < 4 {
		w = 4
	}
	if widthCap > 0 && w > widthCap {
		w = widthCap
	}
	if w < 1 {
		w = 1
	}
	return w
}

type browseOpts struct {
	fields   []string
	lock     int
	freeze   string
	noFollow bool
	noMenu   bool
	noAppend bool
	width    int
}

func parseBrowseOpts(args string) browseOpts {
	var o browseOpts
	toks := strings.Fields(args)
	for i := 0; i < len(toks); i++ {
		t := strings.ToUpper(toks[i])
		switch t {
		case "FIELDS":
			i++
			for i < len(toks) {
				u := strings.ToUpper(toks[i])
				if u == "LOCK" || u == "FREEZE" || u == "NOFOLLOW" || u == "NOMENU" || u == "NOAPPEND" || u == "WIDTH" {
					i--
					break
				}
				name := strings.Trim(toks[i], ",")
				if name != "" {
					o.fields = append(o.fields, name)
				}
				i++
			}
		case "LOCK":
			if i+1 < len(toks) {
				o.lock, _ = strconv.Atoi(toks[i+1])
				i++
			}
		case "FREEZE":
			if i+1 < len(toks) {
				o.freeze = toks[i+1]
				i++
			}
		case "NOFOLLOW":
			o.noFollow = true
		case "NOMENU":
			o.noMenu = true
		case "NOAPPEND":
			o.noAppend = true
		case "WIDTH":
			if i+1 < len(toks) {
				o.width, _ = strconv.Atoi(toks[i+1])
				i++
			}
		}
	}
	return o
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

// applyInputByte appends or overwrites at the end (legacy helper).
func applyInputByte(buf string, ch byte, max int, insert bool) string {
	pos := len(buf)
	if !insert && pos >= max && max > 0 {
		pos = max - 1
	}
	s, _ := applyInputAt(buf, pos, ch, max, insert)
	return s
}

// applyInputAt inserts or overwrites at pos; returns new buffer and caret.
func applyInputAt(buf string, pos int, ch byte, max int, insert bool) (string, int) {
	if max < 1 {
		return buf, pos
	}
	if pos < 0 {
		pos = 0
	}
	if pos > len(buf) {
		pos = len(buf)
	}
	if insert {
		if len(buf) >= max {
			return buf, pos
		}
		return buf[:pos] + string(ch) + buf[pos:], pos + 1
	}
	if pos >= len(buf) {
		if len(buf) >= max {
			return buf, pos
		}
		return buf + string(ch), pos + 1
	}
	return buf[:pos] + string(ch) + buf[pos+1:], pos + 1
}
