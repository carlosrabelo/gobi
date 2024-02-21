package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

// handleDir implements DIR [LIKE] [<pattern>] as an alias for the
// DISPLAY FILES / LIST FILES directory listing.
func handleDir(ctx *context.Context, cmd Command) error {
	return outputFiles(ctx, cmd.Args)
}

// outputFiles lists files in the default directory. Without a pattern it
// shows database files with their record counts, like dBase II DISPLAY
// FILES; with LIKE <pattern> it lists every matching file name.
func outputFiles(ctx *context.Context, args string) error {
	pattern, explicit, err := parseFilesPattern(args)
	if err != nil {
		return err
	}

	dir := ctx.Config.DefaultDir
	if dir == "" {
		dir = "."
	}

	names, err := matchDirFiles(dir, pattern)
	if err != nil {
		return fmt.Errorf("*** Error reading directory: %w", err)
	}

	if !explicit {
		fmt.Fprintf(ctx.Stdout, "DATABASE FILES    # RECORDS\r\n")
		for _, name := range names {
			count := dbfRecordCountText(filepath.Join(dir, name))
			fmt.Fprintf(ctx.Stdout, "%-16s  %9s\r\n", strings.ToUpper(name), count)
		}
	} else {
		for _, name := range names {
			fmt.Fprintf(ctx.Stdout, "%s\r\n", strings.ToUpper(name))
		}
	}

	fmt.Fprintf(ctx.Stdout, "%d FILE(S)\r\n", len(names))
	return nil
}

// parseFilesPattern interprets the [LIKE] [<pattern>] arguments shared by
// DISPLAY FILES, LIST FILES, and DIR. It reports whether a pattern was
// given explicitly; otherwise the dBase II default *.dbf listing applies.
func parseFilesPattern(args string) (string, bool, error) {
	word, rest := splitLeadingWord(args)
	if strings.EqualFold(word, "LIKE") {
		pattern := strings.TrimSpace(rest)
		if pattern == "" {
			return "", false, fmt.Errorf("*** LIKE requires a file pattern")
		}
		return pattern, true, nil
	}

	pattern := strings.TrimSpace(args)
	if pattern == "" {
		return "*.dbf", false, nil
	}
	return pattern, true, nil
}

// matchDirFiles returns the sorted file names in dir matching the dBase
// wildcard pattern (case-insensitive * and ?).
func matchDirFiles(dir, pattern string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	upperPattern := strings.ToUpper(pattern)
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ok, err := filepath.Match(upperPattern, strings.ToUpper(entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %s: %w", pattern, err)
		}
		if ok {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// dbfRecordCountText reads a DBF header and renders its record count, or
// a placeholder when the file is not a readable database.
func dbfRecordCountText(path string) string {
	hdr, err := dbf.ReadFileHeader(path)
	if err != nil {
		return "?"
	}
	return fmt.Sprintf("%d", hdr.RecordCount)
}
