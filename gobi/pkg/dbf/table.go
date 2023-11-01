package dbf

import (
	"encoding/binary"
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

	if f, ok := tbl.underlying.(interface{ Flush() error }); ok {
		if err := f.Flush(); err != nil {
			return fmt.Errorf("dbf: flush failed: %w", err)
		}
	}

	if s, ok := tbl.underlying.(interface{ Sync() error }); ok {
		if err := s.Sync(); err != nil {
			return fmt.Errorf("dbf: sync failed: %w", err)
		}
	}

	if c, ok := tbl.underlying.(io.Closer); ok {
		if err := c.Close(); err != nil {
			return fmt.Errorf("dbf: close failed: %w", err)
		}
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

func (tbl *Table) AppendRecord(w io.WriteSeeker, rec *Record) (int, error) {
	if len(rec.Data) != int(tbl.Header.RecordLen) {
		return -1, fmt.Errorf("dbf: record length %d does not match table record length %d", len(rec.Data), tbl.Header.RecordLen)
	}

	appendOff := int64(tbl.HeaderSize() + int(tbl.Header.RecordCount)*int(tbl.Header.RecordLen))
	if _, err := w.Seek(appendOff, io.SeekStart); err != nil {
		return -1, fmt.Errorf("dbf: seeking to append position: %w", err)
	}

	if _, err := w.Write(rec.Data); err != nil {
		return -1, fmt.Errorf("dbf: writing appended record: %w", err)
	}

	if err := WriteEOF(w); err != nil {
		return -1, fmt.Errorf("dbf: writing EOF marker: %w", err)
	}

	newCount := tbl.Header.RecordCount + 1
	recNo := int(tbl.Header.RecordCount)
	tbl.Header.RecordCount = newCount

	if _, err := w.Seek(1, io.SeekStart); err != nil {
		return recNo, fmt.Errorf("dbf: seeking to record count: %w", err)
	}
	var cnt [2]byte
	binary.LittleEndian.PutUint16(cnt[:], newCount)
	if _, err := w.Write(cnt[:]); err != nil {
		return recNo, fmt.Errorf("dbf: updating record count: %w", err)
	}

	return recNo, nil
}

func (tbl *Table) Pack(w io.ReadWriteSeeker) (int, error) {
	trunc, ok := w.(interface{ Truncate(int64) error })
	if !ok {
		return 0, fmt.Errorf("dbf: underlying writer does not support truncate")
	}

	headerSize := tbl.HeaderSize()
	headerBuf := make([]byte, headerSize)
	if _, err := w.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("dbf: seeking to header: %w", err)
	}
	if _, err := io.ReadFull(w, headerBuf); err != nil {
		return 0, fmt.Errorf("dbf: reading header: %w", err)
	}

	oldCount := int(tbl.Header.RecordCount)
	var active []*Record
	for i := 0; i < oldCount; i++ {
		rec, err := tbl.ReadRecordAt(w, i)
		if err != nil {
			return 0, fmt.Errorf("dbf: reading record %d: %w", i, err)
		}
		if rec.Deleted {
			continue
		}
		data := make([]byte, len(rec.Data))
		copy(data, rec.Data)
		data[0] = deletionActive
		active = append(active, &Record{Deleted: false, Data: data})
	}

	newCount := uint16(len(active))
	binary.LittleEndian.PutUint16(headerBuf[1:3], newCount)

	if _, err := w.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("dbf: seeking to rewrite header: %w", err)
	}
	if _, err := w.Write(headerBuf); err != nil {
		return 0, fmt.Errorf("dbf: writing header: %w", err)
	}

	for _, rec := range active {
		if err := tbl.WriteRecord(w, rec); err != nil {
			return 0, fmt.Errorf("dbf: writing packed record: %w", err)
		}
	}

	if err := WriteEOF(w); err != nil {
		return 0, fmt.Errorf("dbf: writing EOF marker: %w", err)
	}

	newSize := int64(headerSize + len(active)*int(tbl.Header.RecordLen) + 1)
	if err := trunc.Truncate(newSize); err != nil {
		return 0, fmt.Errorf("dbf: truncating file: %w", err)
	}

	removed := oldCount - len(active)
	tbl.Header.RecordCount = newCount
	return removed, nil
}
