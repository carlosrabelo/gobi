package expr

import (
	"strings"
	"testing"
)

func TestTransformTemplateChars(t *testing.T) {
	got, err := Transform(StringVal("ab"), "!!")
	if err != nil || got != "AB" {
		t.Fatalf("!: %q %v", got, err)
	}
	got, err = Transform(NumberVal(12), "***")
	if err != nil || !strings.Contains(got, "*") {
		t.Fatalf("*: %q %v", got, err)
	}
	got, err = Transform(NumberVal(1234.5), "9,999.99")
	if err != nil || got != "1,234.50" {
		t.Fatalf(",: %q %v", got, err)
	}
	got, err = Transform(StringVal("xy"), "!!A")
	if err != nil || got != "XY" {
		t.Fatalf("A after !: %q %v", got, err)
	}
	got, err = Transform(NumberVal(5), "###")
	if err != nil || strings.TrimSpace(got) != "5" {
		t.Fatalf("#: %q %v", got, err)
	}
}

func TestTransformFunctionChars(t *testing.T) {
	got, err := Transform(NumberVal(0), "@Z 999")
	if err != nil || strings.TrimSpace(got) != "" {
		t.Fatalf("@Z: %q %v", got, err)
	}
	got, err = Transform(NumberVal(12), "@B 999")
	if err != nil || !strings.HasPrefix(strings.TrimRight(got, " "), "12") {
		t.Fatalf("@B: %q %v", got, err)
	}
	got, err = Transform(NumberVal(1), "@C 9")
	if err != nil || !strings.HasSuffix(got, "CR") {
		t.Fatalf("@C: %q %v", got, err)
	}
	got, err = Transform(NumberVal(-1), "@( 9")
	if err != nil || got != "(1)" {
		t.Fatalf("@(: %q %v", got, err)
	}
	got, err = Transform(StringVal("ab1"), "@A")
	if err != nil || got != "ab" {
		t.Fatalf("@A: %q %v", got, err)
	}
	got, err = Transform(StringVal("ab"), "@!")
	if err != nil || got != "AB" {
		t.Fatalf("@!: %q %v", got, err)
	}
	d := mustEval(t, `CTOD("07/04/86")`, &testEnv{})
	got, err = Transform(d, "@E")
	if err != nil || got != "04/07/86" {
		t.Fatalf("@E: %q %v", got, err)
	}
	got, err = TransformWithDate(d, "@D", DateBritish, false)
	if err != nil || got != "04/07/86" {
		t.Fatalf("@D British: %q %v", got, err)
	}
	if PictureScrollWidth("@S8 XXXXXXXX") != 8 {
		t.Fatalf("scroll width %d", PictureScrollWidth("@S8 XXXXXXXX"))
	}
	stripped := StripPictureLiterals("12-34", "@R 99-99")
	if stripped != "1234" {
		t.Fatalf("@R strip %q", stripped)
	}
}

func TestTransformFloatingDollar(t *testing.T) {
	got, err := Transform(NumberVal(12.3), "$9.99")
	if err != nil {
		t.Fatal(err)
	}
	if got != "$12.30" {
		t.Fatalf("$9.99: %q", got)
	}
	got, err = Transform(NumberVal(12.3), "$$9.99")
	if err != nil {
		t.Fatal(err)
	}
	if got != "$12.30" {
		t.Fatalf("$$9.99: %q", got)
	}
	got, err = Transform(NumberVal(1.2), "$$$9.99")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "$") || !strings.Contains(got, "1.20") {
		t.Fatalf("$$$9.99: %q", got)
	}
}

func TestFormatPictureDisplayAtR(t *testing.T) {
	got := FormatPictureDisplay("1234", "@R 99-99")
	if got != "12-34" {
		t.Fatalf("got %q", got)
	}
	if PictureDisplayWidth("@R 99-99", 4) != 5 {
		t.Fatalf("display width %d", PictureDisplayWidth("@R 99-99", 4))
	}
}

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
