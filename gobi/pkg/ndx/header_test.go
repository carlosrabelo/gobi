package ndx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func sampleHeader() *Header {
	return &Header{
		RootPageID:     3,
		PageCount:      5,
		KeyLength:      20,
		MaxKeysPerPage: 12,
		KeyType:        KeyTypeCharacter,
		Expression:     `UPPER(LAST)+UPPER(FIRST)`,
	}
}

func TestWriteReadHeaderRoundTrip(t *testing.T) {
	want := sampleHeader()

	var buf bytes.Buffer
	if err := WriteHeader(&buf, want); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if buf.Len() != PageSize {
		t.Fatalf("expected %d-byte page, got %d", PageSize, buf.Len())
	}

	got, err := ReadHeader(&buf)
	if err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}
	if got.RootPageID != want.RootPageID ||
		got.PageCount != want.PageCount ||
		got.KeyLength != want.KeyLength ||
		got.MaxKeysPerPage != want.MaxKeysPerPage ||
		got.KeyType != want.KeyType ||
		got.Expression != want.Expression {
		t.Fatalf("round trip mismatch: got %#v want %#v", got, want)
	}
}

func TestWriteHeaderLayout(t *testing.T) {
	want := sampleHeader()

	var buf bytes.Buffer
	if err := WriteHeader(&buf, want); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	page := buf.Bytes()

	if got := binary.LittleEndian.Uint16(page[0:2]); got != want.RootPageID {
		t.Fatalf("root page id = %d, want %d", got, want.RootPageID)
	}
	if got := binary.LittleEndian.Uint16(page[2:4]); got != want.PageCount {
		t.Fatalf("page count = %d, want %d", got, want.PageCount)
	}
	if got := binary.LittleEndian.Uint16(page[4:6]); got != want.KeyLength {
		t.Fatalf("key length = %d, want %d", got, want.KeyLength)
	}
	if got := binary.LittleEndian.Uint16(page[6:8]); got != want.MaxKeysPerPage {
		t.Fatalf("max keys per page = %d, want %d", got, want.MaxKeysPerPage)
	}
	if got := binary.LittleEndian.Uint16(page[8:10]); got != uint16(want.KeyType) {
		t.Fatalf("key type = %d, want %d", got, want.KeyType)
	}
	if got := string(page[10 : 10+len(want.Expression)]); got != want.Expression {
		t.Fatalf("expression prefix = %q, want %q", got, want.Expression)
	}
	if page[10+len(want.Expression)] != 0 {
		t.Fatal("expected null terminator after expression")
	}
	for i := paddingOffset; i < PageSize; i++ {
		if page[i] != 0 {
			t.Fatalf("expected zero padding at offset %d, got 0x%02X", i, page[i])
		}
	}
}

func TestReadHeaderEmptyIndex(t *testing.T) {
	want := &Header{
		RootPageID:     0,
		PageCount:      1,
		KeyLength:      10,
		MaxKeysPerPage: 8,
		KeyType:        KeyTypeNumeric,
		Expression:     "SALARY",
	}

	var buf bytes.Buffer
	if err := WriteHeader(&buf, want); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}

	got, err := ReadHeader(&buf)
	if err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}
	if got.RootPageID != 0 {
		t.Fatalf("expected empty root page, got %d", got.RootPageID)
	}
}

func TestReadHeaderExpressionWithEmbeddedNull(t *testing.T) {
	var page [PageSize]byte
	binary.LittleEndian.PutUint16(page[4:6], 5)
	binary.LittleEndian.PutUint16(page[8:10], uint16(KeyTypeCharacter))
	copy(page[10:], []byte("ABC\x00DEF"))

	got, err := parseHeaderPage(page[:])
	if err != nil {
		t.Fatalf("parseHeaderPage: %v", err)
	}
	if got.Expression != "ABC" {
		t.Fatalf("expected truncated expression %q, got %q", "ABC", got.Expression)
	}
}

func TestReadHeaderExpressionWithoutNullTerminator(t *testing.T) {
	var page [PageSize]byte
	binary.LittleEndian.PutUint16(page[8:10], uint16(KeyTypeCharacter))
	copy(page[10:13], []byte("ABC"))

	got, err := parseHeaderPage(page[:])
	if err != nil {
		t.Fatalf("parseHeaderPage: %v", err)
	}
	if got.Expression != "ABC" {
		t.Fatalf("expected expression %q, got %q", "ABC", got.Expression)
	}
}

func TestReadHeaderExpressionFilledField(t *testing.T) {
	var page [PageSize]byte
	binary.LittleEndian.PutUint16(page[8:10], uint16(KeyTypeCharacter))
	for i := 0; i < expressionSize; i++ {
		page[expressionOffset+i] = 'A'
	}

	got, err := parseHeaderPage(page[:])
	if err != nil {
		t.Fatalf("parseHeaderPage: %v", err)
	}
	if len(got.Expression) != expressionSize {
		t.Fatalf("expected %d-byte expression, got %d", expressionSize, len(got.Expression))
	}
}

func TestWriteHeaderWriteError(t *testing.T) {
	err := WriteHeader(failingWriter{err: errors.New("disk full")}, sampleHeader())
	if err == nil {
		t.Fatal("expected write error")
	}
}

func TestWriteHeaderRejectsInvalidKeyLength(t *testing.T) {
	h := sampleHeader()
	h.KeyLength = MaxKeyLength + 1

	err := WriteHeader(ioDiscard{}, h)
	if err == nil {
		t.Fatal("expected invalid key length error")
	}
}

func TestReadHeaderRejectsInvalidKeyLength(t *testing.T) {
	var page [PageSize]byte
	binary.LittleEndian.PutUint16(page[4:6], MaxKeyLength+1)
	binary.LittleEndian.PutUint16(page[8:10], uint16(KeyTypeCharacter))

	_, err := parseHeaderPage(page[:])
	if err == nil {
		t.Fatal("expected invalid key length error")
	}
}

func TestWriteHeaderRejectsInvalidKeyType(t *testing.T) {
	h := sampleHeader()
	h.KeyType = 9

	err := WriteHeader(ioDiscard{}, h)
	if err == nil {
		t.Fatal("expected invalid key type error")
	}
}

func TestReadHeaderRejectsInvalidKeyType(t *testing.T) {
	var page [PageSize]byte
	binary.LittleEndian.PutUint16(page[8:10], 9)

	_, err := parseHeaderPage(page[:])
	if err == nil {
		t.Fatal("expected invalid key type error")
	}
}

func TestWriteHeaderRejectsLongExpression(t *testing.T) {
	h := sampleHeader()
	h.Expression = string(make([]byte, expressionSize))

	err := WriteHeader(ioDiscard{}, h)
	if err == nil {
		t.Fatal("expected expression too long error")
	}
}

func TestReadHeaderShortPage(t *testing.T) {
	_, err := ReadHeader(bytes.NewReader(make([]byte, PageSize-1)))
	if err == nil {
		t.Fatal("expected short page error")
	}
}

func TestWriteHeaderNilHeader(t *testing.T) {
	err := WriteHeader(ioDiscard{}, nil)
	if err == nil {
		t.Fatal("expected nil header error")
	}
}

func TestParseHeaderPageWrongSize(t *testing.T) {
	_, err := parseHeaderPage(make([]byte, 64))
	if err == nil {
		t.Fatal("expected wrong page size error")
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

type failingWriter struct {
	err error
}

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}
