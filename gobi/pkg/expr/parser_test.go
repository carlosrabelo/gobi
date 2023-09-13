package expr

import (
	"strings"
	"testing"
)

func TestParseNumberLiteral(t *testing.T) {
	input := "42;"

	l := NewLexer(input)
	p := NewParser(l)
	exp := p.ParseExpression()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	number, ok := exp.(*NumberLiteral)
	if !ok {
		t.Fatalf("expected *NumberLiteral, got %T", exp)
	}
	if number.Value != 42 {
		t.Fatalf("expected Value=42, got %f", number.Value)
	}
	if number.Token.Literal != "42" {
		t.Fatalf("expected Token.Literal=42, got %q", number.Token.Literal)
	}
}

func TestParseStringLiteral(t *testing.T) {
	input := `"hello"`

	l := NewLexer(input)
	p := NewParser(l)
	exp := p.ParseExpression()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	str, ok := exp.(*StringLiteral)
	if !ok {
		t.Fatalf("expected *StringLiteral, got %T", exp)
	}
	if str.Value != "hello" {
		t.Fatalf("expected Value=hello, got %q", str.Value)
	}
}

func TestParseBooleanLiteral(t *testing.T) {
	tests := []struct {
		input string
		value bool
	}{
		{".T.", true},
		{".t.", true},
		{".Y.", true},
		{".F.", false},
		{".f.", false},
		{".N.", false},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		exp := p.ParseExpression()

		if len(p.Errors()) != 0 {
			t.Fatalf("input %q: parser errors: %v", tt.input, p.Errors())
		}

		b, ok := exp.(*BooleanLiteral)
		if !ok {
			t.Fatalf("input %q: expected *BooleanLiteral, got %T", tt.input, exp)
		}
		if b.Value != tt.value {
			t.Fatalf("input %q: expected Value=%v, got %v", tt.input, tt.value, b.Value)
		}
	}
}

func TestParseIdentifier(t *testing.T) {
	input := "NOME"

	l := NewLexer(input)
	p := NewParser(l)
	exp := p.ParseExpression()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	ident, ok := exp.(*Identifier)
	if !ok {
		t.Fatalf("expected *Identifier, got %T", exp)
	}
	if ident.Name != "NOME" {
		t.Fatalf("expected Name=NOME, got %q", ident.Name)
	}
}

func TestParsePrefixExpressions(t *testing.T) {
	prefixTests := []struct {
		input    string
		operator string
		value    interface{}
	}{
		{"-5", "-", float64(5)},
		{".NOT. .T.", ".NOT.", true},
		{".NOT. .F.", ".NOT.", false},
	}

	for _, tt := range prefixTests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		exp := p.ParseExpression()

		if len(p.Errors()) != 0 {
			t.Fatalf("input %q: parser errors: %v", tt.input, p.Errors())
		}

		ue, ok := exp.(*UnaryExpression)
		if !ok {
			t.Fatalf("input %q: expected *UnaryExpression, got %T", tt.input, exp)
		}
		if ue.Operator != tt.operator {
			t.Fatalf("input %q: expected Operator=%q, got %q", tt.input, tt.operator, ue.Operator)
		}
		testLiteralExpression(t, ue.Right, tt.value)
	}
}

func TestParseInfixExpressions(t *testing.T) {
	infixTests := []struct {
		input    string
		left     interface{}
		operator string
		right    interface{}
	}{
		{"1 + 2", float64(1), "+", float64(2)},
		{"1 - 2", float64(1), "-", float64(2)},
		{"1 * 2", float64(1), "*", float64(2)},
		{"1 / 2", float64(1), "/", float64(2)},
		{"1 = 2", float64(1), "=", float64(2)},
		{"1 <> 2", float64(1), "<>", float64(2)},
		{"1 < 2", float64(1), "<", float64(2)},
		{"1 > 2", float64(1), ">", float64(2)},
		{"1 <= 2", float64(1), "<=", float64(2)},
		{"1 >= 2", float64(1), ">=", float64(2)},
		{".T. .AND. .F.", true, ".AND.", false},
		{".T. .OR. .F.", true, ".OR.", false},
		{"x + y", "x", "+", "y"},
	}

	for _, tt := range infixTests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		exp := p.ParseExpression()

		if len(p.Errors()) != 0 {
			t.Fatalf("input %q: parser errors: %v", tt.input, p.Errors())
		}

		be, ok := exp.(*BinaryExpression)
		if !ok {
			t.Fatalf("input %q: expected *BinaryExpression, got %T", tt.input, exp)
		}
		testLiteralExpression(t, be.Left, tt.left)
		if be.Operator != tt.operator {
			t.Fatalf("input %q: expected Operator=%q, got %q", tt.input, tt.operator, be.Operator)
		}
		testLiteralExpression(t, be.Right, tt.right)
	}
}

func TestParseOperatorPrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1 + 2 * 3", "(1 + (2 * 3))"},
		{"1 * 2 + 3", "((1 * 2) + 3)"},
		{"-1 * 2", "((-1) * 2)"},
		{".NOT. .T. .AND. .F.", "((.NOT..T.) .AND. .F.)"},
		{"1 + 2 + 3", "((1 + 2) + 3)"},
		{"1 * 2 * 3", "((1 * 2) * 3)"},
		{"1 = 2 .AND. 3 = 4", "((1 = 2) .AND. (3 = 4))"},
		{"1 + 2 .OR. 3 * 4", "((1 + 2) .OR. (3 * 4))"},
		{"1 < 2 .AND. 3 > 4", "((1 < 2) .AND. (3 > 4))"},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		exp := p.ParseExpression()

		if len(p.Errors()) != 0 {
			t.Fatalf("input %q: parser errors: %v", tt.input, p.Errors())
		}

		actual := exp.String()
		if actual != tt.expected {
			t.Fatalf("input %q: expected %q, got %q", tt.input, tt.expected, actual)
		}
	}
}

func TestParseGroupedExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"(1 + 2) * 3", "((1 + 2) * 3)"},
		{"1 + (2 * 3)", "(1 + (2 * 3))"},
		{"(1 + 2) * (3 + 4)", "((1 + 2) * (3 + 4))"},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		exp := p.ParseExpression()

		if len(p.Errors()) != 0 {
			t.Fatalf("input %q: parser errors: %v", tt.input, p.Errors())
		}

		actual := exp.String()
		if actual != tt.expected {
			t.Fatalf("input %q: expected %q, got %q", tt.input, tt.expected, actual)
		}
	}
}

func TestParseCallExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"TRIM(x)", "TRIM(x)"},
		{"SUBSTR(\"hello\", 1, 3)", `SUBSTR("hello", 1, 3)`},
		{"UPPER(TRIM(x))", "UPPER(TRIM(x))"},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		exp := p.ParseExpression()

		if len(p.Errors()) != 0 {
			t.Fatalf("input %q: parser errors: %v", tt.input, p.Errors())
		}

		actual := exp.String()
		if actual != tt.expected {
			t.Fatalf("input %q: expected %q, got %q", tt.input, tt.expected, actual)
		}
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		input   string
		errText string
	}{
		{"(1 + 2", "expected next token to be )"},
		{".AND.", "no prefix parse function for .AND. found"},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		_ = p.ParseExpression()

		if len(p.Errors()) == 0 {
			t.Fatalf("input %q: expected errors, got none", tt.input)
		}

		found := false
		for _, err := range p.Errors() {
			if strings.Contains(err, tt.errText) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("input %q: expected error containing %q, got %v", tt.input, tt.errText, p.Errors())
		}
	}
}

func TestParseChainedComparisons(t *testing.T) {
	input := "1 < 2 < 3"

	l := NewLexer(input)
	p := NewParser(l)
	exp := p.ParseExpression()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	be, ok := exp.(*BinaryExpression)
	if !ok {
		t.Fatalf("expected *BinaryExpression, got %T", exp)
	}

	leftBe, ok := be.Left.(*BinaryExpression)
	if !ok {
		t.Fatalf("expected left *BinaryExpression, got %T", be.Left)
	}
	testLiteralExpression(t, leftBe.Left, float64(1))
	if leftBe.Operator != "<" {
		t.Fatalf("expected left operator '<', got %q", leftBe.Operator)
	}
	testLiteralExpression(t, leftBe.Right, float64(2))
	if be.Operator != "<" {
		t.Fatalf("expected operator '<', got %q", be.Operator)
	}
	testLiteralExpression(t, be.Right, float64(3))
}

func TestParseEmptyInput(t *testing.T) {
	input := ""

	l := NewLexer(input)
	p := NewParser(l)
	exp := p.ParseExpression()

	if exp != nil {
		t.Fatalf("expected nil for empty input, got %s", exp.String())
	}
	if len(p.Errors()) == 0 {
		t.Fatalf("expected errors for empty input")
	}
}

// test helpers

func testLiteralExpression(t *testing.T, exp Expression, expected interface{}) {
	t.Helper()
	switch v := expected.(type) {
	case float64:
		testNumberLiteral(t, exp, v)
	case string:
		testIdentifierLiteral(t, exp, v)
	case bool:
		testBooleanLiteral(t, exp, v)
	default:
		t.Fatalf("unexpected test value type: %T", expected)
	}
}

func testNumberLiteral(t *testing.T, exp Expression, value float64) {
	t.Helper()
	nl, ok := exp.(*NumberLiteral)
	if !ok {
		t.Fatalf("expected *NumberLiteral, got %T", exp)
	}
	if nl.Value != value {
		t.Fatalf("expected Value=%v, got %v", value, nl.Value)
	}
}

func testIdentifierLiteral(t *testing.T, exp Expression, name string) {
	t.Helper()
	ident, ok := exp.(*Identifier)
	if !ok {
		t.Fatalf("expected *Identifier, got %T", exp)
	}
	if ident.Name != name {
		t.Fatalf("expected Name=%q, got %q", name, ident.Name)
	}
}

func testBooleanLiteral(t *testing.T, exp Expression, value bool) {
	t.Helper()
	b, ok := exp.(*BooleanLiteral)
	if !ok {
		t.Fatalf("expected *BooleanLiteral, got %T", exp)
	}
	if b.Value != value {
		t.Fatalf("expected Value=%v, got %v", value, b.Value)
	}
}
