package expr

import (
	"testing"
	"time"
)

func TestDateAcceptance(t *testing.T) {
	env := &testEnv{}
	got := mustEval(t, `CTOD("07/04/86")`, env)
	if got.Kind != KindDate || got.EmptyDate {
		t.Fatalf("CTOD %v", got)
	}
	next := mustEval(t, `CTOD("07/04/86")+1`, env)
	want := time.Date(1986, 7, 5, 0, 0, 0, 0, time.Local)
	if next.Date.Year() != want.Year() || next.Date.Month() != want.Month() || next.Date.Day() != want.Day() {
		t.Fatalf("plus1 got %v", next.Date)
	}
	diff := mustEval(t, `CTOD("07/05/86")-CTOD("07/04/86")`, env)
	if diff.Num != 1 {
		t.Fatalf("diff %v", diff.Num)
	}
	// 1986-07-06 was a Sunday
	sun := mustEval(t, `DOW(CTOD("07/06/86"))`, env)
	if sun.Num != 1 {
		t.Fatalf("DOW Sunday got %v", sun.Num)
	}
	empty := mustEval(t, `CTOD("")`, env)
	if !empty.EmptyDate {
		t.Fatal("expected empty date")
	}
}

func TestTIMEAndLUPDATE(t *testing.T) {
	got := mustEval(t, `TIME()`, &testEnv{})
	if len(got.Str) != 8 || got.Str[2] != ':' {
		t.Fatalf("TIME %q", got.Str)
	}
	env := &testEnv{lupdate: DateVal(time.Date(1986, 7, 4, 0, 0, 0, 0, time.Local))}
	got = mustEval(t, `LUPDATE()`, env)
	if got.YYYYMMDD() != "19860704" {
		t.Fatalf("LUPDATE %q", got.YYYYMMDD())
	}
}
