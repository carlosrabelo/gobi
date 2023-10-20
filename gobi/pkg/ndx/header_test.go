package ndx

import (
	"bytes"
	"testing"
)

func TestHeaderOffsets(t *testing.T) {
	h := Header{
		Root:      1,
		PageCount: 2,
		KeyLen:    8,
		Type:      KeyNum,
		Unique:    true,
		Expr:      "AMOUNT",
	}
	buf, err := EncodeHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	if len(buf) != PageSize {
		t.Fatalf("len %d", len(buf))
	}
	got, err := DecodeHeader(buf)
	if err != nil {
		t.Fatal(err)
	}
	if got.Root != 1 || got.PageCount != 2 {
		t.Fatalf("root/pages: %+v", got)
	}
	if got.KeyLen != 8 || got.Type != KeyNum {
		t.Fatalf("key: %+v", got)
	}
	if !got.Unique {
		t.Fatal("unique")
	}
	if got.Expr != "amount" {
		t.Fatalf("expr %q", got.Expr)
	}
	if got.EntrySize != uint32(EntrySizeOf(8)) {
		t.Fatalf("entry size %d", got.EntrySize)
	}
	if got.MaxKeys != uint16(MaxKeysOf(int(got.EntrySize))) {
		t.Fatalf("max keys %d", got.MaxKeys)
	}

	var out bytes.Buffer
	if err := WriteHeader(&out, h); err != nil {
		t.Fatal(err)
	}
	got2, err := ReadHeader(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if got2.Root != got.Root {
		t.Fatalf("%+v", got2)
	}
}

func TestHeaderRejectsBadTypeAndLen(t *testing.T) {
	buf := make([]byte, PageSize)
	buf[16] = 2 // bad type
	buf[12] = 8
	if _, err := DecodeHeader(buf); err == nil {
		t.Fatal("expected type error")
	}
	buf[16] = 0
	buf[12] = 101
	if _, err := DecodeHeader(buf); err == nil {
		t.Fatal("expected length error")
	}
}
