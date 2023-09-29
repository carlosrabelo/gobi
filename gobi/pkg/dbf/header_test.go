package dbf

import (
	"bytes"
	"testing"
)

func TestReadHeaderValidStd(t *testing.T) {
	data := []byte{
		0x02,
		0x0A, 0x00,
		0x50, 0x06, 0x01,
		0x20, 0x00,
	}
	h, err := readHeader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Signature != SignatureStd {
		t.Errorf("signature = 0x%02X, want 0x%02X", h.Signature, SignatureStd)
	}
	if h.RecordCount != 10 {
		t.Errorf("record count = %d, want 10", h.RecordCount)
	}
	if h.RecordLen != 32 {
		t.Errorf("record length = %d, want 32", h.RecordLen)
	}
}

func TestReadHeaderValidMemo(t *testing.T) {
	data := []byte{
		0x82,
		0x01, 0x00,
		0x5A, 0x0C, 0x1F,
		0x40, 0x00,
	}
	h, err := readHeader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Signature != SignatureMemo {
		t.Errorf("signature = 0x%02X, want 0x%02X", h.Signature, SignatureMemo)
	}
	if h.RecordCount != 1 {
		t.Errorf("record count = %d, want 1", h.RecordCount)
	}
	if h.RecordLen != 64 {
		t.Errorf("record length = %d, want 64", h.RecordLen)
	}
}

func TestReadHeaderInvalidSignature(t *testing.T) {
	data := []byte{
		0x03,
		0x00, 0x00,
		0x00, 0x00, 0x00,
		0x00, 0x00,
	}
	_, err := readHeader(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestReadHeaderShortData(t *testing.T) {
	_, err := readHeader(bytes.NewReader([]byte{0x02, 0x00}))
	if err == nil {
		t.Fatal("expected error for short data")
	}
}

func TestReadHeaderRecordCountZero(t *testing.T) {
	data := []byte{
		0x02,
		0x00, 0x00,
		0x50, 0x06, 0x01,
		0x20, 0x00,
	}
	h, err := readHeader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.RecordCount != 0 {
		t.Errorf("record count = %d, want 0", h.RecordCount)
	}
}

func TestReadHeaderRecordCountMaxUint16(t *testing.T) {
	data := []byte{
		0x02,
		0xFF, 0xFF,
		0x50, 0x06, 0x01,
		0xE8, 0x03,
	}
	h, err := readHeader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.RecordCount != 65535 {
		t.Errorf("record count = %d, want 65535", h.RecordCount)
	}
	if h.RecordLen != 1000 {
		t.Errorf("record length = %d, want 1000", h.RecordLen)
	}
}

func TestReadHeaderRecordCountLittleEndian(t *testing.T) {
	data := []byte{
		0x02,
		0xD2, 0x04,
		0x50, 0x06, 0x01,
		0x20, 0x00,
	}
	h, err := readHeader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.RecordCount != 1234 {
		t.Errorf("record count = %d, want 1234", h.RecordCount)
	}
}

func TestReadHeaderRecordLenLittleEndian(t *testing.T) {
	data := []byte{
		0x02,
		0x01, 0x00,
		0x50, 0x06, 0x01,
		0x01, 0x04,
	}
	h, err := readHeader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.RecordLen != 1025 {
		t.Errorf("record length = %d, want 1025", h.RecordLen)
	}
}
