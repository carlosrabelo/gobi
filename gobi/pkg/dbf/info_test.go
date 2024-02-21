package dbf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.dbf")
	data := []byte{
		0x02,
		0x0A, 0x00,
		0x50, 0x06, 0x01,
		0x20, 0x00,
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write dbf: %v", err)
	}

	hdr, err := ReadFileHeader(path)
	if err != nil {
		t.Fatalf("ReadFileHeader: %v", err)
	}
	if hdr.RecordCount != 10 {
		t.Fatalf("record count = %d, want 10", hdr.RecordCount)
	}
}

func TestReadFileHeaderInvalidSignature(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.dbf")
	if err := os.WriteFile(path, []byte{0x03, 0, 0, 0, 0, 0, 0, 0}, 0644); err != nil {
		t.Fatalf("write dbf: %v", err)
	}

	if _, err := ReadFileHeader(path); err == nil {
		t.Fatal("expected error for invalid signature")
	}
}
