package ndx

import (
	"testing"
)

func TestInsertSplitSearchDelete(t *testing.T) {
	f := &memFile{}
	idx, err := Create(f, 8, KeyNum, true, "n")
	if err != nil {
		t.Fatal(err)
	}
	max := int(idx.Header().MaxKeys)
	// Insert enough to force at least one split (max+1 keys).
	for i := 0; i <= max; i++ {
		if err := idx.Insert(EncodeKeyNum(float64(i)), uint32(i+1)); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	if idx.Header().Root == 0 {
		t.Fatal("root should be set")
	}
	if idx.Header().PageCount < 3 {
		t.Fatalf("expected split pages, pagecount=%d", idx.Header().PageCount)
	}
	rec, ok, err := idx.Search(EncodeKeyNum(0))
	if err != nil || !ok || rec != 1 {
		t.Fatalf("search 0: rec=%d ok=%v err=%v", rec, ok, err)
	}
	rec, ok, err = idx.Search(EncodeKeyNum(float64(max)))
	if err != nil || !ok || rec != uint32(max+1) {
		t.Fatalf("search max: rec=%d ok=%v err=%v", rec, ok, err)
	}
	_, ok, err = idx.Search(EncodeKeyNum(-1))
	if err != nil || ok {
		t.Fatalf("miss: ok=%v err=%v", ok, err)
	}

	// Unique duplicate ignored.
	if err := idx.Insert(EncodeKeyNum(0), 999); err != nil {
		t.Fatal(err)
	}
	rec, ok, _ = idx.Search(EncodeKeyNum(0))
	if !ok || rec != 1 {
		t.Fatalf("unique kept first: %d", rec)
	}

	// Delete down to empty root.
	for i := 0; i <= max; i++ {
		if err := idx.Delete(EncodeKeyNum(float64(i)), uint32(i+1)); err != nil {
			t.Fatalf("delete %d: %v", i, err)
		}
	}
	if idx.Header().Root != 0 {
		t.Fatalf("root after last delete: %d", idx.Header().Root)
	}
}

func TestCharKeyRoundTrip(t *testing.T) {
	f := &memFile{}
	idx, err := Create(f, 4, KeyChar, false, "name")
	if err != nil {
		t.Fatal(err)
	}
	k := EncodeKeyChar("AB", 4)
	if err := idx.Insert(k, 3); err != nil {
		t.Fatal(err)
	}
	rec, ok, err := idx.Search(EncodeKeyChar("AB", 4))
	if err != nil || !ok || rec != 3 {
		t.Fatalf("rec=%d ok=%v err=%v", rec, ok, err)
	}
}
