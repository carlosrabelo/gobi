package dbf

import (
	"strings"
	"testing"
)

func TestMemoEmptyAndCrossBlock(t *testing.T) {
	mem := &MemFile{}
	m, err := CreateMemo(mem)
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.ReadText(0)
	if err != nil || got != "" {
		t.Fatalf("empty %q %v", got, err)
	}
	// text longer than one block
	text := strings.Repeat("A", MemoBlockSize-1) + "BC"
	block, err := m.AllocWrite(text)
	if err != nil {
		t.Fatal(err)
	}
	if block != 1 {
		t.Fatalf("block %d", block)
	}
	got, err = m.ReadText(block)
	if err != nil {
		t.Fatal(err)
	}
	if got != text {
		t.Fatalf("len got %d want %d", len(got), len(text))
	}
	// end marks present after text
	raw := mem.Bytes()
	off := int(block)*MemoBlockSize + len(text)
	if raw[off] != 0x1A || raw[off+1] != 0x1A {
		t.Fatalf("marks at %d: %02X %02X", off, raw[off], raw[off+1])
	}
}

func TestMemoReplaceChangesBlock(t *testing.T) {
	dbfMem := &MemFile{}
	fields := []Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "NOTE", Type: 'M', Length: 10},
	}
	tab, err := Create(dbfMem, fields, 86, 7, 4)
	if err != nil {
		t.Fatal(err)
	}
	memoMem := &MemFile{}
	memo, err := CreateMemo(memoMem)
	if err != nil {
		t.Fatal(err)
	}
	tab.Memo = memo
	b1, err := memo.AllocWrite("one")
	if err != nil {
		t.Fatal(err)
	}
	rec := Record{Values: []Value{{Str: "ANN"}, {Str: FormatBlockDigits(b1)}}}
	if err := tab.Append(rec); err != nil {
		t.Fatal(err)
	}
	b2, err := memo.AllocWrite("two")
	if err != nil {
		t.Fatal(err)
	}
	rec2, err := tab.Read(1)
	if err != nil {
		t.Fatal(err)
	}
	if rec2.Values[0].Str != "ANN" {
		t.Fatalf("name %q", rec2.Values[0].Str)
	}
	rec2.Values[1] = Value{Str: FormatBlockDigits(b2)}
	if err := tab.Write(1, rec2); err != nil {
		t.Fatal(err)
	}
	got, err := tab.Read(1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Values[0].Str != "ANN" {
		t.Fatal("name changed")
	}
	nb, _ := ParseBlockDigits(got.Values[1].Str)
	if nb != b2 || nb == b1 {
		t.Fatalf("block %d want %d (old %d)", nb, b2, b1)
	}
	text, err := memo.ReadText(nb)
	if err != nil || text != "two" {
		t.Fatalf("text %q %v", text, err)
	}
}
