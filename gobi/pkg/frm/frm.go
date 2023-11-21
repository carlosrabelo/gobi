package frm

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Form is a text .frm definition.
type Form struct {
	Title   string
	Group   string // optional field name
	Columns []Column
}

// Column is one report column.
type Column struct {
	Field string
	Width int
}

// Parse reads a text .frm from r.
func Parse(r io.Reader) (Form, error) {
	sc := bufio.NewScanner(r)
	if !sc.Scan() {
		return Form{}, fmt.Errorf("empty form")
	}
	title := sc.Text()
	if len(title) > 254 {
		return Form{}, fmt.Errorf("title too long")
	}
	f := Form{Title: title}
	for sc.Scan() {
		line := sc.Text()
		trim := strings.TrimSpace(line)
		if trim == "" {
			break
		}
		up := strings.ToUpper(trim)
		if strings.HasPrefix(up, "GROUP ") {
			f.Group = strings.TrimSpace(trim[6:])
			continue
		}
		parts := strings.Fields(trim)
		if len(parts) < 2 {
			return Form{}, fmt.Errorf("bad column line %q", line)
		}
		w, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil || w < 1 {
			return Form{}, fmt.Errorf("bad column width")
		}
		field := strings.Join(parts[:len(parts)-1], " ")
		f.Columns = append(f.Columns, Column{Field: field, Width: w})
		if len(f.Columns) > 24 {
			return Form{}, fmt.Errorf("too many columns")
		}
	}
	if err := sc.Err(); err != nil {
		return Form{}, err
	}
	if len(f.Columns) == 0 {
		return Form{}, fmt.Errorf("no columns")
	}
	return f, nil
}
