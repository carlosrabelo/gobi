package dbf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"testing"
)

func buildDBFWithRecords(fields []FieldDescriptor, records [][]byte) []byte {
	recLen := 1
	for _, f := range fields {
		recLen += int(f.Length)
	}
	data := buildDBF(SignatureStd, uint16(len(records)), uint16(recLen), fields, true)
	for _, rec := range records {
		data = append(data, rec...)
	}
	data = append(data, 0x1A)
	return data
}

func TestReadRecordActive(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'H', 'E', 'L', 'L', 'O'},
	}
	data := buildDBFWithRecords(fields, records)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec, err := tbl.ReadRecord(bytes.NewReader(records[0]))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Deleted {
		t.Error("record should not be marked as deleted")
	}
	if string(rec.Data) != string(records[0]) {
		t.Errorf("data = %q, want %q", string(rec.Data), string(records[0]))
	}
}

func TestReadRecordDeleted(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x2A, 'W', 'O', 'R', 'L', 'D'},
	}
	data := buildDBFWithRecords(fields, records)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec, err := tbl.ReadRecord(bytes.NewReader(records[0]))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rec.Deleted {
		t.Error("record should be marked as deleted")
	}
}

func TestReadRecordEOF(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x1A, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
	data := buildDBFWithRecords(fields, records)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = tbl.ReadRecord(bytes.NewReader(records[0]))
	if err == nil {
		t.Fatal("expected EOF for 0x1A marker")
	}
}

func TestReadRecordInvalidDeletionFlag(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0xFF, 'A', 'B', 'C', 'D', 'E'},
	}
	data := buildDBFWithRecords(fields, records)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = tbl.ReadRecord(bytes.NewReader(records[0]))
	if err == nil {
		t.Fatal("expected error for invalid deletion flag")
	}
}

func TestReadRecordTruncated(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	data := buildDBFWithRecords(fields, nil)
	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = tbl.ReadRecord(bytes.NewReader([]byte{0x20, 'A', 'B'}))
	if err == nil {
		t.Fatal("expected error for truncated record")
	}
}

func TestReadMultipleRecords(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
		{Name: "AGE", Type: FieldTypeNumeric, Length: 3},
	}
	records := [][]byte{
		{0x20, 'H', 'E', 'L', 'L', 'O', ' ', '2', '5'},
		{0x2A, 'W', 'O', 'R', 'L', 'D', ' ', '3', '0'},
		{0x20, 'F', 'O', 'O', ' ', ' ', ' ', '1', '8'},
	}
	all := append([]byte{}, records[0]...)
	all = append(all, records[1]...)
	all = append(all, records[2]...)
	all = append(all, 0x1A)

	data := buildDBFWithRecords(fields, records)
	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := bytes.NewReader(all)
	rec0, err := tbl.ReadRecord(r)
	if err != nil {
		t.Fatalf("record 0: %v", err)
	}
	if rec0.Deleted {
		t.Error("record 0 should not be deleted")
	}

	rec1, err := tbl.ReadRecord(r)
	if err != nil {
		t.Fatalf("record 1: %v", err)
	}
	if !rec1.Deleted {
		t.Error("record 1 should be deleted")
	}

	rec2, err := tbl.ReadRecord(r)
	if err != nil {
		t.Fatalf("record 2: %v", err)
	}
	if rec2.Deleted {
		t.Error("record 2 should not be deleted")
	}

	_, err = tbl.ReadRecord(r)
	if err == nil {
		t.Fatal("expected EOF after all records")
	}
}

func TestRecordFieldData(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
		{Name: "AGE", Type: FieldTypeNumeric, Length: 3},
	}
	records := [][]byte{
		{0x20, 'H', 'E', 'L', 'L', 'O', ' ', '2', '5'},
	}
	data := buildDBFWithRecords(fields, records)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec, err := tbl.ReadRecord(bytes.NewReader(records[0]))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nameData := rec.FieldData(tbl, 0)
	if string(nameData) != "HELLO" {
		t.Errorf("field 0 = %q, want %q", string(nameData), "HELLO")
	}

	ageData := rec.FieldData(tbl, 1)
	if string(ageData) != " 25" {
		t.Errorf("field 1 = %q, want %q", string(ageData), " 25")
	}
}

func TestRecordFieldDataOutOfBounds(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'H', 'E', 'L', 'L', 'O'},
	}
	data := buildDBFWithRecords(fields, records)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec, err := tbl.ReadRecord(bytes.NewReader(records[0]))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.FieldData(tbl, -1) != nil {
		t.Error("expected nil for negative index")
	}
	if rec.FieldData(tbl, 1) != nil {
		t.Error("expected nil for out-of-bounds index")
	}
}

func TestReadAllRecordsWithEOFMarker(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
		{0x20, 'T', 'W', 'O', ' ', ' '},
	}
	all := append([]byte{}, records[0]...)
	all = append(all, records[1]...)
	all = append(all, 0x1A)

	data := buildDBFWithRecords(fields, records)
	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := tbl.ReadAllRecords(bytes.NewReader(all))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("record count = %d, want 2", len(result))
	}
}

func TestReadAllRecordsPhysicalTruncation(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
		{0x20, 'T', 'W', 'O', ' ', ' '},
	}

	recLen := 1
	for _, f := range fields {
		recLen += int(f.Length)
	}
	data := buildDBF(SignatureStd, uint16(len(records)), uint16(recLen), fields, true)
	for _, rec := range records {
		data = append(data, rec...)
	}

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	all := append([]byte{}, records[0]...)
	all = append(all, records[1]...)

	result, err := tbl.ReadAllRecords(bytes.NewReader(all))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("record count = %d, want 2 (physical truncation)", len(result))
	}
}

func TestReadAllRecordsEmpty(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	data := buildDBFWithRecords(fields, nil)
	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := tbl.ReadAllRecords(bytes.NewReader([]byte{0x1A}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("record count = %d, want 0", len(result))
	}
}

func TestWriteEOF(t *testing.T) {
	var buf bytes.Buffer
	err := WriteEOF(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 1 {
		t.Fatalf("written bytes = %d, want 1", buf.Len())
	}
	if buf.Bytes()[0] != 0x1A {
		t.Errorf("marker = 0x%02X, want 0x1A", buf.Bytes()[0])
	}
}

func TestWriteReadEOFMarker(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	recLen := 6

	var buf bytes.Buffer
	buf.Write(buildDBF(SignatureStd, 1, uint16(recLen), fields, true))
	buf.Write([]byte{0x20, 'T', 'E', 'S', 'T', ' '})
	WriteEOF(&buf)

	tbl, err := Open(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recs, err := tbl.ReadAllRecords(bytes.NewReader(buf.Bytes()[len(buf.Bytes())-7:]))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recs) != 1 {
		t.Errorf("record count = %d, want 1", len(recs))
	}
}

func buildDBFFile(fields []FieldDescriptor, records [][]byte) ([]byte, *Table) {
	recLen := 1
	for _, f := range fields {
		recLen += int(f.Length)
	}
	data := buildDBF(SignatureStd, uint16(len(records)), uint16(recLen), fields, true)
	for _, rec := range records {
		data = append(data, rec...)
	}
	data = append(data, 0x1A)
	tbl, _ := Open(bytes.NewReader(data))
	return data, tbl
}

func TestHeaderSize(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
		{Name: "AGE", Type: FieldTypeNumeric, Length: 3},
	}
	_, tbl := buildDBFFile(fields, nil)

	want := 8 + 2*16 + 1
	if tbl.HeaderSize() != want {
		t.Errorf("header size = %d, want %d", tbl.HeaderSize(), want)
	}
}

func TestRecordOffset(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
		{0x20, 'T', 'W', 'O', ' ', ' '},
	}
	_, tbl := buildDBFFile(fields, records)

	headerSize := tbl.HeaderSize()
	off0, err := tbl.RecordOffset(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if off0 != int64(headerSize) {
		t.Errorf("record 0 offset = %d, want %d", off0, headerSize)
	}

	off1, err := tbl.RecordOffset(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want1 := int64(headerSize + int(tbl.Header.RecordLen))
	if off1 != want1 {
		t.Errorf("record 1 offset = %d, want %d", off1, want1)
	}
}

func TestRecordOffsetOutOfRange(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
	}
	_, tbl := buildDBFFile(fields, records)

	_, err := tbl.RecordOffset(-1)
	if err == nil {
		t.Error("expected error for negative record number")
	}

	_, err = tbl.RecordOffset(5)
	if err == nil {
		t.Error("expected error for record number beyond count")
	}
}

func TestReadRecordAt(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
		{0x20, 'T', 'W', 'O', ' ', ' '},
		{0x20, 'T', 'H', 'R', 'E', 'E'},
	}
	data, tbl := buildDBFFile(fields, records)

	r := bytes.NewReader(data)
	rec, err := tbl.ReadRecordAt(r, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	name, _ := rec.DecodeField(tbl, 0)
	if name.(string) != "TWO" {
		t.Errorf("name = %q, want %q", name, "TWO")
	}

	rec0, err := tbl.ReadRecordAt(r, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	name0, _ := rec0.DecodeField(tbl, 0)
	if name0.(string) != "ONE" {
		t.Errorf("name = %q, want %q", name0, "ONE")
	}
}

func TestReadRecordAtOutOfRange(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
	}
	data, tbl := buildDBFFile(fields, records)

	_, err := tbl.ReadRecordAt(bytes.NewReader(data), 99)
	if err == nil {
		t.Fatal("expected error for out-of-range record number")
	}
}

func TestWriteRecordAtInPlace(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
		{0x20, 'T', 'W', 'O', ' ', ' '},
	}
	data, tbl := buildDBFFile(fields, records)

	buf := make([]byte, len(data))
	copy(buf, data)

	rec, _ := NewRecord(tbl, false, []interface{}{"NEW"})

	w := &writeSeeker{data: buf}
	if err := tbl.WriteRecordAt(w, 1, rec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	readback, err := tbl.ReadRecordAt(bytes.NewReader(w.data), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	name, _ := readback.DecodeField(tbl, 0)
	if name.(string) != "NEW" {
		t.Errorf("name = %q, want %q", name, "NEW")
	}

	rec0, _ := tbl.ReadRecordAt(bytes.NewReader(w.data), 0)
	name0, _ := rec0.DecodeField(tbl, 0)
	if name0.(string) != "ONE" {
		t.Errorf("record 0 should be unchanged: %q, want %q", name0, "ONE")
	}
}

type writeSeeker struct {
	data []byte
	pos  int
}

func (w *writeSeeker) Write(p []byte) (int, error) {
	copy(w.data[w.pos:], p)
	w.pos += len(p)
	return len(p), nil
}

func (w *writeSeeker) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekStart {
		w.pos = int(offset)
	}
	return int64(w.pos), nil
}

type growWriteSeeker struct {
	data []byte
	pos  int
}

func (g *growWriteSeeker) Write(p []byte) (int, error) {
	end := g.pos + len(p)
	for end > len(g.data) {
		g.data = append(g.data, 0)
	}
	copy(g.data[g.pos:], p)
	g.pos = end
	return len(p), nil
}

func (g *growWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekStart {
		g.pos = int(offset)
	}
	return int64(g.pos), nil
}

func TestAppendRecord(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
		{0x20, 'T', 'W', 'O', ' ', ' '},
	}
	data, tbl := buildDBFFile(fields, records)

	buf := make([]byte, len(data)+100)
	copy(buf, data)

	g := &growWriteSeeker{data: buf}
	copy(g.data, data)

	rec, _ := NewRecord(tbl, false, []interface{}{"THREE"})
	recNo, err := tbl.AppendRecord(g, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recNo != 2 {
		t.Errorf("returned record number = %d, want 2", recNo)
	}
	if tbl.Header.RecordCount != 3 {
		t.Errorf("header record count = %d, want 3", tbl.Header.RecordCount)
	}

	readback, err := tbl.ReadRecordAt(bytes.NewReader(g.data), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	name, _ := readback.DecodeField(tbl, 0)
	if name.(string) != "THREE" {
		t.Errorf("appended name = %q, want %q", name, "THREE")
	}
}

func TestAppendRecordUpdatesHeaderCount(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
	}
	data, tbl := buildDBFFile(fields, records)

	buf := make([]byte, len(data)+100)
	copy(buf, data)
	g := &growWriteSeeker{data: buf}
	copy(g.data, data)

	rec, _ := NewRecord(tbl, false, []interface{}{"TWO"})
	tbl.AppendRecord(g, rec)

	cnt := binary.LittleEndian.Uint16(g.data[1:3])
	if cnt != 2 {
		t.Errorf("on-disk record count = %d, want 2", cnt)
	}
}

func TestAppendMultipleRecords(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
	}
	data, tbl := buildDBFFile(fields, records)

	buf := make([]byte, len(data)+200)
	copy(buf, data)
	g := &growWriteSeeker{data: buf}
	copy(g.data, data)

	rec1, _ := NewRecord(tbl, false, []interface{}{"TWO"})
	rec2, _ := NewRecord(tbl, false, []interface{}{"THR"})

	no1, _ := tbl.AppendRecord(g, rec1)
	no2, _ := tbl.AppendRecord(g, rec2)

	if no1 != 1 {
		t.Errorf("first append = %d, want 1", no1)
	}
	if no2 != 2 {
		t.Errorf("second append = %d, want 2", no2)
	}
	if tbl.Header.RecordCount != 3 {
		t.Errorf("record count = %d, want 3", tbl.Header.RecordCount)
	}

	all, _ := tbl.ReadAllRecords(bytes.NewReader(g.data[tbl.HeaderSize():]))
	if len(all) != 3 {
		t.Fatalf("total records = %d, want 3", len(all))
	}
	names := []string{"ONE", "TWO", "THR"}
	for i, want := range names {
		val, _ := all[i].DecodeField(tbl, 0)
		if val.(string) != want {
			t.Errorf("record %d = %q, want %q", i, val, want)
		}
	}
}

func TestAppendRecordLengthMismatch(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
	}
	data, tbl := buildDBFFile(fields, records)

	buf := make([]byte, len(data)+100)
	copy(buf, data)
	g := &growWriteSeeker{data: buf}
	copy(g.data, data)

	badRec := &Record{Data: []byte{0x20, 'X'}}
	_, err := tbl.AppendRecord(g, badRec)
	if err == nil {
		t.Fatal("expected error for length mismatch")
	}
}

type mockErrReader struct {
	err error
}

func (m *mockErrReader) Read(p []byte) (n int, err error) {
	return 0, m.err
}

type mockErrReadSeeker struct {
	seekErr error
}

func (m *mockErrReadSeeker) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (m *mockErrReadSeeker) Seek(offset int64, whence int) (int64, error) {
	return 0, m.seekErr
}

type mockErrWriteSeeker struct {
	seekErr  error
	writeErr error
}

func (m *mockErrWriteSeeker) Write(p []byte) (n int, err error) {
	return 0, m.writeErr
}

func (m *mockErrWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	return 0, m.seekErr
}

type mockPartialErrReader struct {
	data []byte
	pos  int
	err  error
}

func (m *mockPartialErrReader) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.data) {
		return 0, m.err
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func TestRecordFieldDataTruncated(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)

	// Record data is shorter than expected offsets
	rec := &Record{Data: []byte{0x20, 'H'}}
	data := rec.FieldData(tbl, 0)
	if data != nil {
		t.Errorf("expected nil for truncated record data, got %q", data)
	}
}

func TestReadRecordGenericError(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)
	expectedErr := fmt.Errorf("read generic error")

	_, err := tbl.ReadRecord(&mockErrReader{err: expectedErr})
	if err == nil || !strings.Contains(err.Error(), "read generic error") {
		t.Errorf("expected read generic error, got %v", err)
	}
}

func TestReadAllRecordsWithError(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)
	expectedErr := fmt.Errorf("read error halfway")

	// Provide first record then fail
	data := []byte{0x20, 'H', 'E', 'L', 'L', 'O'}
	reader := &mockPartialErrReader{data: data, err: expectedErr}

	_, err := tbl.ReadAllRecords(reader)
	if err == nil || !strings.Contains(err.Error(), "read error halfway") {
		t.Errorf("expected read error halfway, got %v", err)
	}
}

func TestReadRecordAtSeekError(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)
	tbl.Header.RecordCount = 1
	expectedErr := fmt.Errorf("seek error")

	_, err := tbl.ReadRecordAt(&mockErrReadSeeker{seekErr: expectedErr}, 0)
	if err == nil || !strings.Contains(err.Error(), "seek error") {
		t.Errorf("expected seek error, got %v", err)
	}
}

func TestWriteRecordAtLengthMismatch(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)
	tbl.Header.RecordCount = 1

	rec := &Record{Data: []byte{0x20, 'A'}} // length mismatch
	err := tbl.WriteRecordAt(&mockErrWriteSeeker{}, 0, rec)
	if err == nil {
		t.Fatal("expected error for record length mismatch")
	}
}

func TestWriteRecordAtSeekError(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)
	tbl.Header.RecordCount = 1
	expectedErr := fmt.Errorf("seek error")

	rec := &Record{Data: []byte{0x20, 'H', 'E', 'L', 'L', 'O'}}
	err := tbl.WriteRecordAt(&mockErrWriteSeeker{seekErr: expectedErr}, 0, rec)
	if err == nil || !strings.Contains(err.Error(), "seek error") {
		t.Errorf("expected seek error, got %v", err)
	}
}

func TestWriteRecordAtWriteError(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)
	tbl.Header.RecordCount = 1
	expectedErr := fmt.Errorf("write error")

	rec := &Record{Data: []byte{0x20, 'H', 'E', 'L', 'L', 'O'}}
	err := tbl.WriteRecordAt(&mockErrWriteSeeker{writeErr: expectedErr}, 0, rec)
	if err == nil || !strings.Contains(err.Error(), "write error") {
		t.Errorf("expected write error, got %v", err)
	}
}

func TestWriteRecordAtInvalidRecordNumber(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)
	tbl.Header.RecordCount = 1

	rec := &Record{Data: []byte{0x20, 'H', 'E', 'L', 'L', 'O'}}
	err := tbl.WriteRecordAt(&mockErrWriteSeeker{}, 99, rec) // out of bounds record index
	if err == nil {
		t.Fatal("expected error for invalid record number")
	}
}

type mockStepWriteSeeker struct {
	seekCount  int
	writeCount int

	failSeekAt  int
	failWriteAt int
}

func (m *mockStepWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	m.seekCount++
	if m.failSeekAt > 0 && m.seekCount == m.failSeekAt {
		return 0, fmt.Errorf("seek error at step %d", m.seekCount)
	}
	return 0, nil
}

func (m *mockStepWriteSeeker) Write(p []byte) (n int, err error) {
	m.writeCount++
	if m.failWriteAt > 0 && m.writeCount == m.failWriteAt {
		return 0, fmt.Errorf("write error at step %d", m.writeCount)
	}
	return len(p), nil
}

func TestAppendRecordSeekAppendError(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)
	rec, _ := NewRecord(tbl, false, []interface{}{"TEST"})

	g := &mockStepWriteSeeker{failSeekAt: 1}
	_, err := tbl.AppendRecord(g, rec)
	if err == nil || !strings.Contains(err.Error(), "seeking to append position") {
		t.Errorf("expected append position seek error, got %v", err)
	}
}

func TestAppendRecordWriteDataError(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)
	rec, _ := NewRecord(tbl, false, []interface{}{"TEST"})

	g := &mockStepWriteSeeker{failWriteAt: 1}
	_, err := tbl.AppendRecord(g, rec)
	if err == nil || !strings.Contains(err.Error(), "writing appended record") {
		t.Errorf("expected writing record error, got %v", err)
	}
}

func TestAppendRecordWriteEOFError(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)
	rec, _ := NewRecord(tbl, false, []interface{}{"TEST"})

	g := &mockStepWriteSeeker{failWriteAt: 2}
	_, err := tbl.AppendRecord(g, rec)
	if err == nil || !strings.Contains(err.Error(), "writing EOF marker") {
		t.Errorf("expected writing EOF marker error, got %v", err)
	}
}

func TestAppendRecordSeekCountError(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)
	rec, _ := NewRecord(tbl, false, []interface{}{"TEST"})

	g := &mockStepWriteSeeker{failSeekAt: 2}
	_, err := tbl.AppendRecord(g, rec)
	if err == nil || !strings.Contains(err.Error(), "seeking to record count") {
		t.Errorf("expected seeking to record count error, got %v", err)
	}
}

func TestAppendRecordWriteCountError(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)
	rec, _ := NewRecord(tbl, false, []interface{}{"TEST"})

	g := &mockStepWriteSeeker{failWriteAt: 3}
	_, err := tbl.AppendRecord(g, rec)
	if err == nil || !strings.Contains(err.Error(), "updating record count") {
		t.Errorf("expected updating record count error, got %v", err)
	}
}
