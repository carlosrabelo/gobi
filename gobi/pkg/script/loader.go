// Package script loads dBase II command files (.prg, .bak).
package script

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotFound indicates the command file does not exist.
var ErrNotFound = errors.New("command file not found")

// Program represents a located and parsed command file.
type Program struct {
	Path  string
	Lines []Line
}

// ResolvePath returns the filesystem path for a command file name.
// When no extension is given, .prg is appended. Relative paths are resolved
// against defaultDir when it is set.
func ResolvePath(defaultDir, filename string) string {
	filename = strings.TrimSpace(filename)
	if !strings.Contains(filename, ".") {
		filename += ".prg"
	}
	if defaultDir != "" && !filepath.IsAbs(filename) {
		filename = filepath.Join(defaultDir, filename)
	}
	return filename
}

// Load locates filename, verifies it exists, and parses the command file.
func Load(defaultDir, filename string) (*Program, error) {
	path := ResolvePath(defaultDir, filename)

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("stat command file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a command file")
	}

	return Read(path)
}
