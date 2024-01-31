package context

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/carlosrabelo/gobi/gobi/internal/config"
	"github.com/carlosrabelo/gobi/gobi/internal/symbols"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
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
	Symbols    *symbols.Registry
	Screen     *term.Screen
	Keys       *term.Reader

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Alternate   *os.File // SET ALTERNATE file; closed by CLOSE ALTERNATE / CLOSE ALL
	PrinterFile *os.File // SET PRINTER TO file; nil = stdout print sink
	Procedure   bool     // procedure file open

	// Locate holds the last LOCATE FOR/WHILE text for CONTINUE (013).
	LocateActive bool
	LocateFOR    string
	LocateWHILE  string

	// PendingTO/PendingVals hold COUNT/SUM/AVERAGE TO until 014 stores them.
	PendingTO   []string
	PendingVals []float64

	// Procedure / PUBLIC / PRIVATE state (029).
	Publics     map[string]bool
	Privates    map[string]bool
	ParamLocals map[string]bool
	PendingArgs []expr.Value

	// ON ERROR / ERROR() / MESSAGE() (031).
	OnError     string
	OnEscape    string
	OnKey       string
	LastError   int
	LastMessage string
	InHandler   bool
	RetryLine   string
	KeyRead     bool // last command read the keyboard (skip ON KEY)

	// Program functions (032).
	PrintCol int
	PrintRow int
	ReadKey  int
	Updated  bool
	FuncKeys [13]string // 1..12

	// Session chrome (033).
	TypeaheadBuf []byte
	Message      string
	Suspended    bool
	InsertMode   bool
	AssistOpen   bool              // ASSIST full-screen menu; suppresses STATUS chrome
	StatusMode   string            // status bar mode label (Command Line, ASSIST, EDIT, …)
	ResizeWired  bool              // hostterm.OnResize registered for this session
	Modules      map[string]string // LOAD name → path (038)
	CatalogPath  string            // SET CATALOG TO path (040)
	CatalogTitle string            // SET TITLE TO text (040)

	// FormatLines are the loaded SET FORMAT TO .fmt lines (023.17).
	FormatLines []string

	openFiles int

	// SessionDate is DATE(); starts as the host local date.
	SessionDate time.Time

	stdinReader *bufio.Reader
	stdinSource io.Reader
}

// New returns a session with areas 1–10, active area 1, and process streams.
func New() *Context {
	now := time.Now()
	y, m, d := now.Date()
	c := &Context{
		Config:      config.New(),
		Symbols:     symbols.New(),
		ActiveArea:  1,
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		SessionDate: time.Date(y, m, d, 0, 0, 0, 0, time.Local),
		Message:     "Enter a Gobi command.",
	}
	for i := 1; i <= AreaCount; i++ {
		c.Areas[i] = &Area{}
	}
	return c
}

// Out returns the writer for screen command output. When SET CONSOLE is off,
// writes are discarded; stderr is unchanged (023.10). When SET ALTERNATE is on
// and a file is open, screen output is also copied there (023.11).
func (c *Context) Out() io.Writer {
	var w io.Writer = c.Stdout
	if c.Config != nil && !c.Config.Console {
		w = io.Discard
	}
	if c.Config != nil && c.Config.Alternate && c.Alternate != nil {
		if w == io.Discard {
			return c.Alternate
		}
		return io.MultiWriter(w, c.Alternate)
	}
	return w
}

// Active returns the active work area.
func (c *Context) Active() *Area {
	return c.Areas[c.ActiveArea]
}

// Select switches the active area. name is "1"–"10", "A"–"J", or an open-table alias.
func (c *Context) Select(name string) error {
	s := strings.TrimSpace(name)
	if s == "" {
		return fmt.Errorf("invalid work area: %q", name)
	}
	if n, err := parseArea(s); err == nil {
		c.ActiveArea = n
		return nil
	}
	want := strings.ToUpper(s)
	for i := 1; i <= AreaCount; i++ {
		a := c.Areas[i]
		if a.HasTable() && a.Alias == want {
			c.ActiveArea = i
			return nil
		}
	}
	return fmt.Errorf("invalid work area: %s", s)
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

// PushTypeahead appends bytes to the typeahead FIFO, respecting SET TYPEAHEAD size (033.3).
func (c *Context) PushTypeahead(b ...byte) {
	max := 20
	if c.Config != nil {
		max = c.Config.Typeahead
	}
	if max <= 0 {
		return // buffering disabled
	}
	c.TypeaheadBuf = append(c.TypeaheadBuf, b...)
	if len(c.TypeaheadBuf) > max {
		c.TypeaheadBuf = c.TypeaheadBuf[len(c.TypeaheadBuf)-max:]
	}
}

// PopTypeahead returns the next typeahead byte, or 0 if empty.
func (c *Context) PopTypeahead() (byte, bool) {
	if len(c.TypeaheadBuf) == 0 {
		return 0, false
	}
	b := c.TypeaheadBuf[0]
	c.TypeaheadBuf = c.TypeaheadBuf[1:]
	return b, true
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
