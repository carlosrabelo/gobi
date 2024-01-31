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

	scr := ensureScreen(ctx)
	ctx.StatusMode = StatusModeBrowse
	defer func() { _ = leaveFullScreen(ctx) }()
	if ctx.Keys == nil {
		ctx.Keys = term.NewReader(ctx.Stdin)
	}
	keys := ctx.Keys
	keys.DrainPendingLineEnds()
	rec := a.Cursor
	if rec < 1 {
		rec = 1
	}
	col := 0
	hScroll := 0
	editing := false
	editBuf := ""
	firstPaint := true

	redraw := func() {
		blank := strings.Repeat(" ", scr.Cols)
		work := statusWorkRows(scr.Rows)
		for r := 0; r < work; r++ {
			scr.WriteString(r, 0, blank, false)
		}
		_ = opts.lock
		startCol := hScroll
		if startCol >= len(cols) {
			startCol = len(cols) - 1
		}
		x := 0
		for ci := startCol; ci < len(cols) && x < scr.Cols; ci++ {
			fi := cols[ci]
			w := browseColWidth(fields[fi], opts.width)
			hdr := fields[fi].Name
			if len(hdr) > w {
				hdr = hdr[:w]
			}
			scr.WriteString(0, x, padRight(hdr, w), false)
			x += w + 1
		}
		maxRows := work - 1
		if maxRows < 1 {
			maxRows = 1
		}
		for row := 0; row < maxRows; row++ {
			rn := rec - maxRows/2 + row
			if rn < 1 || rn > a.RecCount() {
				continue
			}
			r, err := a.Table.Read(rn)
			if err != nil {
				continue
			}
			x := 0
			for ci := startCol; ci < len(cols) && x < scr.Cols; ci++ {
				fi := cols[ci]
				w := browseColWidth(fields[fi], opts.width)
				text := fieldDisplay(r.Values[fi], fields[fi])
				if fields[fi].Type == 'M' {
					text = memoBrowseLabel(a, r.Values[fi])
				}
				if rn == rec && ci == col && editing {
					text = editBuf
				}
				if len(text) > w {
					text = text[:w]
				}
				rev := rn == rec && ci == col
				scr.WriteString(row+1, x, padRight(text, w), rev && ctx.Config.Intensity)
				x += w + 1
			}
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

	startCellEdit := func() error {
		fi := cols[col]
		if opts.freeze != "" && !strings.EqualFold(fields[fi].Name, opts.freeze) {
			return nil
		}
		r, err := a.Table.Read(rec)
		if err != nil {
			return err
		}
		if fields[fi].Type == 'M' {
			text := memoFieldText(a, r.Values[fi])
			newText, ok, err := editMemoBuffer(ctx, fields[fi].Name, text)
			firstPaint = true // memo left the screen
			if err != nil {
				return err
			}
			if ok {
				if err := writeMemoField(a, &r, fi, newText); err != nil {
					return err
				}
				_ = a.Table.Write(rec, r)
				_ = syncAllIndexes(ctx)
			}
			redraw()
			return nil
		}
		editBuf = strings.TrimRight(fieldDisplay(r.Values[fi], fields[fi]), " ")
		editing = true
		redraw()
		return nil
	}

	redraw()
	for {
		k, err := keys.ReadKey()
		if err != nil {
			if err == io.EOF {
				a.Cursor = rec
				_ = a.LoadCurrent()
				return nil
			}
			return fail(ctx, "%s", err.Error())
		}
		if editing {
			switch k {
			case term.KeyEsc:
				editing = false
				redraw()
			case term.KeyEnter:
				fi := cols[col]
				v, err := parseFieldInput(fields[fi], editBuf)
				if err == nil {
					r, _ := a.Table.Read(rec)
					r.Values[fi] = v
					_ = a.Table.Write(rec, r)
					_ = syncAllIndexes(ctx)
				}
				editing = false
				redraw()
			case term.KeyBackspace:
				if len(editBuf) > 0 {
					editBuf = editBuf[:len(editBuf)-1]
					redraw()
				}
			default:
				if term.IsInputByte(k) {
					w := browseColWidth(fields[cols[col]], opts.width)
					if len(editBuf) < w {
						editBuf += string(byte(k))
						redraw()
					}
				}
			}
			continue
		}
		switch k {
		case term.KeyEsc, term.KeyCtrlQ:
			a.Cursor = rec
			_ = a.LoadCurrent()
			return nil
		case term.KeyRight:
			if col < len(cols)-1 {
				col++
				if col >= hScroll+3 {
					hScroll++
				}
				redraw()
			}
		case term.KeyLeft:
			if col > 0 {
				col--
				if col < hScroll {
					hScroll = col
				}
				redraw()
			}
		case term.KeyDown:
			if rec < a.RecCount() {
				rec++
				redraw()
			}
		case term.KeyUp:
			if rec > 1 {
				rec--
				redraw()
			}
		case term.KeyCtrlU:
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
	up := strings.ToUpper(args)
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
	_ = up
	return o
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}
