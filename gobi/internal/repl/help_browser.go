package repl

import (
	"io"
	"strings"
	"unicode"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// helpBrowser is the full-screen HELP UI (spec 035.8). topicName empty = catalog.
func (m *Mux) helpBrowser(ctx *context.Context, topicName string) error {
	scr := ensureScreen(ctx)
	ctx.StatusMode = StatusModeHelp
	keys := term.NewReader(ctx.Stdin)
	prevKeys := ctx.Keys
	ctx.Keys = keys
	defer func() {
		ctx.Keys = prevKeys
		ctx.StatusMode = ""
	}()

	restore, err := term.EnterRaw(ctx.Stdin)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	defer restore()

	modeCatalog := topicName == ""
	current := topicName
	scroll := 0
	lines := helpBrowserLines(ctx, modeCatalog, current)

	// One clear on entry.
	if err := flushScreenClear(ctx); err != nil {
		return err
	}

	for {
		drawHelpBrowser(ctx, scr, modeCatalog, current, lines, scroll)
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
		page := scr.Rows - 4
		if page < 1 {
			page = 1
		}
		switch {
		case k == term.KeyEsc || k == term.KeyCtrlQ:
			if !modeCatalog {
				modeCatalog = true
				current = ""
				scroll = 0
				lines = helpBrowserLines(ctx, true, "")
				continue
			}
			_ = leaveFullScreen(ctx)
			return nil
		case k == term.KeyUp:
			if scroll > 0 {
				scroll--
			}
		case k == term.KeyDown:
			if scroll+page < len(lines) {
				scroll++
			}
		case k == term.KeyPgUp:
			scroll -= page
			if scroll < 0 {
				scroll = 0
			}
		case k == term.KeyPgDn:
			scroll += page
			if scroll > len(lines)-1 {
				scroll = len(lines) - 1
			}
			if scroll < 0 {
				scroll = 0
			}
		case k == term.KeyEnter && modeCatalog:
			name := catalogTopicAt(lines, scroll)
			if name == "" {
				continue
			}
			if _, ok := help.Lookup(name); ok {
				modeCatalog = false
				current = name
				scroll = 0
				lines = helpBrowserLines(ctx, false, current)
			}
		case k >= 'a' && k <= 'z' || k >= 'A' && k <= 'Z':
			letter := byte(unicode.ToUpper(rune(k)))
			scroll = helpJumpLetter(lines, scroll, letter)
		}
	}
}

func helpBrowserLines(ctx *context.Context, catalog bool, topic string) []string {
	width := helpWidth(ctx)
	var text string
	if catalog {
		text = help.FormatCatalog(width)
	} else if t, ok := help.Lookup(topic); ok {
		text = help.FormatTopic(t, width)
	}
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func drawHelpBrowser(ctx *context.Context, scr *term.Screen, catalog bool, topic string, lines []string, scroll int) {
	// Dirty-cell friendly: rewrite rows without Screen.Clear (which blanks pending
	// but we still only Present diffs). Clear pending by writing spaces per row.
	for r := 0; r < scr.Rows; r++ {
		scr.WriteString(r, 0, strings.Repeat(" ", scr.Cols), false)
	}
	title := "Help  Catalog"
	if !catalog {
		title = "Help  " + topic
	}
	if len(title) > scr.Cols {
		title = title[:scr.Cols]
	}
	scr.WriteString(0, 0, title, ctx.Config.Intensity)
	work := statusWorkRows(scr.Rows)
	bodyRows := work - 1
	if bodyRows < 1 {
		bodyRows = 1
	}
	for i := 0; i < bodyRows; i++ {
		li := scroll + i
		if li >= len(lines) {
			break
		}
		line := lines[li]
		if len(line) > scr.Cols {
			line = line[:scr.Cols]
		}
		scr.WriteString(1+i, 0, line, false)
	}
	prev := ctx.Message
	ctx.Message = "Up/Down scroll  Enter open  Esc back/exit"
	paintStatusBand(ctx, StatusModeHelp, true)
	ctx.Message = prev
}

func catalogTopicAt(lines []string, scroll int) string {
	if scroll < 0 || scroll >= len(lines) {
		return ""
	}
	line := strings.TrimSpace(lines[scroll])
	if line == "" || !strings.HasPrefix(lines[scroll], "  ") {
		return ""
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return strings.ToUpper(fields[0])
}

func helpJumpLetter(lines []string, from int, letter byte) int {
	n := len(lines)
	for i := 1; i <= n; i++ {
		idx := (from + i) % n
		name := catalogTopicAt(lines, idx)
		if name != "" && name[0] == letter {
			return idx
		}
	}
	return from
}
