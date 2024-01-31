package expr

import "testing"

func TestProgramFunctionsAcceptance(t *testing.T) {
	got := mustEval(t, `ISALPHA("a")`, &testEnv{})
	if !got.Log {
		t.Fatal("ISALPHA")
	}
	got = mustEval(t, "VERSION()", &testEnv{})
	if got.Str != "Gobi 0.1.0" {
		t.Fatalf("VERSION %q", got.Str)
	}
	got = mustEval(t, "FKMAX()", &testEnv{})
	if got.Num != 12 {
		t.Fatalf("FKMAX %v", got.Num)
	}
}
