package dbf

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

type packFile struct {
	data []byte
	pos  int
}

func (f *packFile) Read(p []byte) (int, error) {
	if f.pos >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.pos:])
	f.pos += n
	return n, nil
}

func (f *packFile) Write(p []byte) (int, error) {
	end := f.pos + len(p)
	for end > len(f.data) {
		f.data = append(f.data, 0)
	}
	copy(f.data[f.pos:], p)
	f.pos = end
	return len(p), nil
}

func (f *packFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		f.pos = int(offset)
	case io.SeekCurrent:
		f.pos += int(offset)
	case io.SeekEnd:
		f.pos = len(f.data) + int(offset)
	}
	if f.pos < 0 {
		f.pos = 0
	}
	return int64(f.pos), nil
}

func (f *packFile) Truncate(size int64) error {
	if size < 0 {
		size = 0
	}
	if int(size) < len(f.data) {
		f.data = f.data[:size]
	}
	if f.pos > len(f.data) {
		f.pos = len(f.data)
	}
	return nil
}

func TestPackRemovesDeletedRecords(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
		{0x2A, 'T', 'W', 'O', ' ', ' '},
		{0x20, 'T', 'H', 'R', 'E', 'E'},
	}
	data, tbl := buildDBFFile(fields, records)

	pf := &packFile{data: make([]byte, len(data)+50)}
	copy(pf.data, data)

	removed, err := tbl.Pack(pf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	if tbl.Header.RecordCount != 2 {
		t.Fatalf("record count = %d, want 2", tbl.Header.RecordCount)
	}

	cnt := binary.LittleEndian.Uint16(pf.data[1:3])
	if cnt != 2 {
		t.Fatalf("on-disk record count = %d, want 2", cnt)
	}

	wantSize := tbl.HeaderSize() + 2*int(tbl.Header.RecordLen) + 1
	if len(pf.data) != wantSize {
		t.Fatalf("file size = %d, want %d", len(pf.data), wantSize)
	}

	reopened, err := Open(bytes.NewReader(pf.data))
	if err != nil {
		t.Fatalf("reopen packed file: %v", err)
	}
	recs, err := reopened.ReadAllRecords(bytes.NewReader(pf.data[reopened.HeaderSize():]))
	if err != nil {
		t.Fatalf("read packed records: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("packed record count = %d, want 2", len(recs))
	}
	names := []string{"ONE", "THREE"}
	for i, want := range names {
		val, _ := recs[i].DecodeField(reopened, 0)
		if val.(string) != want {
			t.Errorf("record %d = %q, want %q", i, val, want)
		}
		if recs[i].Deleted {
			t.Errorf("record %d should be active after pack", i)
		}
	}
}

func TestPackNoDeletedRecords(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
		{0x20, 'T', 'W', 'O', ' ', ' '},
	}
	data, tbl := buildDBFFile(fields, records)

	pf := &packFile{data: make([]byte, len(data))}
	copy(pf.data, data)

	removed, err := tbl.Pack(pf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed != 0 {
		t.Fatalf("removed = %d, want 0", removed)
	}
	if tbl.Header.RecordCount != 2 {
		t.Fatalf("record count = %d, want 2", tbl.Header.RecordCount)
	}
}

func TestPackAllDeletedRecords(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x2A, 'O', 'N', 'E', ' ', ' '},
		{0x2A, 'T', 'W', 'O', ' ', ' '},
	}
	data, tbl := buildDBFFile(fields, records)

	pf := &packFile{data: make([]byte, len(data))}
	copy(pf.data, data)

	removed, err := tbl.Pack(pf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	if tbl.Header.RecordCount != 0 {
		t.Fatalf("record count = %d, want 0", tbl.Header.RecordCount)
	}
	wantSize := tbl.HeaderSize() + 1
	if len(pf.data) != wantSize {
		t.Fatalf("file size = %d, want %d", len(pf.data), wantSize)
	}
}

type noTruncateFile struct {
	data []byte
	pos  int
}

func (f *noTruncateFile) Read(p []byte) (int, error) {
	if f.pos >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.pos:])
	f.pos += n
	return n, nil
}

func (f *noTruncateFile) Write(p []byte) (int, error) {
	end := f.pos + len(p)
	for end > len(f.data) {
		f.data = append(f.data, 0)
	}
	copy(f.data[f.pos:], p)
	f.pos = end
	return len(p), nil
}

func (f *noTruncateFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		f.pos = int(offset)
	case io.SeekCurrent:
		f.pos += int(offset)
	case io.SeekEnd:
		f.pos = len(f.data) + int(offset)
	}
	if f.pos < 0 {
		f.pos = 0
	}
	return int64(f.pos), nil
}

func TestPackRequiresTruncate(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
		{0x2A, 'T', 'W', 'O', ' ', ' '},
	}
	data, tbl := buildDBFFile(fields, records)

	pf := &noTruncateFile{data: make([]byte, len(data))}
	copy(pf.data, data)

	_, err := tbl.Pack(pf)
	if err == nil {
		t.Fatal("expected error when truncate is unsupported")
	}
}
