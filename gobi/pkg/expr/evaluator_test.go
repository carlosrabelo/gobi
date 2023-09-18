package expr

import (
	"fmt"
	"strings"
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
	switch strings.ToUpper(name) {
	case "EOF":
		if len(args) != 0 {
			return nil, fmt.Errorf("environment: function %q expects 0 arguments, got %d", name, len(args))
		}
		return &BooleanObject{Value: e.eof}, nil
	default:
		return nil, fmt.Errorf("environment: unknown function %q", name)
	}
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
func TestEvalFieldLookupString(t *testing.T) {
	env := &testEnvironment{fields: map[string]Object{
		"NOME": &StringObject{Value: "Joao"},
	}}

	result := testEvalWithEnv(t, "NOME", env)
	strObj, ok := result.(*StringObject)
	if !ok {
		t.Fatalf("expected *StringObject, got %T", result)
	}
	if strObj.Value != "Joao" {
		t.Fatalf("expected Value=Joao, got %q", strObj.Value)
	}
}

func TestEvalFieldLookupNumber(t *testing.T) {
	env := &testEnvironment{fields: map[string]Object{
		"SALARIO": &NumberObject{Value: 2500.50},
	}}

	result := testEvalWithEnv(t, "SALARIO", env)
	numObj, ok := result.(*NumberObject)
	if !ok {
		t.Fatalf("expected *NumberObject, got %T", result)
	}
	if numObj.Value != 2500.50 {
		t.Fatalf("expected Value=2500.50, got %v", numObj.Value)
	}
}

func TestEvalFieldLookupBoolean(t *testing.T) {
	env := &testEnvironment{fields: map[string]Object{
		"APROVADO": &BooleanObject{Value: true},
	}}

	result := testEvalWithEnv(t, "APROVADO", env)
	boolObj, ok := result.(*BooleanObject)
	if !ok {
		t.Fatalf("expected *BooleanObject, got %T", result)
	}
	if boolObj.Value != true {
		t.Fatalf("expected Value=true, got %v", boolObj.Value)
	}
}

func TestEvalFieldNotFound(t *testing.T) {
	env := &testEnvironment{fields: map[string]Object{}}

	l := NewLexer("INEXISTENTE")
	p := NewParser(l)
	exp := p.ParseExpression()

	_, err := Eval(exp, env)
	if err == nil {
		t.Fatal("expected error for missing field, got nil")
	}
}

func TestEvalFieldLookupNilEnv(t *testing.T) {
	l := NewLexer("CAMPO")
	p := NewParser(l)
	exp := p.ParseExpression()

	_, err := Eval(exp, nil)
	if err == nil {
		t.Fatal("expected error for nil environment, got nil")
	}
}

func TestEvalVariableLookupString(t *testing.T) {
	env := &testEnvironment{variables: map[string]Object{
		"NOME": &StringObject{Value: "Maria"},
	}}

	result := testEvalWithEnv(t, "NOME", env)
	strObj, ok := result.(*StringObject)
	if !ok {
		t.Fatalf("expected *StringObject, got %T", result)
	}
	if strObj.Value != "Maria" {
		t.Fatalf("expected Value=Maria, got %q", strObj.Value)
	}
}

func TestEvalVariableLookupNumber(t *testing.T) {
	env := &testEnvironment{variables: map[string]Object{
		"CONTADOR": &NumberObject{Value: 99},
	}}

	result := testEvalWithEnv(t, "CONTADOR", env)
	numObj, ok := result.(*NumberObject)
	if !ok {
		t.Fatalf("expected *NumberObject, got %T", result)
	}
	if numObj.Value != 99 {
		t.Fatalf("expected Value=99, got %v", numObj.Value)
	}
}

func TestEvalVariableLookupBoolean(t *testing.T) {
	env := &testEnvironment{variables: map[string]Object{
		"FLAG": &BooleanObject{Value: true},
	}}

	result := testEvalWithEnv(t, "FLAG", env)
	boolObj, ok := result.(*BooleanObject)
	if !ok {
		t.Fatalf("expected *BooleanObject, got %T", result)
	}
	if boolObj.Value != true {
		t.Fatalf("expected Value=true, got %v", boolObj.Value)
	}
}

func TestEvalVariableShadowsField(t *testing.T) {
	env := &testEnvironment{
		fields: map[string]Object{
			"NOME": &StringObject{Value: "Campo"},
		},
		variables: map[string]Object{
			"NOME": &StringObject{Value: "Variavel"},
		},
	}

	result := testEvalWithEnv(t, "NOME", env)
	strObj, ok := result.(*StringObject)
	if !ok {
		t.Fatalf("expected *StringObject, got %T", result)
	}
	if strObj.Value != "Variavel" {
		t.Fatalf("expected variable to shadow field, got %q", strObj.Value)
	}
}

func TestEvalVariableNotFoundFallsBackToField(t *testing.T) {
	env := &testEnvironment{
		fields: map[string]Object{
			"CODIGO": &NumberObject{Value: 123},
		},
		variables: map[string]Object{},
	}

	result := testEvalWithEnv(t, "CODIGO", env)
	numObj, ok := result.(*NumberObject)
	if !ok {
		t.Fatalf("expected *NumberObject, got %T", result)
	}
	if numObj.Value != 123 {
		t.Fatalf("expected Value=123, got %v", numObj.Value)
	}
}

func TestEvalIdentifierNotFound(t *testing.T) {
	l := NewLexer("NADA")
	p := NewParser(l)
	exp := p.ParseExpression()

	_, err := Eval(exp, &testEnvironment{})
	if err == nil {
		t.Fatal("expected error for missing identifier, got nil")
	}
}

func TestEvalAndTruthTable(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{".T. .AND. .T.", true},
		{".T. .AND. .F.", false},
		{".F. .AND. .T.", false},
		{".F. .AND. .F.", false},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		b, ok := result.(*BooleanObject)
		if !ok {
			t.Fatalf("input %q: expected *BooleanObject, got %T", tt.input, result)
		}
		if b.Value != tt.expected {
			t.Fatalf("input %q: expected %v, got %v", tt.input, tt.expected, b.Value)
		}
	}
}

func TestEvalOrTruthTable(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{".T. .OR. .T.", true},
		{".T. .OR. .F.", true},
		{".F. .OR. .T.", true},
		{".F. .OR. .F.", false},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		b, ok := result.(*BooleanObject)
		if !ok {
			t.Fatalf("input %q: expected *BooleanObject, got %T", tt.input, result)
		}
		if b.Value != tt.expected {
			t.Fatalf("input %q: expected %v, got %v", tt.input, tt.expected, b.Value)
		}
	}
}

func TestEvalAndShortCircuit(t *testing.T) {
	env := &testEnvironment{fields: map[string]Object{}}

	l := NewLexer(".F. .AND. INEXISTENTE")
	p := NewParser(l)
	exp := p.ParseExpression()

	result, err := Eval(exp, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := result.(*BooleanObject)
	if !ok {
		t.Fatalf("expected *BooleanObject, got %T", result)
	}
	if b.Value != false {
		t.Fatalf("expected false, got %v", b.Value)
	}
}

func TestEvalOrShortCircuit(t *testing.T) {
	env := &testEnvironment{fields: map[string]Object{}}

	l := NewLexer(".T. .OR. INEXISTENTE")
	p := NewParser(l)
	exp := p.ParseExpression()

	result, err := Eval(exp, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := result.(*BooleanObject)
	if !ok {
		t.Fatalf("expected *BooleanObject, got %T", result)
	}
	if b.Value != true {
		t.Fatalf("expected true, got %v", b.Value)
	}
}

func TestEvalAndNoShortCircuitWhenLeftIsTrue(t *testing.T) {
	env := &testEnvironment{fields: map[string]Object{}}

	l := NewLexer(".T. .AND. INEXISTENTE")
	p := NewParser(l)
	exp := p.ParseExpression()

	_, err := Eval(exp, env)
	if err == nil {
		t.Fatal("expected error for missing field on right side, got nil")
	}
}

func TestEvalOrNoShortCircuitWhenLeftIsFalse(t *testing.T) {
	env := &testEnvironment{fields: map[string]Object{}}

	l := NewLexer(".F. .OR. INEXISTENTE")
	p := NewParser(l)
	exp := p.ParseExpression()

	_, err := Eval(exp, env)
	if err == nil {
		t.Fatal("expected error for missing field on right side, got nil")
	}
}

func TestEvalAndNonBooleanOperand(t *testing.T) {
	_, err := Eval(&BinaryExpression{
		Left:     &NumberLiteral{Value: 1},
		Operator: ".AND.",
		Right:    &BooleanLiteral{Value: true},
	}, nil)
	if err == nil {
		t.Fatal("expected error for non-boolean operand, got nil")
	}
}

func TestEvalOrNonBooleanOperand(t *testing.T) {
	_, err := Eval(&BinaryExpression{
		Left:     &BooleanLiteral{Value: false},
		Operator: ".OR.",
		Right:    &StringLiteral{Value: "x"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for non-boolean operand, got nil")
	}
}

func TestEvalUnsupportedBinaryOperator(t *testing.T) {
	_, err := Eval(&BinaryExpression{
		Left:     &NumberLiteral{Value: 1},
		Operator: "%",
		Right:    &NumberLiteral{Value: 2},
	}, nil)
	if err == nil {
		t.Fatal("expected error for unsupported binary operator, got nil")
	}
}

func TestEvalArithmeticOperators(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"1 + 2", 3},
		{"5 - 2", 3},
		{"3 * 4", 12},
		{"8 / 2", 4},
		{"2 + 3 * 4", 14},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		num, ok := result.(*NumberObject)
		if !ok {
			t.Fatalf("input %q: expected *NumberObject, got %T", tt.input, result)
		}
		if num.Value != tt.want {
			t.Fatalf("input %q: expected %v, got %v", tt.input, tt.want, num.Value)
		}
	}
}

func TestEvalStringEquality(t *testing.T) {
	env := &testEnvironment{
		fields: map[string]Object{
			"P.KEY": &StringObject{Value: "001"},
			"S.KEY": &StringObject{Value: "001"},
		},
	}

	result, err := Eval(&BinaryExpression{
		Left:     &Identifier{Name: "P.KEY"},
		Operator: "=",
		Right:    &Identifier{Name: "S.KEY"},
	}, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := result.(*BooleanObject)
	if !ok || !b.Value {
		t.Fatalf("expected true equality, got %#v", result)
	}
}

func TestEvalStringNotEqual(t *testing.T) {
	env := &testEnvironment{
		fields: map[string]Object{
			"P.KEY": &StringObject{Value: "001"},
			"S.KEY": &StringObject{Value: "002"},
		},
	}

	result, err := Eval(&BinaryExpression{
		Left:     &Identifier{Name: "P.KEY"},
		Operator: "<>",
		Right:    &Identifier{Name: "S.KEY"},
	}, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := result.(*BooleanObject)
	if !ok || !b.Value {
		t.Fatalf("expected true inequality, got %#v", result)
	}
}

func TestEvalEOFReturnsTrue(t *testing.T) {
	env := &testEnvironment{eof: true}

	result := testEvalWithEnv(t, "EOF()", env)
	b, ok := result.(*BooleanObject)
	if !ok {
		t.Fatalf("expected *BooleanObject, got %T", result)
	}
	if b.Value != true {
		t.Fatalf("expected true, got %v", b.Value)
	}
}

func TestEvalEOFReturnsFalse(t *testing.T) {
	env := &testEnvironment{eof: false}

	result := testEvalWithEnv(t, "EOF()", env)
	b, ok := result.(*BooleanObject)
	if !ok {
		t.Fatalf("expected *BooleanObject, got %T", result)
	}
	if b.Value != false {
		t.Fatalf("expected false, got %v", b.Value)
	}
}

func TestEvalEOFCaseInsensitive(t *testing.T) {
	env := &testEnvironment{eof: true}

	result := testEvalWithEnv(t, "eof()", env)
	b, ok := result.(*BooleanObject)
	if !ok {
		t.Fatalf("expected *BooleanObject, got %T", result)
	}
	if b.Value != true {
		t.Fatalf("expected true, got %v", b.Value)
	}
}

func TestEvalEOFArgCountError(t *testing.T) {
	l := NewLexer("EOF(1)")
	p := NewParser(l)
	exp := p.ParseExpression()

	_, err := Eval(exp, &testEnvironment{eof: true})
	if err == nil {
		t.Fatal("expected error for EOF() with arguments, got nil")
	}
}

