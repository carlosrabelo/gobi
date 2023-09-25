package context

import (
	"fmt"
)

// Goto moves to physical record n (1-based). Out of range is an error; the cursor stays.
func (c *Context) Goto(n int) error {
	a := c.Active()
	if !a.HasTable() {
		return fmt.Errorf("no database in use")
	}
	count := a.RecCount()
	if n < 1 || n > count {
		return fmt.Errorf("record out of range: %d", n)
	}
	a.Cursor = n
	a.BOF = false
	return a.LoadCurrent()
}

// GoTop moves to record 1, or leaves EOF set on an empty table.
func (c *Context) GoTop() error {
	a := c.Active()
	if !a.HasTable() {
		return fmt.Errorf("no database in use")
	}
	a.BOF = false
	if a.RecCount() == 0 {
		a.Cursor = 1
		return a.LoadCurrent()
	}
	a.Cursor = 1
	return a.LoadCurrent()
}

// GoBottom moves to the last record, or leaves EOF set on an empty table.
func (c *Context) GoBottom() error {
	a := c.Active()
	if !a.HasTable() {
		return fmt.Errorf("no database in use")
	}
	a.BOF = false
	count := a.RecCount()
	if count == 0 {
		a.Cursor = 1
		return a.LoadCurrent()
	}
	a.Cursor = count
	return a.LoadCurrent()
}

// Skip adds n to the 1-based cursor. Past the end sets EOF; before 1 clamps and sets BOF.
func (c *Context) Skip(n int) error {
	a := c.Active()
	if !a.HasTable() {
		return fmt.Errorf("no database in use")
	}
	count := a.RecCount()
	if count == 0 {
		a.Cursor = 1
		a.BOF = n < 0
		return a.LoadCurrent()
	}
	pos := a.Cursor + n
	if pos < 1 {
		a.Cursor = 1
		a.BOF = true
		return a.LoadCurrent()
	}
	a.BOF = false
	if pos > count {
		a.Cursor = count + 1
		return a.LoadCurrent()
	}
	a.Cursor = pos
	return a.LoadCurrent()
}
