package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

const (
	// MaxOpenFiles is the session limit for open files.
	MaxOpenFiles = 15
)

// Area is one work area. A new area has no table.
type Area struct {
	Alias  string
	Path   string
	Table  *dbf.Table
	file   *os.File
	Cursor int // 1-based; RecCount+1 at EOF; 0 when no table
	Record dbf.Record
	Landed bool // true when Cursor points at a physical record
	Found  bool
	BOF    bool
	NDX    [7]string
	Format bool // format file open in this area (stub until formats exist)

	// IndexOrder is 1-based record numbers in controlling-index order (empty = physical).
	IndexOrder []int
	IndexPos   int // index into IndexOrder for current key position
}

// HasTable reports whether a DBF is open in the area.
func (a *Area) HasTable() bool {
	return a != nil && a.Table != nil
}

// RecCount returns the header record count, or 0 with no table.
func (a *Area) RecCount() int {
	if !a.HasTable() {
		return 0
	}
	return int(a.Table.Header.RecordCount)
}

// LoadCurrent reads the record at Cursor when it is in range.
func (a *Area) LoadCurrent() error {
	if !a.HasTable() {
		a.Landed = false
		a.Record = dbf.Record{}
		return nil
	}
	n := a.Cursor
	count := a.RecCount()
	if count == 0 || n < 1 || n > count {
		a.Landed = false
		a.Record = dbf.Record{}
		return nil
	}
	rec, err := a.Table.Read(n)
	if err != nil {
		return err
	}
	a.Record = rec
	a.Landed = true
	return nil
}

// CloseArea closes the table, indexes, and format flag in area n and frees its file slot.
func (c *Context) CloseArea(n int) error {
	if n < 1 || n > AreaCount {
		return fmt.Errorf("invalid work area: %d", n)
	}
	a := c.Areas[n]
	a.NDX = [7]string{}
	a.Format = false
	a.Alias = ""
	a.Path = ""
	a.Cursor = 0
	a.Record = dbf.Record{}
	a.Landed = false
	a.Found = false
	a.BOF = false
	if a.Table != nil {
		_ = a.Table.Close()
		a.Table = nil
	}
	if a.file != nil {
		err := a.file.Close()
		a.file = nil
		if c.openFiles > 0 {
			c.openFiles--
		}
		return err
	}
	return nil
}

// CloseIndexes clears index bindings in the active area.
func (c *Context) CloseIndexes() {
	c.Active().NDX = [7]string{}
}

// CloseFormat clears the format-file flag in the active area.
func (c *Context) CloseFormat() {
	c.Active().Format = false
}

// CloseAlternate closes the session alternate file when one is open.
func (c *Context) CloseAlternate() error {
	if c.Alternate == nil {
		return nil
	}
	err := c.Alternate.Close()
	c.Alternate = nil
	if c.openFiles > 0 {
		c.openFiles--
	}
	return err
}

// CloseProcedure clears the procedure-file stub (real files arrive in 029).
func (c *Context) CloseProcedure() {
	c.Procedure = false
}

// CloseDatabases closes every area's table and indexes and selects area 1.
func (c *Context) CloseDatabases() error {
	var first error
	for i := 1; i <= AreaCount; i++ {
		if err := c.CloseArea(i); err != nil && first == nil {
			first = err
		}
	}
	c.ActiveArea = 1
	return first
}

// CloseAll closes databases, formats, alternate, and procedure, then selects area 1.
func (c *Context) CloseAll() error {
	var first error
	for i := 1; i <= AreaCount; i++ {
		c.Areas[i].Format = false
		if err := c.CloseArea(i); err != nil && first == nil {
			first = err
		}
	}
	if err := c.CloseAlternate(); err != nil && first == nil {
		first = err
	}
	c.CloseProcedure()
	c.ActiveArea = 1
	return first
}

// Use closes the active area, then opens name as a DBF when name is not empty.
// alias is the work-area alias; when empty, the base file name is used.
func (c *Context) Use(name, alias string) error {
	if err := c.CloseArea(c.ActiveArea); err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	pathName := ensureDBFExt(name)
	path, err := c.Resolve(pathName)
	if err != nil {
		return err
	}
	if !fileExists(path) {
		return fmt.Errorf("file does not exist: %s", pathName)
	}
	if c.openFiles >= MaxOpenFiles {
		return fmt.Errorf("too many files open")
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("cannot open %s: %v", pathName, err)
	}
	tab, err := dbf.Open(f)
	if err != nil {
		_ = f.Close()
		return err
	}
	a := c.Active()
	a.file = f
	a.Table = tab
	a.Path = path
	if alias != "" {
		a.Alias = strings.ToUpper(alias)
	} else {
		a.Alias = defaultAlias(path)
	}
	a.Found = false
	a.BOF = false
	c.openFiles++
	count := a.RecCount()
	if count >= 1 {
		a.Cursor = 1
	} else {
		a.Cursor = 1 // empty table: EOF, RECNO() is 1
	}
	return a.LoadCurrent()
}

// OpenFileCount returns how many files the session currently has open.
func (c *Context) OpenFileCount() int {
	return c.openFiles
}

func ensureDBFExt(name string) string {
	if filepath.Ext(name) == "" {
		return name + ".dbf"
	}
	return name
}

func defaultAlias(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.ToUpper(strings.TrimSuffix(base, ext))
}
