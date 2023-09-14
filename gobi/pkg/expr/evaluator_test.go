package expr

import (
	"fmt"
	"testing"
)

type testEnvironment struct {
	fields    map[string]Object
	variables map[string]Object
	eof       bool
	recno     float64
	deleted   bool
	found     bool
}

func (e *testEnvironment) GetField(name string) (Object, bool) {
	if e == nil || e.fields == nil {
		return nil, false
	}
	obj, ok := e.fields[name]
	return obj, ok
}

func (e *testEnvironment) GetVariable(name string) (Object, bool) {
	if e == nil || e.variables == nil {
		return nil, false
	}
	obj, ok := e.variables[name]
	return obj, ok
}

func (e *testEnvironment) CallFunction(name string, args []Object) (Object, error) {
	return nil, fmt.Errorf("environment: unknown function %q", name)
}
func TestEvalNumberLiteral(t *testing.T) {
	result := testEval(t, "42")
	numberObj, ok := result.(*NumberObject)
	if !ok {
		t.Fatalf("expected *NumberObject, got %T", result)
	}
	if numberObj.Value != 42 {
		t.Fatalf("expected Value=42, got %v", numberObj.Value)
	}
}

func TestEvalFloatLiteral(t *testing.T) {
	result := testEval(t, "45.67")
	numberObj, ok := result.(*NumberObject)
	if !ok {
		t.Fatalf("expected *NumberObject, got %T", result)
	}
	if numberObj.Value != 45.67 {
		t.Fatalf("expected Value=45.67, got %v", numberObj.Value)
	}
}

func TestEvalStringLiteral(t *testing.T) {
	result := testEval(t, `"hello"`)
	strObj, ok := result.(*StringObject)
	if !ok {
		t.Fatalf("expected *StringObject, got %T", result)
	}
	if strObj.Value != "hello" {
		t.Fatalf("expected Value=hello, got %q", strObj.Value)
	}
}

func TestEvalStringLiteralSingleQuote(t *testing.T) {
	result := testEval(t, `'world'`)
	strObj, ok := result.(*StringObject)
	if !ok {
		t.Fatalf("expected *StringObject, got %T", result)
	}
	if strObj.Value != "world" {
		t.Fatalf("expected Value=world, got %q", strObj.Value)
	}
}

func TestEvalStringLiteralBracket(t *testing.T) {
	result := testEval(t, `[bracket string]`)
	strObj, ok := result.(*StringObject)
	if !ok {
		t.Fatalf("expected *StringObject, got %T", result)
	}
	if strObj.Value != "bracket string" {
		t.Fatalf("expected Value='bracket string', got %q", strObj.Value)
	}
}

func TestEvalBooleanTrue(t *testing.T) {
	result := testEval(t, ".T.")
	boolObj, ok := result.(*BooleanObject)
	if !ok {
		t.Fatalf("expected *BooleanObject, got %T", result)
	}
	if boolObj.Value != true {
		t.Fatalf("expected Value=true, got %v", boolObj.Value)
	}
}

func TestEvalBooleanFalse(t *testing.T) {
	result := testEval(t, ".F.")
	boolObj, ok := result.(*BooleanObject)
	if !ok {
		t.Fatalf("expected *BooleanObject, got %T", result)
	}
	if boolObj.Value != false {
		t.Fatalf("expected Value=false, got %v", boolObj.Value)
	}
}

func TestEvalBooleanYes(t *testing.T) {
	result := testEval(t, ".Y.")
	boolObj, ok := result.(*BooleanObject)
	if !ok {
		t.Fatalf("expected *BooleanObject, got %T", result)
	}
	if boolObj.Value != true {
		t.Fatalf("expected Value=true, got %v", boolObj.Value)
	}
}

func TestEvalBooleanNo(t *testing.T) {
	result := testEval(t, ".N.")
	boolObj, ok := result.(*BooleanObject)
	if !ok {
		t.Fatalf("expected *BooleanObject, got %T", result)
	}
	if boolObj.Value != false {
		t.Fatalf("expected Value=false, got %v", boolObj.Value)
	}
}

func TestEvalNilNode(t *testing.T) {
	_, err := Eval(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil node, got nil")
	}
}

func TestEvalUnsupportedNode(t *testing.T) {
	_, err := Eval(&UnaryExpression{Operator: "-", Right: &NumberLiteral{Value: 1}}, nil)
	if err == nil {
		t.Fatal("expected error for unsupported node, got nil")
	}
}

func TestEvalObjectTypes(t *testing.T) {
	tests := []struct {
		input   string
		objType ObjectType
		str     string
	}{
		{"42", NumberType, "42"},
		{`"hello"`, StringType, "hello"},
		{".T.", BooleanType, ".T."},
		{".F.", BooleanType, ".F."},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		if result.Type() != tt.objType {
			t.Fatalf("input %q: expected Type()=%q, got %q", tt.input, tt.objType, result.Type())
		}
		if result.String() != tt.str {
			t.Fatalf("input %q: expected String()=%q, got %q", tt.input, tt.str, result.String())
		}
	}
}

func testEval(t *testing.T, input string) Object {
	t.Helper()
	return testEvalWithEnv(t, input, &testEnvironment{fields: map[string]Object{}})
}

func testEvalWithEnv(t *testing.T, input string, env Environment) Object {
	t.Helper()
	l := NewLexer(input)
	p := NewParser(l)
	exp := p.ParseExpression()

	if len(p.Errors()) != 0 {
		t.Fatalf("input %q: parser errors: %v", input, p.Errors())
	}

	result, err := Eval(exp, env)
	if err != nil {
		t.Fatalf("input %q: eval error: %v", input, err)
	}
	return result
}
