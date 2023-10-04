package dbf

import (
	"fmt"
	"io"
)

type Table struct {
	Header     *Header
	Fields     []FieldDescriptor
	Offset     []int
	underlying io.Reader
}

func Open(r io.Reader) (*Table, error) {
	hdr, err := readHeader(r)
	if err != nil {
		return nil, err
	}

	fields, err := readFieldDescriptors(r)
	if err != nil {
		return nil, err
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("dbf: no field descriptors found")
	}

	offsets := make([]int, len(fields)+1)
	for i, f := range fields {
		offsets[i+1] = offsets[i] + int(f.Length)
	}

	expectedLen := offsets[len(fields)] + 1
	if int(hdr.RecordLen) != expectedLen {
		return nil, fmt.Errorf("dbf: record length %d does not match sum of field lengths %d (plus deletion flag)", hdr.RecordLen, expectedLen)
	}

	return &Table{
		Header:     hdr,
		Fields:     fields,
		Offset:     offsets,
		underlying: r,
	}, nil
}

func (tbl *Table) Underlying() io.Reader {
	return tbl.underlying
}

func (tbl *Table) Close() error {
	if tbl.underlying == nil {
		return nil
	}
	if c, ok := tbl.underlying.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (t *Table) FieldByName(name string) (*FieldDescriptor, int) {
	for i, f := range t.Fields {
		if f.Name == name {
			return &t.Fields[i], i
		}
	}
	return nil, -1
}

func (tbl *Table) HeaderSize() int {
	return 8 + len(tbl.Fields)*fieldDescriptorSize + 1
}

func (tbl *Table) RecordOffset(recNo int) (int64, error) {
	if recNo < 0 || recNo >= int(tbl.Header.RecordCount) {
		return 0, fmt.Errorf("dbf: record number %d out of range [0, %d)", recNo, tbl.Header.RecordCount)
	}
	return int64(tbl.HeaderSize() + recNo*int(tbl.Header.RecordLen)), nil
}
