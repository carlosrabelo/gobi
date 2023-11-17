package expr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandMacrosOnce(t *testing.T) {
	mem := map[string]Value{"X": StringVal("LIST")}
	got, err := ExpandMacros("? &X", func(name string) (Value, bool) {
		v, ok := mem[name]
		return v, ok
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "? LIST" {
		t.Fatalf("got %q", got)
	}
	// Once only: result is not re-expanded.
	mem["Y"] = StringVal("&X")
	got, err = ExpandMacros("? &Y", func(name string) (Value, bool) {
		v, ok := mem[name]
		return v, ok
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "? &X" {
		t.Fatalf("got %q want ? &X", got)
	}
}

func TestExpandMacrosErrors(t *testing.T) {
	_, err := ExpandMacros("&MISSING", func(string) (Value, bool) {
		return Value{}, false
	})
	if err == nil {
		t.Fatal("expected missing error")
	}
	_, err = ExpandMacros("&N", func(string) (Value, bool) {
		return NumberVal(1), true
	})
	if err == nil {
		t.Fatal("expected type error")
	}
}

func TestCompatAcceptance(t *testing.T) {
	got := mustEval(t, `ASC("A")`, &testEnv{})
	if got.Num != 65 {
		t.Fatalf("ASC got %v", got)
	}
	got = mustEval(t, `INT(-1.9)`, &testEnv{})
	if got.Num != -1 {
		t.Fatalf("INT got %v", got)
	}
	got = mustEval(t, `AT("B", "ABC")`, &testEnv{})
	if got.Num != 2 {
		t.Fatalf("AT got %v", got)
	}
	got = mustEval(t, `"AB" $ "XABC"`, &testEnv{})
	if !got.Log {
		t.Fatal("$ expected true")
	}
	got = mustEval(t, `1 # 2`, &testEnv{})
	if !got.Log {
		t.Fatal("# expected true")
	}
	got = mustEval(t, `"A" + "B"`, &testEnv{})
	if got.Str != "AB" {
		t.Fatalf("+ got %q", got.Str)
	}
	got = mustEval(t, `"AB  " - "CD"`, &testEnv{})
	if got.Str != "ABCD  " {
		t.Fatalf("- got %q", got.Str)
	}
}

func TestStringHelpers(t *testing.T) {
	got := mustEval(t, `LEFT("ABCD", 2)`, &testEnv{})
	if got.Str != "AB" {
		t.Fatalf("LEFT %q", got.Str)
	}
	got = mustEval(t, `RIGHT("ABCD", 2)`, &testEnv{})
	if got.Str != "CD" {
		t.Fatalf("RIGHT %q", got.Str)
	}
	got = mustEval(t, `LTRIM("  AB")`, &testEnv{})
	if got.Str != "AB" {
		t.Fatalf("LTRIM %q", got.Str)
	}
	got = mustEval(t, `RTRIM("AB  ")`, &testEnv{})
	if got.Str != "AB" {
		t.Fatalf("RTRIM %q", got.Str)
	}
	got = mustEval(t, `SPACE(3)`, &testEnv{})
	if got.Str != "   " {
		t.Fatalf("SPACE %q", got.Str)
	}
	got = mustEval(t, `REPLICATE("A", 3)`, &testEnv{})
	if got.Str != "AAA" {
		t.Fatalf("REPLICATE %q", got.Str)
	}
	got = mustEval(t, `STUFF("ABCD", 2, 2, "XY")`, &testEnv{})
	if got.Str != "AXYD" {
		t.Fatalf("STUFF %q", got.Str)
	}
}

func TestNumericHelpers(t *testing.T) {
	got := mustEval(t, `ROUND(1.55, 1)`, &testEnv{})
	if got.Num != 1.6 {
		t.Fatalf("ROUND %v", got.Num)
	}
	got = mustEval(t, `ABS(-3)`, &testEnv{})
	if got.Num != 3 {
		t.Fatalf("ABS %v", got.Num)
	}
	got = mustEval(t, `MAX(1, 2)`, &testEnv{})
	if got.Num != 2 {
		t.Fatalf("MAX %v", got.Num)
	}
	got = mustEval(t, `MIN(1, 2)`, &testEnv{})
	if got.Num != 1 {
		t.Fatalf("MIN %v", got.Num)
	}
	got = mustEval(t, `MOD(10, 3)`, &testEnv{})
	if got.Num != 1 {
		t.Fatalf("MOD %v", got.Num)
	}
	got = mustEval(t, `SQRT(9)`, &testEnv{})
	if got.Num != 3 {
		t.Fatalf("SQRT %v", got.Num)
	}
	got = mustEval(t, `EXP(0)`, &testEnv{})
	if got.Num != 1 {
		t.Fatalf("EXP %v", got.Num)
	}
	got = mustEval(t, `LOG(1)`, &testEnv{})
	if got.Num != 0 {
		t.Fatalf("LOG %v", got.Num)
	}
}

func TestTYPEIIF(t *testing.T) {
	env := &testEnv{mem: map[string]Value{"N": NumberVal(1)}}
	got := mustEval(t, `TYPE("1+1")`, env)
	if got.Str != "N" {
		t.Fatalf("TYPE N got %q", got.Str)
	}
	got = mustEval(t, `TYPE('"A"')`, env)
	if got.Str != "C" {
		t.Fatalf("TYPE C got %q", got.Str)
	}
	got = mustEval(t, `TYPE("NOPE")`, env)
	if got.Str != "U" {
		t.Fatalf("TYPE U got %q", got.Str)
	}
	got = mustEval(t, `IIF(.T., 1, 2)`, env)
	if got.Num != 1 {
		t.Fatalf("IIF %v", got.Num)
	}
	got = mustEval(t, `IIF(.F., 1, 2)`, env)
	if got.Num != 2 {
		t.Fatalf("IIF %v", got.Num)
	}
}

func TestFILEDISKSPACE(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.txt")
	if err := os.WriteFile(path, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := &testEnv{default_: dir, hasDefault: true}
	got := mustEval(t, `FILE("x.txt")`, env)
	if !got.Log {
		t.Fatal("FILE expected true")
	}
	got = mustEval(t, `FILE("missing.txt")`, env)
	if got.Log {
		t.Fatal("FILE expected false")
	}
	got = mustEval(t, `DISKSPACE()`, env)
	if got.Kind != KindNumber || got.Num <= 0 {
		t.Fatalf("DISKSPACE %v", got)
	}
}
