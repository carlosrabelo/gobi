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
	syncIndexPos(a)
	return a.LoadCurrent()
}

// GoTop moves to the first record in controlling-index order, or physical record 1.
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
	if len(a.IndexOrder) > 0 {
		a.IndexPos = firstVisibleIndex(c, a, 0, 1)
		if a.IndexPos < 0 {
			a.Cursor = a.RecCount() + 1
			return a.LoadCurrent()
		}
		a.Cursor = a.IndexOrder[a.IndexPos]
		return a.LoadCurrent()
	}
	a.Cursor = 1
	return a.LoadCurrent()
}

// GoBottom moves to the last record in controlling-index order, or the last physical.
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
	if len(a.IndexOrder) > 0 {
		a.IndexPos = firstVisibleIndex(c, a, len(a.IndexOrder)-1, -1)
		if a.IndexPos < 0 {
			a.Cursor = count + 1
			return a.LoadCurrent()
		}
		a.Cursor = a.IndexOrder[a.IndexPos]
		return a.LoadCurrent()
	}
	a.Cursor = count
	return a.LoadCurrent()
}

// Skip adds n steps in controlling-index order when bound, else physical order.
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
	if len(a.IndexOrder) > 0 {
		return skipIndex(c, a, n)
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

func skipIndex(c *Context, a *Area, n int) error {
	dir := 1
	if n < 0 {
		dir = -1
		n = -n
	}
	for step := 0; step < n; step++ {
		next := firstVisibleIndex(c, a, a.IndexPos+dir, dir)
		if next < 0 {
			if dir < 0 {
				a.Cursor = 1
				a.BOF = true
				a.IndexPos = 0
				return a.LoadCurrent()
			}
			a.Cursor = a.RecCount() + 1
			a.BOF = false
			a.IndexPos = len(a.IndexOrder)
			return a.LoadCurrent()
		}
		a.IndexPos = next
		a.Cursor = a.IndexOrder[a.IndexPos]
		a.BOF = false
	}
	return a.LoadCurrent()
}

func firstVisibleIndex(c *Context, a *Area, start, dir int) int {
	for i := start; i >= 0 && i < len(a.IndexOrder); i += dir {
		rec := a.IndexOrder[i]
		if c.Config.Deleted {
			r, err := a.Table.Read(rec)
			if err != nil {
				continue
			}
			if r.Deleted {
				continue
			}
		}
		return i
	}
	return -1
}

func syncIndexPos(a *Area) {
	if len(a.IndexOrder) == 0 {
		return
	}
	for i, r := range a.IndexOrder {
		if r == a.Cursor {
			a.IndexPos = i
			return
		}
	}
}
