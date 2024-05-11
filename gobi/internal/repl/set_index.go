package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// applySetIndex implements SET INDEX TO [<ndx list>]: it closes the indexes
// bound to the active table and, when a list is given, binds the named NDX
// files in order (the first becomes the controlling index). An empty list
// leaves the table without indexes.
func applySetIndex(ctx *context.Context, cmd Command) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	names := splitIndexNames(cmd.ToClause)

	closeOpenIndexes(area)
	if len(names) == 0 {
		return nil
	}
	if err := bindUseIndexes(ctx, area, names); err != nil {
		closeOpenIndexes(area)
		return err
	}
	return nil
}

// splitIndexNames splits a comma-separated NDX file list, dropping blanks.
func splitIndexNames(list string) []string {
	var names []string
	for _, part := range strings.Split(list, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			names = append(names, part)
		}
	}
	return names
}
