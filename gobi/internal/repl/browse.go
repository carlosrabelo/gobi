package repl

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
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
	defer func() { _ = scr.ClearTerminal(ctx.Out()) }()
	keys := term.NewReader(ctx.Stdin)
	rec := a.Cursor
	if rec < 1 {
		rec = 1
	}
	col := 0
	hScroll := 0
	editing := false
	editBuf := ""

	status := func() {
		name := fields[cols[col]].Name
		mark := " "
		if editing {
			mark = "*"
		}
		line := fmt.Sprintf("Rec:%d %s%s", rec, name, mark)
		scr.WriteString(scr.Rows-1, 0, padRight(line, scr.Cols), false)
	}

	redraw := func() {
		scr.Clear()
		_ = opts.lock
		startCol := hScroll
		if startCol >= len(cols) {
			startCol = len(cols) - 1
		}
		x := 0
		for ci := startCol; ci < len(cols) && x < scr.Cols; ci++ {
			fi := cols[ci]
			w := int(fields[fi].Length)
			if opts.width > 0 && w > opts.width {
				w = opts.width
			}
			if w < 1 {
				w = 1
			}
			hdr := fields[fi].Name
			if len(hdr) > w {
				hdr = hdr[:w]
			}
			scr.WriteString(0, x, padRight(hdr, w), false)
			x += w + 1
		}
		maxRows := scr.Rows - 2
		for row := 0; row < maxRows; row++ {
			rn := rec - maxRows/2 + row
			if rn < 1 || rn > a.RecCount() {
				continue
			}
			_ = a.Table // read without moving cursor permanently
			r, err := a.Table.Read(rn)
			if err != nil {
				continue
			}
			x := 0
			for ci := startCol; ci < len(cols) && x < scr.Cols; ci++ {
				fi := cols[ci]
				w := int(fields[fi].Length)
				if opts.width > 0 && w > opts.width {
					w = opts.width
				}
				text := fieldDisplay(r.Values[fi], fields[fi])
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
		status()
		_ = flushScreenClear(ctx)
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
				if k >= 32 && k < 127 {
					w := int(fields[cols[col]].Length)
					if opts.width > 0 && w > opts.width {
						w = opts.width
					}
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
			if opts.freeze != "" && !strings.EqualFold(fields[cols[col]].Name, opts.freeze) {
				continue
			}
			r, _ := a.Table.Read(rec)
			editBuf = strings.TrimRight(fieldDisplay(r.Values[cols[col]], fields[cols[col]]), " ")
			editing = true
			redraw()
		default:
			if k >= 32 && k < 127 {
				if opts.noAppend && false {
					_ = opts
				}
				editBuf = string(byte(k))
				editing = true
				redraw()
			}
		}
	}
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
