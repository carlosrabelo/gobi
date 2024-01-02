package ndx

import (
	"fmt"
	"io"
	"os"
)

// Index is an open NDX index file bound to a DBF work area.
type Index struct {
	Path string
	file io.ReadWriteSeeker
	pm   *PageManager
}

// Manager returns the page manager for index operations.
func (idx *Index) Manager() *PageManager {
	if idx == nil {
		return nil
	}
	return idx.pm
}

// Close closes the underlying index file.
func (idx *Index) Close() error {
	if idx == nil || idx.file == nil {
		return nil
	}
	if closer, ok := idx.file.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func openIndexFile(path string) (io.ReadWriteSeeker, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("ndx: creating index file: %w", err)
	}
	return file, nil
}

// OpenIndex opens an existing NDX file without truncating it and returns a
// handle ready for searches and updates (e.g. USE ... INDEX bindings).
func OpenIndex(path string) (*Index, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("ndx: opening index file: %w", err)
	}
	pm, err := OpenPageManager(file)
	if err != nil {
		closeIndexFile(file)
		return nil, err
	}
	return &Index{Path: path, file: file, pm: pm}, nil
}

func closeIndexFile(file io.ReadWriteSeeker) {
	if closer, ok := file.(io.Closer); ok {
		_ = closer.Close()
	}
}
