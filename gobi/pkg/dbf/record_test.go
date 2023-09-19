package dbf

import (
	"bytes"
	"testing"
)

func TestDecodeC(t *testing.T) {
	fields := []Field{{Name: "NAME", Type: 'C', Length: 8}}
	raw := []byte{' '}
	raw = append(raw, []byte("ABC     ")...)
	rec, err := DecodeRecord(fields, raw)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Values[0].Str != "ABC" {
		t.Fatalf("got %q", rec.Values[0].Str)
	}
}

func TestDecodeN(t *testing.T) {
	fields := []Field{{Name: "AMT", Type: 'N', Length: 6, Decimals: 1}}
	raw := append([]byte{' '}, []byte("  12.5")...)
	rec, err := DecodeRecord(fields, raw)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Values[0].Num != 12.5 {
		t.Fatalf("got %v", rec.Values[0].Num)
	}
}

func TestDecodeL(t *testing.T) {
	fields := []Field{{Name: "OK", Type: 'L', Length: 1}}
	rec, err := DecodeRecord(fields, []byte{' ', 'Y'})
	if err != nil {
		t.Fatal(err)
	}
	if !rec.Values[0].Log {
		t.Fatal("want true")
	}
}

func TestDecodeD(t *testing.T) {
	fields := []Field{{Name: "WHEN", Type: 'D', Length: 8}}
	raw := append([]byte{' '}, []byte("19860704")...)
	rec, err := DecodeRecord(fields, raw)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Values[0].Str != "19860704" {
		t.Fatalf("got %q", rec.Values[0].Str)
	}
}

func TestDecodeMEmpty(t *testing.T) {
	fields := []Field{{Name: "NOTE", Type: 'M', Length: 10}}
	raw := append([]byte{' '}, []byte("          ")...)
	rec, err := DecodeRecord(fields, raw)
	if err != nil {
		t.Fatal(err)
	}
	if !rec.Values[0].Empty {
		t.Fatal("want empty memo")
	}
}

func TestDecodeDeleted(t *testing.T) {
	fields := []Field{{Name: "A", Type: 'C', Length: 1}}
	rec, err := DecodeRecord(fields, []byte{'*', 'X'})
	if err != nil {
		t.Fatal(err)
	}
	if !rec.Deleted {
		t.Fatal("want deleted")
	}
}

func TestDecodeBadMark(t *testing.T) {
	fields := []Field{{Name: "A", Type: 'C', Length: 1}}
	_, err := DecodeRecord(fields, []byte{'!', 'X'})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReadRecordsEndMarker(t *testing.T) {
	fields := []Field{{Name: "A", Type: 'C', Length: 1}}
	h := Header{RecordCount: 1, RecordLen: 2, Fields: fields}
	// claim 1 record but only end marker
	_, err := ReadRecords(bytes.NewReader([]byte{endMarker}), h)
	if err == nil {
		t.Fatal("expected short table error")
	}
}
