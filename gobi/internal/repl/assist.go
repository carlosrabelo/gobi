package repl

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

type assistItem struct {
	Label  string
	Help   string
	Action string // dispatch id; empty = leave Assist
}

type assistCat struct {
	Name  string
	Items []assistItem
}

// assistMenu is the Assist category bar and submenus (037). Refused designers omitted.
var assistMenu = []assistCat{
	{Name: "Set Up", Items: []assistItem{
		{Label: "Database file", Help: "USE a table", Action: "use"},
		{Label: "Index file", Help: "SET INDEX TO", Action: "setindex"},
		{Label: "Default drive/directory", Help: "SET DEFAULT TO", Action: "setdefault"},
	}},
	{Name: "Create", Items: []assistItem{
		{Label: "Database file", Help: "CREATE a table", Action: "create"},
		{Label: "From file", Help: "CREATE FROM", Action: "createfrom"},
	}},
	{Name: "Update", Items: []assistItem{
		{Label: "Append", Help: "APPEND", Action: "append"},
		{Label: "Edit", Help: "EDIT", Action: "edit"},
		{Label: "Browse", Help: "BROWSE", Action: "browse"},
		{Label: "Replace", Help: "REPLACE", Action: "replace"},
		{Label: "Delete", Help: "DELETE", Action: "delete"},
		{Label: "Recall", Help: "RECALL", Action: "recall"},
		{Label: "Pack", Help: "PACK", Action: "pack"},
	}},
	{Name: "Position", Items: []assistItem{
		{Label: "Goto record", Help: "GOTO n", Action: "goto"},
		{Label: "Skip", Help: "SKIP n", Action: "skip"},
		{Label: "Locate", Help: "LOCATE FOR", Action: "locate"},
		{Label: "Continue", Help: "CONTINUE", Action: "continue"},
		{Label: "Seek", Help: "SEEK", Action: "seek"},
		{Label: "Find", Help: "FIND", Action: "find"},
		{Label: "Go top", Help: "GO TOP", Action: "gotop"},
		{Label: "Go bottom", Help: "GO BOTTOM", Action: "gobottom"},
	}},
	{Name: "Retrieve", Items: []assistItem{
		{Label: "List", Help: "LIST", Action: "list"},
		{Label: "Display", Help: "DISPLAY", Action: "display"},
		{Label: "Report", Help: "REPORT FORM", Action: "report"},
		{Label: "Label", Help: "LABEL FORM", Action: "label"},
		{Label: "Sum", Help: "SUM", Action: "sum"},
		{Label: "Average", Help: "AVERAGE", Action: "average"},
		{Label: "Count", Help: "COUNT", Action: "count"},
	}},
	{Name: "Organize", Items: []assistItem{
		{Label: "Index", Help: "INDEX ON", Action: "index"},
		{Label: "Sort", Help: "SORT", Action: "sort"},
		{Label: "Copy", Help: "COPY TO", Action: "copy"},
		{Label: "Copy structure", Help: "COPY STRUCTURE TO", Action: "copystruct"},
		{Label: "Join", Help: "JOIN", Action: "join"},
		{Label: "Total", Help: "TOTAL", Action: "total"},
	}},
	{Name: "Modify", Items: []assistItem{
		{Label: "Structure", Help: "MODIFY STRUCTURE", Action: "modstruct"},
		{Label: "Command file", Help: "MODIFY COMMAND", Action: "modcmd"},
	}},
	{Name: "Tools", Items: []assistItem{
		{Label: "Dir", Help: "DIR", Action: "dir"},
		{Label: "Display status", Help: "DISPLAY STATUS", Action: "status"},
		{Label: "Display memory", Help: "DISPLAY MEMORY", Action: "memory"},
		{Label: "Display structure", Help: "DISPLAY STRUCTURE", Action: "structure"},
		{Label: "Help", Help: "HELP", Action: "help"},
		{Label: "Run DOS / shell", Help: "RUN", Action: "run"},
	}},
	{Name: "Exit", Items: []assistItem{
		{Label: "Exit to dot prompt", Help: "Leave ASSIST", Action: "leave"},
		{Label: "Quit to DOS", Help: "QUIT", Action: "quit"},
		{Label: "Cancel", Help: "Leave ASSIST", Action: "leave"},
	}},
}

func (m *Mux) cmdASSIST(ctx *context.Context, cmd command.Command) error {
	if strings.TrimSpace(cmd.Args) != "" {
		return fail(ctx, "ASSIST takes no arguments")
	}
	ctx.AssistOpen = true
	ctx.StatusMode = StatusModeAssist
	defer func() {
		ctx.AssistOpen = false
		ctx.StatusMode = ""
		if scr := ctx.Screen; scr != nil {
			_ = scr.ClearTerminal(ctx.Out())
		}
		_ = applyStatusScroll(ctx)
	}()

	scr := ensureScreen(ctx)
	if ctx.Keys == nil {
		ctx.Keys = term.NewReader(ctx.Stdin)
	}
	keys := ctx.Keys
	cat := 0
	barScroll := 0
	inSub := false
	item := 0
	msg := "Esc=leave  Left/Right=menu  Down=open"

	drawAssistShell(ctx, scr, cat, barScroll, inSub, item, msg)
	if err := flushScreenClear(ctx); err != nil {
		return err
	}
	keys.DrainPendingLineEnds()

	for {
		drawAssistShell(ctx, scr, cat, barScroll, inSub, item, msg)
		if err := flushScreen(ctx); err != nil {
			return err
		}
		k, err := keys.ReadKey()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fail(ctx, "%s", err.Error())
		}
		switch {
		case k == term.KeyEsc || k == term.KeyCtrlQ:
			if inSub {
				inSub = false
				msg = "Esc=leave  Left/Right=menu  Down=open"
				continue
			}
			return nil
		case !inSub && k == term.KeyLeft:
			if cat > 0 {
				cat--
			}
			barScroll = assistEnsureCatVisible(scr.Cols, cat, barScroll)
		case !inSub && k == term.KeyRight:
			if cat < len(assistMenu)-1 {
				cat++
			}
			barScroll = assistEnsureCatVisible(scr.Cols, cat, barScroll)
		case !inSub && (k == term.KeyDown || k == term.KeyEnter):
			inSub = true
			item = 0
			msg = "Esc=back  Enter=run  Up/Down=item"
		case inSub && k == term.KeyUp:
			if item > 0 {
				item--
			}
		case inSub && k == term.KeyDown:
			if item < len(assistMenu[cat].Items)-1 {
				item++
			}
		case inSub && k == term.KeyEnter:
			it := assistMenu[cat].Items[item]
			leave, qerr, note := m.runAssistAction(ctx, keys, it.Action)
			if note != "" {
				msg = note
			}
			if qerr != nil {
				return qerr
			}
			if leave {
				return nil
			}
		case k >= 'a' && k <= 'z' || k >= 'A' && k <= 'Z':
			letter := byte(unicode.ToUpper(rune(k)))
			if inSub {
				item = assistJumpItem(cat, item, letter)
			} else {
				cat = assistJumpCat(cat, letter)
				barScroll = assistEnsureCatVisible(scr.Cols, cat, barScroll)
			}
		}
	}
}

func assistJumpCat(from int, letter byte) int {
	n := len(assistMenu)
	for i := 1; i <= n; i++ {
		j := (from + i) % n
		if assistMenu[j].Name != "" && byte(unicode.ToUpper(rune(assistMenu[j].Name[0]))) == letter {
			return j
		}
	}
	return from
}

func assistJumpItem(cat, from int, letter byte) int {
	items := assistMenu[cat].Items
	n := len(items)
	for i := 1; i <= n; i++ {
		j := (from + i) % n
		if items[j].Label != "" && byte(unicode.ToUpper(rune(items[j].Label[0]))) == letter {
			return j
		}
	}
	return from
}

// runAssistAction runs a menu action. leave means exit Assist; qerr is QUIT.
func (m *Mux) runAssistAction(ctx *context.Context, keys *term.Reader, action string) (leave bool, qerr error, note string) {
	switch action {
	case "leave":
		return true, nil, ""
	case "quit":
		return true, ErrQuit, ""
	case "use":
		name, ok, err := assistPrompt(ctx, keys, "Database file:")
		if err != nil {
			return false, err, ""
		}
		if !ok {
			return false, nil, "cancelled"
		}
		line := "USE"
		if strings.TrimSpace(name) != "" {
			line = "USE " + strings.TrimSpace(name)
		}
		return m.assistDispatch(ctx, line)
	case "setindex":
		name, ok, err := assistPrompt(ctx, keys, "Index file:")
		if err != nil {
			return false, err, ""
		}
		if !ok {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "SET INDEX TO "+strings.TrimSpace(name))
	case "setdefault":
		path, ok, err := assistPrompt(ctx, keys, "Default directory:")
		if err != nil {
			return false, err, ""
		}
		if !ok {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "SET DEFAULT TO "+strings.TrimSpace(path))
	case "create":
		name, ok, err := assistPrompt(ctx, keys, "New database:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(name) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "CREATE "+strings.TrimSpace(name))
	case "createfrom":
		name, ok, err := assistPrompt(ctx, keys, "New database:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(name) == "" {
			return false, nil, "cancelled"
		}
		from, ok, err := assistPrompt(ctx, keys, "FROM file:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(from) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "CREATE "+strings.TrimSpace(name)+" FROM "+strings.TrimSpace(from))
	case "append":
		return m.assistDispatch(ctx, "APPEND")
	case "edit":
		return m.assistDispatch(ctx, "EDIT")
	case "browse":
		return m.assistDispatch(ctx, "BROWSE")
	case "replace":
		body, ok, err := assistPrompt(ctx, keys, "REPLACE (field WITH expr [FOR …]):")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(body) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "REPLACE "+strings.TrimSpace(body))
	case "delete":
		return m.assistDispatch(ctx, "DELETE")
	case "recall":
		return m.assistDispatch(ctx, "RECALL")
	case "pack":
		return m.assistDispatch(ctx, "PACK")
	case "goto":
		n, ok, err := assistPrompt(ctx, keys, "Record number:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(n) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "GOTO "+strings.TrimSpace(n))
	case "skip":
		n, ok, err := assistPrompt(ctx, keys, "Skip records (default 1):")
		if err != nil {
			return false, err, ""
		}
		if !ok {
			return false, nil, "cancelled"
		}
		n = strings.TrimSpace(n)
		if n == "" {
			n = "1"
		}
		return m.assistDispatch(ctx, "SKIP "+n)
	case "locate":
		expr, ok, err := assistPrompt(ctx, keys, "LOCATE FOR:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(expr) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "LOCATE FOR "+strings.TrimSpace(expr))
	case "continue":
		return m.assistDispatch(ctx, "CONTINUE")
	case "seek":
		expr, ok, err := assistPrompt(ctx, keys, "SEEK expression:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(expr) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "SEEK "+strings.TrimSpace(expr))
	case "find":
		text, ok, err := assistPrompt(ctx, keys, "FIND string:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(text) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "FIND "+strings.TrimSpace(text))
	case "gotop":
		return m.assistDispatch(ctx, "GO TOP")
	case "gobottom":
		return m.assistDispatch(ctx, "GO BOTTOM")
	case "list":
		return m.assistDispatch(ctx, "LIST")
	case "display":
		return m.assistDispatch(ctx, "DISPLAY")
	case "report":
		name, ok, err := assistPrompt(ctx, keys, "REPORT FORM name:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(name) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "REPORT FORM "+strings.TrimSpace(name))
	case "label":
		name, ok, err := assistPrompt(ctx, keys, "LABEL FORM name:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(name) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "LABEL FORM "+strings.TrimSpace(name))
	case "sum":
		body, ok, err := assistPrompt(ctx, keys, "SUM expr [FOR …]:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(body) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "SUM "+strings.TrimSpace(body))
	case "average":
		body, ok, err := assistPrompt(ctx, keys, "AVERAGE expr [FOR …]:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(body) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "AVERAGE "+strings.TrimSpace(body))
	case "count":
		body, ok, err := assistPrompt(ctx, keys, "COUNT [FOR …] (empty=all):")
		if err != nil {
			return false, err, ""
		}
		if !ok {
			return false, nil, "cancelled"
		}
		body = strings.TrimSpace(body)
		if body == "" {
			return m.assistDispatch(ctx, "COUNT")
		}
		if strings.HasPrefix(strings.ToUpper(body), "FOR ") {
			return m.assistDispatch(ctx, "COUNT "+body)
		}
		return m.assistDispatch(ctx, "COUNT FOR "+body)
	case "index":
		expr, ok, err := assistPrompt(ctx, keys, "INDEX ON expression:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(expr) == "" {
			return false, nil, "cancelled"
		}
		to, ok, err := assistPrompt(ctx, keys, "TO index name:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(to) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "INDEX ON "+strings.TrimSpace(expr)+" TO "+strings.TrimSpace(to))
	case "sort":
		fields, ok, err := assistPrompt(ctx, keys, "SORT ON fields:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(fields) == "" {
			return false, nil, "cancelled"
		}
		to, ok, err := assistPrompt(ctx, keys, "TO file:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(to) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "SORT TO "+strings.TrimSpace(to)+" ON "+strings.TrimSpace(fields))
	case "copy":
		to, ok, err := assistPrompt(ctx, keys, "COPY TO file:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(to) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "COPY TO "+strings.TrimSpace(to))
	case "copystruct":
		to, ok, err := assistPrompt(ctx, keys, "COPY STRUCTURE TO file:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(to) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "COPY STRUCTURE TO "+strings.TrimSpace(to))
	case "join":
		body, ok, err := assistPrompt(ctx, keys, "JOIN WITH … TO … FOR …:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(body) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "JOIN "+strings.TrimSpace(body))
	case "total":
		body, ok, err := assistPrompt(ctx, keys, "TOTAL ON … TO …:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(body) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "TOTAL "+strings.TrimSpace(body))
	case "modstruct":
		return m.assistDispatch(ctx, "MODIFY STRUCTURE")
	case "modcmd":
		name, ok, err := assistPrompt(ctx, keys, "MODIFY COMMAND file:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(name) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "MODIFY COMMAND "+strings.TrimSpace(name))
	case "dir":
		return m.assistDispatch(ctx, "DIR")
	case "status":
		return m.assistDispatch(ctx, "DISPLAY STATUS")
	case "memory":
		return m.assistDispatch(ctx, "DISPLAY MEMORY")
	case "structure":
		return m.assistDispatch(ctx, "DISPLAY STRUCTURE")
	case "help":
		topic, ok, err := assistPrompt(ctx, keys, "HELP topic (empty=list):")
		if err != nil {
			return false, err, ""
		}
		if !ok {
			return false, nil, "cancelled"
		}
		topic = strings.TrimSpace(topic)
		if topic == "" {
			return m.assistDispatch(ctx, "HELP")
		}
		return m.assistDispatch(ctx, "HELP "+topic)
	case "run":
		text, ok, err := assistPrompt(ctx, keys, "RUN command:")
		if err != nil {
			return false, err, ""
		}
		if !ok || strings.TrimSpace(text) == "" {
			return false, nil, "cancelled"
		}
		return m.assistDispatch(ctx, "RUN "+strings.TrimSpace(text))
	default:
		return false, nil, "action pending: " + action
	}
}

func (m *Mux) assistDispatch(ctx *context.Context, line string) (leave bool, qerr error, note string) {
	// Line-mode and nested full-screen commands must not paint over Assist chrome.
	// Spec 037: they take over the screen; Assist redraws when they return.
	if scr := ctx.Screen; scr != nil {
		_ = scr.ClearTerminal(ctx.Out())
	}
	err := m.Dispatch(ctx, command.Split(line))
	if errors.Is(err, ErrQuit) {
		return true, err, ""
	}
	if err != nil {
		note = strings.TrimPrefix(err.Error(), "*** ")
		if len(note) > 70 {
			note = note[:70]
		}
		return false, nil, note
	}
	return false, nil, "ok"
}

// assistPrompt edits one line on the Assist screen. ok is false when Esc cancels.
func assistPrompt(ctx *context.Context, keys *term.Reader, label string) (text string, ok bool, err error) {
	scr := ensureScreen(ctx)
	var b strings.Builder
	for {
		scr.Clear()
		scr.WriteString(2, 0, label, false)
		line := b.String()
		if len(line) > scr.Cols {
			line = line[len(line)-scr.Cols:]
		}
		scr.WriteString(3, 0, line, ctx.Config.Intensity)
		scr.WriteString(scr.Rows-2, 0, "Enter=accept  Esc=cancel", false)
		scr.WriteString(scr.Rows-1, 0, assistStatusLine(ctx, scr.Cols), false)
		if err := flushScreenClear(ctx); err != nil {
			return "", false, err
		}
		k, err := keys.ReadKey()
		if err != nil {
			if err == io.EOF {
				return b.String(), true, nil
			}
			return "", false, err
		}
		switch {
		case k == term.KeyEsc || k == term.KeyCtrlQ:
			return "", false, nil
		case k == term.KeyEnter || k == '\n':
			return b.String(), true, nil
		case k == term.KeyBackspace:
			s := b.String()
			if len(s) > 0 {
				b.Reset()
				b.WriteString(s[:len(s)-1])
			}
		case term.IsInputByte(k) && b.Len() < 80:
			b.WriteByte(byte(k))
		}
	}
}

func assistEnsureCatVisible(cols, cat, scroll int) int {
	for {
		if cat < scroll {
			scroll = cat
		}
		col := 0
		visibleEnd := scroll
		for i := scroll; i < len(assistMenu); i++ {
			need := len(assistMenu[i].Name)
			if i > scroll {
				need += 2
			}
			if col+need > cols {
				break
			}
			col += need
			visibleEnd = i
		}
		if cat <= visibleEnd {
			return scroll
		}
		scroll++
		if scroll > cat {
			return cat
		}
	}
}

func drawAssistShell(ctx *context.Context, scr *term.Screen, cat, barScroll int, inSub bool, item int, message string) {
	// Blank rows via spaces (dirty present) instead of Screen.Clear + PresentClear.
	for r := 0; r < scr.Rows; r++ {
		scr.WriteString(r, 0, strings.Repeat(" ", scr.Cols), false)
	}
	rev := ctx.Config.Intensity
	drawAssistBar(scr, cat, barScroll, rev)
	if inSub {
		drawAssistSubmenu(scr, cat, item, rev)
	}
	if message == "" {
		message = "Esc=leave  Left/Right=menu  Down=open"
	}
	prev := ctx.Message
	ctx.Message = message
	ctx.StatusMode = StatusModeAssist
	paintStatusBand(ctx, StatusModeAssist, true) // ASSIST always shows the bar
	ctx.Message = prev
}

func drawAssistBar(scr *term.Screen, cat, scroll int, reverseActive bool) {
	col := 0
	for i := scroll; i < len(assistMenu); i++ {
		label := assistMenu[i].Name
		need := len(label)
		if i > scroll {
			need += 2
		}
		if col+need > scr.Cols {
			break
		}
		if i > scroll {
			scr.WriteString(0, col, "  ", false)
			col += 2
		}
		scr.WriteString(0, col, label, reverseActive && i == cat)
		col += len(label)
	}
}

func drawAssistSubmenu(scr *term.Screen, cat, item int, reverseActive bool) {
	items := assistMenu[cat].Items
	work := statusWorkRows(scr.Rows)
	maxRows := work - 2 // below menu bar, above item-help row
	if maxRows < 1 {
		maxRows = 1
	}
	start := 0
	if item >= maxRows {
		start = item - maxRows + 1
	}
	for i := start; i < len(items) && i-start < maxRows; i++ {
		row := 1 + (i - start)
		label := "  " + items[i].Label
		if len(label) > scr.Cols {
			label = label[:scr.Cols]
		}
		scr.WriteString(row, 0, label, reverseActive && i == item)
	}
	if item >= 0 && item < len(items) && items[item].Help != "" {
		helpRow := work - 1
		if helpRow > 1 {
			h := items[item].Help
			if len(h) > scr.Cols {
				h = h[:scr.Cols]
			}
			scr.WriteString(helpRow, 0, h, false)
		}
	}
}

func assistStatusLine(ctx *context.Context, cols int) string {
	a := ctx.Active()
	alias := "(none)"
	if a.HasTable() && a.Alias != "" {
		alias = a.Alias
	}
	rec := 0
	if a.Landed {
		rec = a.Cursor
	}
	line := fmt.Sprintf("%s Rec:%d", alias, rec)
	if len(line) > cols {
		return line[:cols]
	}
	return line
}

// assistCreateHasReport reports whether Create offers Report (must be false).
func assistCreateHasReport() bool {
	for _, it := range assistMenu[1].Items {
		if strings.EqualFold(it.Label, "Report") || strings.Contains(strings.ToLower(it.Label), "report") {
			return true
		}
	}
	return false
}
