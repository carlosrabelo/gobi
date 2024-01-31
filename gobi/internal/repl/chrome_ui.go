package repl

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/carlosrabelo/gobi/gobi/internal/config"
	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/hostterm"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// statusChromeRows reserves status + spacer + message (classic 22/23/24 of 25).
const statusChromeRows = 3

// defaultDotMessage is the prompt hint on the message line (Gobi branding).
const defaultDotMessage = "Enter a Gobi command."

// Status mode labels (segment 1 of the bar).
const (
	StatusModeCommand = "Command Line"
	StatusModeAssist  = "ASSIST"
	StatusModeEdit    = "EDIT"
	StatusModeAppend  = "APPEND"
	StatusModeBrowse  = "BROWSE"
	StatusModeHelp    = "Help"
	StatusModeModify  = "MODIFY"
)

func paintChrome(ctx *context.Context) {
	if ctx.AssistOpen {
		return
	}
	scr := ctx.Screen
	if scr == nil || scr.Rows < 2 || scr.Cols < 8 {
		return
	}
	if ctx.Config.Status {
		paintStatusBand(ctx, statusModeOr(ctx, StatusModeCommand), true)
		return
	}
	a := ctx.Active()
	if ctx.Config.Scoreboard {
		var parts []string
		if a.Landed && a.Record.Deleted {
			parts = append(parts, "Del")
		}
		if ctx.InsertMode {
			parts = append(parts, "Ins")
		}
		if hostCapsOn() {
			parts = append(parts, "Caps")
		}
		if hostNumOn() {
			parts = append(parts, "Num")
		}
		text := strings.Join(parts, " ")
		if text == "" {
			return
		}
		col := scr.Cols - len(text)
		if col < 0 {
			col = 0
		}
		scr.WriteString(0, col, text, false)
	}
}

func statusModeOr(ctx *context.Context, fallback string) string {
	if ctx.StatusMode != "" {
		return ctx.StatusMode
	}
	return fallback
}

// paintStatusBand draws the III Plus segmented status bar and message line.
// force draws even when SET STATUS is OFF (ASSIST, EDIT, BROWSE always show it).
func paintStatusBand(ctx *context.Context, mode string, force bool) {
	if !force && (ctx.Config == nil || !ctx.Config.Status) {
		return
	}
	scr := ensureScreen(ctx)
	statusRow, msgRow, ok := statusLayout(scr.Rows)
	if !ok {
		return
	}
	rev := ctx.Config.Intensity
	runes := []rune(buildStatusLine(ctx, mode, scr.Cols))
	for c := 0; c < scr.Cols; c++ {
		ch := ' '
		if c < len(runes) {
			ch = runes[c]
		}
		scr.Put(statusRow, c, ch, rev)
	}
	if msgRow > statusRow+1 {
		for c := 0; c < scr.Cols; c++ {
			scr.Put(statusRow+1, c, ' ', false)
		}
	}
	msg := strings.TrimSpace(ctx.Message)
	if msg == "" {
		msg = defaultDotMessage
	}
	if len(msg) > scr.Cols {
		msg = msg[:scr.Cols]
	}
	pad := (scr.Cols - len(msg)) / 2
	if pad < 0 {
		pad = 0
	}
	for c := 0; c < scr.Cols; c++ {
		scr.Put(msgRow, c, ' ', false)
	}
	scr.WriteString(msgRow, pad, msg, false)
}

// statusWorkRows returns how many top rows are free for content when the band is shown.
func statusWorkRows(rows int) int {
	statusRow, _, ok := statusLayout(rows)
	if !ok {
		return rows
	}
	return statusRow
}

// statusLayout returns 0-based status and message rows (III Plus: 22 and 24 on 25 lines).
func statusLayout(rows int) (statusRow, msgRow int, ok bool) {
	if rows < statusChromeRows+1 {
		return 0, 0, false
	}
	statusRow = rows - statusChromeRows
	msgRow = rows - 1
	return statusRow, msgRow, true
}

// buildStatusLine paints the classic segmented bar; widths adapt to cols.
// Separators are term.StatusSep (║ by default, │ if reassigned).
func buildStatusLine(ctx *context.Context, mode string, cols int) string {
	a := ctx.Active()
	if mode == "" {
		mode = StatusModeCommand
	}
	drive := "<" + statusDrive(ctx) + ">"
	name := ""
	rec := "Rec: None"
	if a != nil && a.HasTable() {
		name = a.Alias
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(a.Path), filepath.Ext(a.Path))
		}
		name = strings.ToUpper(name)
		total := a.RecCount()
		cur := a.Cursor
		if !a.Landed && cur > total {
			rec = fmt.Sprintf("Rec: EOF/%d", total)
		} else if a.Landed || cur > 0 {
			if cur < 1 {
				cur = 1
			}
			rec = fmt.Sprintf("Rec: %d/%d", cur, total)
		} else {
			rec = fmt.Sprintf("Rec: 0/%d", total)
		}
	}
	// Right segment: only live indicators (III Plus SET STATUS / SCOREBOARD).
	// Blank when none apply — never a version string here (use VERSION()).
	var flags []string
	if a != nil && a.Landed && a.Record.Deleted {
		flags = append(flags, "Del")
	}
	if ctx.InsertMode {
		flags = append(flags, "Ins")
	}
	if hostCapsOn() {
		flags = append(flags, "Caps")
	}
	if hostNumOn() {
		flags = append(flags, "Num")
	}
	flag := strings.Join(flags, " ")

	sep := string(term.StatusSep)
	modeW, driveW, recW, flagW := 12, 4, 12, 8
	if cols < 60 {
		modeW, driveW, recW, flagW = 7, 4, 10, 4
		if mode == StatusModeCommand {
			mode = "Command"
		}
	}
	if cols < 48 {
		modeW, driveW, recW, flagW = 3, 4, 9, 3
		if mode == StatusModeCommand || mode == "Command" {
			mode = "Cmd"
		}
	}
	overhead := len([]rune(sep))*5 + modeW + driveW + recW + flagW
	nameW := cols - overhead
	if nameW < 4 {
		nameW = 4
	}

	mode = fitStatusField(mode, modeW)
	drive = fitStatusField(drive, driveW)
	name = fitStatusField(name, nameW)
	rec = fitStatusField(rec, recW)
	flag = fitStatusField(flag, flagW)

	line := sep + mode + sep + drive + sep + name + sep + rec + sep + flag + sep
	runes := []rune(line)
	if len(runes) < cols {
		line += strings.Repeat(" ", cols-len(runes))
		runes = []rune(line)
	}
	if len(runes) > cols {
		line = string(runes[:cols])
	}
	return line
}

func fitStatusField(s string, width int) string {
	var b strings.Builder
	for _, r := range s {
		if r == 0 {
			continue
		}
		if r < unicode.MaxASCII && !unicode.IsPrint(r) && r != ' ' {
			continue
		}
		b.WriteRune(r)
	}
	s = b.String()
	runes := []rune(s)
	if len(runes) > width {
		return string(runes[:width])
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func statusDrive(ctx *context.Context) string {
	d := strings.TrimSpace(ctx.Config.Default)
	if len(d) >= 2 && d[1] == ':' {
		// Windows-style volume (C:).
		return strings.ToUpper(d[:2])
	}
	// POSIX hosts: no fake DOS drive — show a short path tag.
	switch {
	case d == "" || d == ".":
		return "./"
	case d == "/":
		return "/"
	case strings.HasPrefix(d, "~/"):
		return "~/"
	default:
		return "./"
	}
}

func clearScreenRow(ctx *context.Context, row int) {
	scr := ctx.Screen
	if scr == nil {
		return
	}
	scr.WriteString(row, 0, strings.Repeat(" ", scr.Cols), false)
}

func flushScreen(ctx *context.Context) error {
	paintChrome(ctx)
	if err := applyStatusScroll(ctx); err != nil {
		return err
	}
	return ensureScreen(ctx).Present(ctx.Out())
}

func flushScreenClear(ctx *context.Context) error {
	paintChrome(ctx)
	if err := ensureScreen(ctx).PresentClear(ctx.Out()); err != nil {
		return err
	}
	return applyStatusScroll(ctx)
}

// applyStatusScroll keeps the dot prompt above the status chrome (DECSTBM).
func applyStatusScroll(ctx *context.Context) error {
	scr := ctx.Screen
	if scr == nil {
		return nil
	}
	out := ctx.Out()
	if ctx.AssistOpen || !ctx.Config.Status {
		return hostterm.ResetScrollRegion(out)
	}
	statusRow, _, ok := statusLayout(scr.Rows)
	if !ok || statusRow < 1 {
		return hostterm.ResetScrollRegion(out)
	}
	return hostterm.SetScrollRegion(out, 1, statusRow)
}

// leaveFullScreen wipes full-screen chrome and restores STATUS scroll if needed.
func leaveFullScreen(ctx *context.Context) error {
	ctx.StatusMode = ""
	scr := ensureScreen(ctx)
	if err := scr.ClearTerminal(ctx.Out()); err != nil {
		return err
	}
	if ctx.Config.Status && !ctx.AssistOpen {
		paintChrome(ctx)
		if err := scr.Present(ctx.Out()); err != nil {
			return err
		}
	}
	return applyStatusScroll(ctx)
}

// showSessionChrome paints STATUS/scoreboard without clearing the work area.
func showSessionChrome(ctx *context.Context, homeCursor bool) {
	if ctx.AssistOpen || (!ctx.Config.Status && !ctx.Config.Scoreboard) {
		return
	}
	scr := ensureScreen(ctx)
	paintChrome(ctx)
	_ = scr.Present(ctx.Out())
	_ = applyStatusScroll(ctx)
	if homeCursor && ctx.Config.Status {
		caps := hostterm.CapsOf(ctx.Stdout)
		_ = hostterm.MoveCursor(ctx.Out(), caps, 0, 0)
	}
}

// placeDotPrompt moves the cursor into the work pane above the status band so
// ". " is never painted on the chrome (and after *** errors sits on a fresh line).
func placeDotPrompt(ctx *context.Context) {
	if ctx.AssistOpen || ctx.Config == nil || !ctx.Config.Status {
		return
	}
	scr := ensureScreen(ctx)
	statusRow, _, ok := statusLayout(scr.Rows)
	if !ok || statusRow < 1 {
		return
	}
	caps := hostterm.CapsOf(ctx.Stdout)
	_ = hostterm.MoveCursor(ctx.Out(), caps, statusRow-1, 0)
	fmt.Fprint(ctx.Out(), "\r\n")
}

// repaintAfterResize rebuilds chrome for the new host size (AUTO).
func repaintAfterResize(ctx *context.Context) {
	if ctx.AssistOpen || ctx.Config == nil || ctx.Config.Screen != config.ScreenAuto {
		return
	}
	scr := ensureScreen(ctx)
	scr.Clear()
	paintChrome(ctx)
	_ = scr.PresentClear(ctx.Out())
	_ = applyStatusScroll(ctx)
	if ctx.Config.Status {
		caps := hostterm.CapsOf(ctx.Stdout)
		statusRow, _, ok := statusLayout(scr.Rows)
		row := 0
		if ok && statusRow > 1 {
			row = statusRow - 1
		}
		_ = hostterm.MoveCursor(ctx.Out(), caps, row, 0)
		fmt.Fprint(ctx.Out(), ". ")
	}
}

func menuReminder(ctx *context.Context) string {
	if !ctx.Config.Menus {
		return ""
	}
	return "^W save  Esc abort  Tab next"
}
