package expr

import "testing"

// exactEnvironment is a test environment exposing the SET EXACT mode.
type exactEnvironment struct {
	testEnvironment
	exact bool
}

func (e *exactEnvironment) ExactComparison() bool { return e.exact }

func evalComparison(t *testing.T, env Environment, source string) bool {
	t.Helper()
	parser := NewParser(NewLexer(source))
	exp := parser.ParseExpression()
	if len(parser.Errors()) > 0 {
		t.Fatalf("parse %q: %v", source, parser.Errors())
	}
	obj, err := Eval(exp, env)
	if err != nil {
		t.Fatalf("eval %q: %v", source, err)
	}
	boolObj, ok := obj.(*BooleanObject)
	if !ok {
		t.Fatalf("eval %q: expected boolean, got %T", source, obj)
	}
	return boolObj.Value
}

func TestCompareExactOffPrefixMatching(t *testing.T) {
	env := &exactEnvironment{exact: false}

	tests := []struct {
		source string
		want   bool
	}{
		{"'Smith' = 'Sm'", true},
		{"'Sm' = 'Smith'", false},
		{"'Smith' = 'Smith'", true},
		{"'Smith' <> 'Sm'", false},
		{"'Smith' = ''", true},
	}
	for _, tt := range tests {
		if got := evalComparison(t, env, tt.source); got != tt.want {
			t.Errorf("EXACT OFF %s = %v, want %v", tt.source, got, tt.want)
		}
	}
}

func TestCompareExactOnFullMatching(t *testing.T) {
	env := &exactEnvironment{exact: true}

	tests := []struct {
		source string
		want   bool
	}{
		{"'Smith' = 'Sm'", false},
		{"'Smith' = 'Smith'", true},
		{"'Smith' <> 'Sm'", true},
	}
	for _, tt := range tests {
		if got := evalComparison(t, env, tt.source); got != tt.want {
			t.Errorf("EXACT ON %s = %v, want %v", tt.source, got, tt.want)
		}
	}
}

func TestCompareWithoutExactSupportStaysExact(t *testing.T) {
	env := &testEnvironment{}
	if got := evalComparison(t, env, "'Smith' = 'Sm'"); got {
		t.Error("expected exact comparison for environments without EXACT support")
	}
}
