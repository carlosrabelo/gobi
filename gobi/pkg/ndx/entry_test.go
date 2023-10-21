package ndx

import (
	"bytes"
	"testing"
)

func TestEntryCharPaddedAndNumeric(t *testing.T) {
	keyLen := 5
	size := EntrySizeOf(keyLen)
	if size != 16 { // 8+5 -> 16
		t.Fatalf("entry size %d", size)
	}
	e := Entry{
		Child: 2,
		RecNo: 7,
		Key:   EncodeKeyChar("AB", keyLen),
	}
	if !bytes.Equal(e.Key, []byte{'A', 'B', ' ', ' ', ' '}) {
		t.Fatalf("pad: %q", e.Key)
	}
	dst := make([]byte, size)
	if err := EncodeEntry(dst, e, keyLen); err != nil {
		t.Fatal(err)
	}
	got, err := DecodeEntry(dst, keyLen)
	if err != nil {
		t.Fatal(err)
	}
	if got.Child != 2 || got.RecNo != 7 || !bytes.Equal(got.Key, e.Key) {
		t.Fatalf("%+v", got)
	}

	nk := EncodeKeyNum(3.5)
	if DecodeKeyNum(nk) != 3.5 {
		t.Fatal("numeric key")
	}
	if CompareKeys(EncodeKeyChar("A", 4), EncodeKeyChar("B", 4), KeyChar) >= 0 {
		t.Fatal("char compare")
	}
	if CompareKeys(EncodeKeyNum(1), EncodeKeyNum(2), KeyNum) >= 0 {
		t.Fatal("num compare")
	}
}
