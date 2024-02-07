package repl

import (
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/hostterm"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// applySessionColors syncs the cell buffer with SET COLOR ON/TO and host caps.
func applySessionColors(ctx *context.Context) {
	if ctx == nil {
		return
	}
	scr := ensureScreen(ctx)
	stdFG, stdBG, enhFG, enhBG := parseColorTO(ctx.Config.Color)
	scr.StandardFG, scr.StandardBG = stdFG, stdBG
	scr.EnhancedFG, scr.EnhancedBG = enhFG, enhBG
	caps := hostterm.CapsOf(ctx.Stdout)
	scr.ColorActive = ctx.Config.ColorOn && caps.Color
	scr.RefreshColorAttrs()
}

// parseColorTO parses III Plus SET COLOR TO text into standard and enhanced pairs.
// Form: [[standard][,[enhanced][,[border][,[background]]]]] with pairs like W/N, GR+/B.
// Border/background slots are accepted and ignored for cell paint (host chrome).
func parseColorTO(text string) (stdFG, stdBG, enhFG, enhBG int) {
	stdFG, stdBG = term.ColorDefault, term.ColorDefault
	enhFG, enhBG = term.ColorDefault, term.ColorDefault
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	parts := splitColorPairs(text)
	if len(parts) > 0 {
		stdFG, stdBG = parseColorPair(parts[0])
	}
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		enhFG, enhBG = parseColorPair(parts[1])
	} else if len(parts) >= 1 && strings.TrimSpace(parts[0]) != "" {
		// One pair: enhanced is the swapped colors (III Plus default).
		enhFG, enhBG = stdBG, stdFG
		if enhFG == term.ColorDefault {
			enhFG = 0
		}
		if enhBG == term.ColorDefault {
			enhBG = 7
		}
	}
	return
}

func splitColorPairs(text string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(text); i++ {
		if i == len(text) || text[i] == ',' {
			out = append(out, strings.TrimSpace(text[start:i]))
			start = i + 1
		}
	}
	return out
}

func parseColorPair(s string) (fg, bg int) {
	fg, bg = term.ColorDefault, term.ColorDefault
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	parts := strings.SplitN(s, "/", 2)
	if f, ok := parseColorName(parts[0]); ok {
		fg = f
	}
	if len(parts) == 2 {
		if b, ok := parseColorName(parts[1]); ok {
			bg = b
		}
	}
	return
}

func parseColorName(s string) (int, bool) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return 0, false
	}
	bright := false
	if strings.HasSuffix(s, "+") {
		bright = true
		s = strings.TrimSuffix(s, "+")
	}
	var base int
	switch s {
	case "N", "BLACK":
		base = 0
	case "B", "BLUE":
		base = 1
	case "G", "GREEN":
		base = 2
	case "BG", "CYAN":
		base = 3
	case "R", "RED":
		base = 4
	case "RB", "MAGENTA":
		base = 5
	case "GR", "BROWN", "YELLOW":
		base = 6
	case "W", "WHITE":
		base = 7
	default:
		return 0, false
	}
	if bright {
		base += 8
	}
	return base, true
}
