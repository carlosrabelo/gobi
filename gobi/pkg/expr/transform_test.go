package expr

import "testing"

func TestTransformAcceptance(t *testing.T) {
	got, err := Transform(NumberVal(1234.5), "9,999.99")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1,234.50" {
		t.Fatalf("got %q", got)
	}
	got, err = Transform(NumberVal(-1), "@X 9")
	if err != nil {
		t.Fatal(err)
	}
	if !stringsHasSuffix(got, "DB") {
		t.Fatalf("got %q", got)
	}
	d := mustEval(t, `CTOD("07/04/86")`, &testEnv{})
	got, err = Transform(d, "@D")
	if err != nil {
		t.Fatal(err)
	}
	if got != "07/04/86" {
		t.Fatalf("date %q", got)
	}
}

func stringsHasSuffix(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}
