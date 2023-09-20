package dbf

import (
	"encoding/binary"
	"fmt"
	"io"
)

// syncer is optional; *os.File implements it.
type syncer interface {
	Sync() error
}

// Table is an open DBF for read and in-place write.
type Table struct {
	Header Header
	rw     io.ReadWriteSeeker
	syncer syncer
}

// Create writes a new empty table (header, terminator, end marker) at the start of rw.
func Create(rw io.ReadWriteSeeker, fields []Field, year, month, day byte) (*Table, error) {
	if _, err := rw.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	h := Header{Year: year, Month: month, Day: day, Fields: fields}
	if err := WriteHeader(rw, h); err != nil {
		return nil, err
	}
	// WriteHeader mutated lengths via local copy — re-read sizes.
	recLen, err := ValidateFields(fields)
	if err != nil {
		return nil, err
	}
	h.RecordLen = recLen
	h.HeaderLen = uint16(headerBase + fieldDescSz*len(fields) + 1)
	if h.HasMemo() {
		h.Signature = sigMemo
	} else {
		h.Signature = sigNoMemo
	}
	if _, err := rw.Write([]byte{endMarker}); err != nil {
		return nil, err
	}
	t := &Table{Header: h, rw: rw}
	if s, ok := rw.(syncer); ok {
		t.syncer = s
	}
	return t, nil
}

// Open reads the header from the start of rw.
func Open(rw io.ReadWriteSeeker) (*Table, error) {
	if _, err := rw.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	h, err := ReadHeader(rw)
	if err != nil {
		return nil, err
	}
	t := &Table{Header: h, rw: rw}
	if s, ok := rw.(syncer); ok {
		t.syncer = s
	}
	return t, nil
}

func (t *Table) recordOffset(n int) (int64, error) {
	if n < 1 || uint32(n) > t.Header.RecordCount {
		return 0, fmt.Errorf("record %d out of range", n)
	}
	return int64(t.Header.HeaderLen) + int64(n-1)*int64(t.Header.RecordLen), nil
}

// Read returns the 1-based record n.
func (t *Table) Read(n int) (Record, error) {
	off, err := t.recordOffset(n)
	if err != nil {
		return Record{}, err
	}
	if _, err := t.rw.Seek(off, io.SeekStart); err != nil {
		return Record{}, err
	}
	raw := make([]byte, t.Header.RecordLen)
	if _, err := io.ReadFull(t.rw, raw); err != nil {
		return Record{}, err
	}
	return DecodeRecord(t.Header.Fields, raw)
}

// Write replaces record n in place without changing the count.
func (t *Table) Write(n int, rec Record) error {
	off, err := t.recordOffset(n)
	if err != nil {
		return err
	}
	raw, err := EncodeRecord(t.Header.Fields, rec)
	if err != nil {
		return err
	}
	if len(raw) != int(t.Header.RecordLen) {
		return fmt.Errorf("encoded length mismatch")
	}
	if _, err := t.rw.Seek(off, io.SeekStart); err != nil {
		return err
	}
	_, err = t.rw.Write(raw)
	return err
}

// Append adds one record, updates the header count, and writes the end marker.
func (t *Table) Append(rec Record) error {
	raw, err := EncodeRecord(t.Header.Fields, rec)
	if err != nil {
		return err
	}
	off := int64(t.Header.HeaderLen) + int64(t.Header.RecordCount)*int64(t.Header.RecordLen)
	if _, err := t.rw.Seek(off, io.SeekStart); err != nil {
		return err
	}
	if _, err := t.rw.Write(raw); err != nil {
		return err
	}
	if _, err := t.rw.Write([]byte{endMarker}); err != nil {
		return err
	}
	t.Header.RecordCount++
	return t.writeCount()
}

func (t *Table) writeCount() error {
	if _, err := t.rw.Seek(4, io.SeekStart); err != nil {
		return err
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], t.Header.RecordCount)
	_, err := t.rw.Write(b[:])
	return err
}

// Close flushes and syncs when the underlying file supports Sync.
func (t *Table) Close() error {
	if t.syncer != nil {
		return t.syncer.Sync()
	}
	return nil
}
