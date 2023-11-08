package dbf

import (
	"fmt"
	"io"
)

// RewriteStructure replaces the DBF header and field descriptors, truncating all records.
func RewriteStructure(w io.ReadWriteSeeker, fields []FieldDescriptor) (*Table, error) {
	trunc, ok := w.(interface{ Truncate(int64) error })
	if !ok {
		return nil, fmt.Errorf("dbf: underlying writer does not support truncate")
	}

	tbl, err := Create(w, fields)
	if err != nil {
		return nil, err
	}

	newSize := int64(tbl.HeaderSize() + 1)
	if err := trunc.Truncate(newSize); err != nil {
		return nil, fmt.Errorf("dbf: truncating file: %w", err)
	}

	return tbl, nil
}
