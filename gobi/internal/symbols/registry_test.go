package symbols

import "testing"

func TestRegistrySetAndGet(t *testing.T) {
	reg := NewRegistry()

	if err := reg.Set("m_var", "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := reg.Get("M_VAR")
	if !ok || val != "hello" {
		t.Fatalf("expected hello, got %v ok=%v", val, ok)
	}
}

func TestRegistryGetMissing(t *testing.T) {
	reg := NewRegistry()

	_, ok := reg.Get("missing")
	if ok {
		t.Fatal("expected missing symbol lookup to fail")
	}
}

func TestRegistryOverwrite(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Set("count", float64(1)); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := reg.Set("COUNT", float64(2)); err != nil {
		t.Fatalf("overwrite: %v", err)
	}

	val, ok := reg.Get("count")
	if !ok || val.(float64) != 2 {
		t.Fatalf("expected 2, got %v", val)
	}
}

func TestRegistryDelete(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Set("temp", true); err != nil {
		t.Fatalf("set: %v", err)
	}

	deleted, err := reg.Delete("TEMP")
	if err != nil || !deleted {
		t.Fatalf("expected delete success, deleted=%v err=%v", deleted, err)
	}
	if reg.Len() != 0 {
		t.Fatalf("expected empty registry, got len=%d", reg.Len())
	}
}

func TestRegistryDeleteMissing(t *testing.T) {
	reg := NewRegistry()

	deleted, err := reg.Delete("missing")
	if err != nil || deleted {
		t.Fatalf("expected no-op delete, deleted=%v err=%v", deleted, err)
	}
}

func TestRegistryClear(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Set("a", 1); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := reg.Set("b", 2); err != nil {
		t.Fatalf("set: %v", err)
	}

	reg.Clear()
	if reg.Len() != 0 {
		t.Fatalf("expected cleared registry, got len=%d", reg.Len())
	}
}

func TestRegistryNamesSorted(t *testing.T) {
	reg := NewRegistry()
	for _, pair := range []struct {
		name string
		val  int
	}{
		{"zebra", 1},
		{"alpha", 2},
		{"middle", 3},
	} {
		if err := reg.Set(pair.name, pair.val); err != nil {
			t.Fatalf("set %s: %v", pair.name, err)
		}
	}

	names := reg.Names()
	if len(names) != 3 || names[0] != "ALPHA" || names[1] != "MIDDLE" || names[2] != "ZEBRA" {
		t.Fatalf("unexpected names: %#v", names)
	}
}

func TestNormalizeNameEmpty(t *testing.T) {
	_, err := NormalizeName("   ")
	if err == nil {
		t.Fatal("expected empty name error")
	}
}

func TestValidateName(t *testing.T) {
	if err := ValidateName("valid_name1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateName("1bad"); err == nil {
		t.Fatal("expected invalid leading digit error")
	}
	if err := ValidateName("bad-name"); err == nil {
		t.Fatal("expected invalid character error")
	}
}

func TestRegistryNilSafe(t *testing.T) {
	var reg *Registry

	if reg.Len() != 0 {
		t.Fatal("expected nil registry len 0")
	}
	if reg.Names() != nil {
		t.Fatal("expected nil names slice")
	}
	if _, ok := reg.Get("x"); ok {
		t.Fatal("expected nil registry lookup to miss")
	}
	if err := reg.Set("x", 1); err == nil {
		t.Fatal("expected error setting on nil registry")
	}
}
