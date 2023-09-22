package repl

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxHistorySize  = 500
	historyFileName = ".gobi_history"
)

// History manages a ring buffer of past commands with file persistence.
type History struct {
	items   []string
	maxSize int
	pos     int
	loaded  bool
}

// NewHistory creates an empty history buffer.
func NewHistory(maxSize int) *History {
	return &History{
		items:   make([]string, 0, maxSize),
		maxSize: maxSize,
		pos:     0,
	}
}

// Add appends a command to the history, skipping empty duplicates.
func (h *History) Add(cmd string) {
	if cmd == "" {
		return
	}
	if len(h.items) > 0 && h.items[len(h.items)-1] == cmd {
		return
	}
	if len(h.items) >= h.maxSize {
		h.items = h.items[1:]
	}
	h.items = append(h.items, cmd)
	h.pos = len(h.items)
}

// Prev returns the previous command in history and moves the cursor back.
func (h *History) Prev() (string, bool) {
	if len(h.items) == 0 || h.pos <= 0 {
		return "", false
	}
	h.pos--
	return h.items[h.pos], true
}

// Next returns the next command in history and moves the cursor forward.
func (h *History) Next() (string, bool) {
	if h.pos >= len(h.items) {
		return "", false
	}
	h.pos++
	if h.pos >= len(h.items) {
		return "", false
	}
	return h.items[h.pos], true
}

// Reset resets the navigation position to the end.
func (h *History) Reset() {
	h.pos = len(h.items)
}

// All returns all history items.
func (h *History) All() []string {
	return h.items
}

// Load reads history from a file in the user's home directory.
func (h *History) Load() error {
	if h.loaded {
		return nil
	}
	h.loaded = true

	path, err := historyFilePath()
	if err != nil {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r\n")
		if line != "" {
			if len(h.items) >= h.maxSize {
				h.items = h.items[1:]
			}
			h.items = append(h.items, line)
		}
	}
	h.pos = len(h.items)

	return scanner.Err()
}

// Save writes history to a file in the user's home directory.
func (h *History) Save() error {
	path, err := historyFilePath()
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, item := range h.items {
		fmt.Fprintln(f, item)
	}

	return nil
}

func historyFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, historyFileName), nil
}
