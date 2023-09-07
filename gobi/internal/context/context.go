package context

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/carlosrabelo/gobi/gobi/internal/config"
	"github.com/carlosrabelo/gobi/gobi/internal/symbols"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/ndx"
	"github.com/carlosrabelo/gobi/gobi/pkg/script"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// AreaName represents the designated database work areas in dBase II.
type AreaName string

const (
	Primary   AreaName = "PRIMARY"
	Secondary AreaName = "SECONDARY"
)

// WorkArea represents a single active database work area.
// It tracks the opened DBF table and any associated active NDX index files.
type WorkArea struct {
	Table        *dbf.Table   `json:"-"`
	Indexes      []*ndx.Index `json:"-"`
	Alias        string       // Alias name for referencing the table
	RecordNo     int          // Current record cursor position (0-based)
	ActiveRecord *dbf.Record  // Current active record loaded in memory
	Found        bool         // True when last LOCATE, CONTINUE, FIND, or SEEK succeeded
	LocateActive bool         // True when a LOCATE search can be resumed with CONTINUE
	LocateFor    string       // FOR expression from the last LOCATE command
	LocateWhile  string       // WHILE expression from the last LOCATE command
	LocateEnd    int          // Exclusive record bound of the LOCATE scope (0 = whole file)
}

// Context manages the global execution state, variables, and open database areas.
type Context struct {
	Config         *config.Config
	Variables      *symbols.Registry // Global memory variable symbol registry
	WorkAreas      map[AreaName]*WorkArea
	ActiveArea     AreaName           // Currently selected work area
	ExecutionStack []string           // PRG script paths on the call stack
	Script         *script.Controller // Active script instruction pointer, nil in plain REPL
	Screen         *term.Screen       // In-memory full-screen TUI buffer
	Stdin          io.Reader          // Input source
	Stdout         io.Writer          // Output destination
	Stderr         io.Writer          // Error output destination

	stdinReader *bufio.Reader // Shared buffered reader over Stdin
	stdinSource io.Reader     // Stdin value the shared reader was built from
}

// StdinReader returns a buffered reader over Stdin shared by the REPL and
// all input-reading commands. Sharing a single reader prevents one consumer
// from buffering (and losing) lines intended for another. The reader is
// rebuilt automatically when Stdin is reassigned.
func (c *Context) StdinReader() *bufio.Reader {
	if c.stdinReader == nil || c.stdinSource != c.Stdin {
		c.stdinReader = bufio.NewReader(c.Stdin)
		c.stdinSource = c.Stdin
	}
	return c.stdinReader
}

// New returns a fully initialized central Context.
func New() *Context {
	areas := make(map[AreaName]*WorkArea)
	areas[Primary] = &WorkArea{Alias: string(Primary)}
	areas[Secondary] = &WorkArea{Alias: string(Secondary)}

	return &Context{
		Config:         config.New(),
		Variables:      symbols.NewRegistry(),
		WorkAreas:      areas,
		ActiveArea:     Primary,
		ExecutionStack: []string{},
		Screen:         term.NewScreen(term.DefaultCols, term.DefaultRows),
		Stdin:          os.Stdin,
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
	}
}

// SelectArea changes the active work area.
func (c *Context) SelectArea(name AreaName) error {
	if name != Primary && name != Secondary {
		return fmt.Errorf("invalid work area: %s", name)
	}
	c.ActiveArea = name
	return nil
}

// GetActiveArea returns the currently active WorkArea.
func (c *Context) GetActiveArea() *WorkArea {
	return c.WorkAreas[c.ActiveArea]
}
