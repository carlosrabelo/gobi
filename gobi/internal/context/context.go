package context

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/config"
)

const (
	// AreaCount is the number of work areas in a session.
	AreaCount = 10
)

// Context is the session: work areas, streams, and config.
type Context struct {
	Config     *config.Config
	Areas      [AreaCount + 1]*Area // index 1 through 10
	ActiveArea int

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Alternate *os.File // SET ALTERNATE file; closed by CLOSE ALTERNATE / CLOSE ALL
	Procedure bool     // procedure file open (stub until 029)

	openFiles int

	stdinReader *bufio.Reader
	stdinSource io.Reader
}

// New returns a session with areas 1–10, active area 1, and process streams.
func New() *Context {
	c := &Context{
		Config:     config.New(),
		ActiveArea: 1,
		Stdin:      os.Stdin,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}
	for i := 1; i <= AreaCount; i++ {
		c.Areas[i] = &Area{}
	}
	return c
}

// Active returns the active work area.
func (c *Context) Active() *Area {
	return c.Areas[c.ActiveArea]
}

// Select switches the active area. name is "1"–"10" or "A"–"J", case-insensitive.
func (c *Context) Select(name string) error {
	n, err := parseArea(name)
	if err != nil {
		return err
	}
	c.ActiveArea = n
	return nil
}

func parseArea(name string) (int, error) {
	s := strings.TrimSpace(name)
	if s == "" {
		return 0, fmt.Errorf("invalid work area: %q", name)
	}
	if n, err := strconv.Atoi(s); err == nil {
		if n < 1 || n > AreaCount {
			return 0, fmt.Errorf("invalid work area: %s", s)
		}
		return n, nil
	}
	if len(s) != 1 {
		return 0, fmt.Errorf("invalid work area: %s", s)
	}
	letter := s[0]
	if letter >= 'a' && letter <= 'j' {
		letter -= 'a' - 'A'
	}
	if letter < 'A' || letter > 'J' {
		return 0, fmt.Errorf("invalid work area: %s", s)
	}
	return int(letter-'A') + 1, nil
}

// StdinReader returns a buffered reader over Stdin. Replacing Stdin rebuilds it.
func (c *Context) StdinReader() *bufio.Reader {
	if c.stdinReader == nil || c.stdinSource != c.Stdin {
		c.stdinReader = bufio.NewReader(c.Stdin)
		c.stdinSource = c.Stdin
	}
	return c.stdinReader
}

// Resolve joins a relative name to DEFAULT, then walks PATH entries when missing.
func (c *Context) Resolve(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty file name")
	}
	if filepath.IsAbs(name) {
		return name, nil
	}
	def := c.Config.Default
	if def == "" {
		def = "."
	}
	first := filepath.Join(def, name)
	if fileExists(first) {
		return first, nil
	}
	for _, entry := range pathEntries(c.Config.Path) {
		candidate := filepath.Join(entry, name)
		if fileExists(candidate) {
			return candidate, nil
		}
	}
	return first, nil
}

func pathEntries(path string) []string {
	if path == "" {
		return nil
	}
	path = strings.ReplaceAll(path, ";", ",")
	parts := strings.Split(path, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
