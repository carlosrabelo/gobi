package lbl

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Form is a text .lbl definition.
type Form struct {
	Height, Width, Margin, Across, Gap int
	Body                               []string
}

// Parse reads a text .lbl.
func Parse(r io.Reader) (Form, error) {
	sc := bufio.NewScanner(r)
	if !sc.Scan() {
		return Form{}, fmt.Errorf("empty label form")
	}
	parts := strings.Fields(sc.Text())
	if len(parts) != 5 {
		return Form{}, fmt.Errorf("bad label header")
	}
	var nums [5]int
	for i := 0; i < 5; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return Form{}, fmt.Errorf("bad label header")
		}
		nums[i] = n
	}
	f := Form{Height: nums[0], Width: nums[1], Margin: nums[2], Across: nums[3], Gap: nums[4]}
	if f.Height < 1 || f.Height > 16 || f.Width < 1 || f.Width > 130 || f.Across < 1 || f.Across > 5 || f.Gap < 0 || f.Margin < 1 {
		return Form{}, fmt.Errorf("bad label sizes")
	}
	for sc.Scan() {
		f.Body = append(f.Body, sc.Text())
	}
	if len(f.Body) != f.Height {
		return Form{}, fmt.Errorf("body length mismatch")
	}
	return f, sc.Err()
}
