package dbf

import (
	"fmt"
	"os"
)

// ReadFileHeader reads the DBF header from path without parsing field descriptors.
func ReadFileHeader(path string) (*Header, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hdr, err := readHeader(f)
	if err != nil {
		return nil, fmt.Errorf("dbf: %s: %w", path, err)
	}
	return hdr, nil
}
