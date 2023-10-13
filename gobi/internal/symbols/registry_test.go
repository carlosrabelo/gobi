package symbols

import (
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func TestRegistryCaseFoldAndLimit(t *testing.T) {
	r := New()
	if err := r.Set("x", expr.NumberVal(1)); err != nil {
		t.Fatal(err)
	}
	v, ok := r.Get("X")
	if !ok || v.Num != 1 {
		t.Fatalf("Get: ok=%v v=%v", ok, v)
	}
	if err := r.Set("abcdefghijk", expr.NumberVal(2)); err == nil {
		t.Fatal("expected 11-char name error")
	}
	if _, ok := r.Get("abcdefghijk"); ok {
		t.Fatal("long name must not store")
	}
	if err := r.Set("ABCDEFGHIJ", expr.NumberVal(3)); err != nil {
		t.Fatal(err)
	}
	r.Release("x")
	if _, ok := r.Get("X"); ok {
		t.Fatal("released")
	}
	r.Release("missing")
}

func TestRegistryMax256(t *testing.T) {
	r := New()
	for i := 0; i < MaxVars; i++ {
		name := nameN(i)
		if err := r.Set(name, expr.NumberVal(float64(i))); err != nil {
			t.Fatalf("set %d: %v", i, err)
		}
	}
	if err := r.Set("ZZZZZZZZZZ", expr.NumberVal(0)); err == nil {
		t.Fatal("expected capacity error")
	}
	if r.Len() != MaxVars {
		t.Fatalf("len %d", r.Len())
	}
	// Overwrite existing is fine.
	if err := r.Set(nameN(0), expr.NumberVal(99)); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseAllLike(t *testing.T) {
	r := New()
	_ = r.Set("N1", expr.NumberVal(1))
	_ = r.Set("N10", expr.NumberVal(10))
	_ = r.Set("X", expr.NumberVal(0))
	r.ReleaseLike("N?", true)
	if _, ok := r.Get("N1"); ok {
		t.Fatal("N1 should be gone")
	}
	if _, ok := r.Get("N10"); !ok {
		t.Fatal("N10 should remain for N?")
	}
	if _, ok := r.Get("X"); !ok {
		t.Fatal("X should remain")
	}
}

func nameN(i int) string {
	// A000 … A255 — 4 chars, unique.
	return "A" + pad3(i)
}

func pad3(i int) string {
	s := []byte{'0', '0', '0'}
	s[2] = byte('0' + i%10)
	s[1] = byte('0' + (i/10)%10)
	s[0] = byte('0' + (i/100)%10)
	return string(s)
}
