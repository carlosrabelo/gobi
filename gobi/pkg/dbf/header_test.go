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
