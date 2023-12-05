package repl

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const defaultHistoryCap = 500

// History stores prompt lines, at most Cap entries, with load/save under Home.
type History struct {
	Home  string
	Cap   int
	lines []string
	pos   int // index into lines; len(lines) means after the newest entry
}

// NewHistory returns an empty history with cap 500 for the given home directory.
func NewHistory(home string) *History {
	return &History{Home: home, Cap: defaultHistoryCap, pos: 0}
}

// Len returns how many lines are stored.
func (h *History) Len() int {
	if h == nil {
		return 0
	}
	return len(h.lines)
}

// Lines returns a copy of stored lines oldest-first.
func (h *History) Lines() []string {
	if h == nil {
		return nil
	}
	out := make([]string, len(h.lines))
	copy(out, h.lines)
	return out
}

// SetCap updates the capacity and drops oldest lines past Cap.
func (h *History) SetCap(n int) {
	if h == nil {
		return
	}
	h.Cap = n
	if h.Cap > 0 && len(h.lines) > h.Cap {
		h.lines = h.lines[len(h.lines)-h.Cap:]
	}
	h.pos = len(h.lines)
}

// Add stores line when non-empty and not equal to the newest entry.
func (h *History) Add(line string) {
	if h == nil || line == "" {
		return
	}
	if len(h.lines) > 0 && h.lines[len(h.lines)-1] == line {
		h.pos = len(h.lines)
		return
	}
	h.lines = append(h.lines, line)
	if h.Cap > 0 && len(h.lines) > h.Cap {
		h.lines = h.lines[len(h.lines)-h.Cap:]
	}
	h.pos = len(h.lines)
}

// Prev walks toward older entries. The walk starts after the newest entry.
func (h *History) Prev() (string, bool) {
	if h == nil || len(h.lines) == 0 || h.pos == 0 {
		return "", false
	}
	h.pos--
	return h.lines[h.pos], true
}

// Next walks toward newer entries.
func (h *History) Next() (string, bool) {
	if h == nil || h.pos >= len(h.lines) {
		return "", false
	}
	h.pos++
	if h.pos >= len(h.lines) {
		return "", false
	}
	return h.lines[h.pos], true
}

// path is $HOME/.gobi_history.
func (h *History) path() string {
	return filepath.Join(h.Home, ".gobi_history")
}

// Load reads history from disk. A missing file yields an empty history.
func (h *History) Load() error {
	if h == nil || h.Home == "" {
		return nil
	}
	f, err := os.Open(h.path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	h.lines = h.lines[:0]
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		h.lines = append(h.lines, line)
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if h.Cap > 0 && len(h.lines) > h.Cap {
		h.lines = h.lines[len(h.lines)-h.Cap:]
	}
	h.pos = len(h.lines)
	return nil
}

// Save writes history to disk, one command per line.
func (h *History) Save() error {
	if h == nil || h.Home == "" {
		return nil
	}
	if err := os.MkdirAll(h.Home, 0o755); err != nil {
		return err
	}
	var b strings.Builder
	for _, line := range h.lines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return os.WriteFile(h.path(), []byte(b.String()), 0o644)
}
