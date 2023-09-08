package expr

import "testing"

func TestLexerStringsAndNumbers(t *testing.T) {
	input := `  "hello"   'world'   [bracket string]  123   45.67  89.`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{STRING, "hello"},
		{STRING, "world"},
		{STRING, "bracket string"},
		{NUMBER, "123"},
		{NUMBER, "45.67"},
		{NUMBER, "89"},
		{ILLEGAL, "."}, // The dot without following digits is treated as illegal in this stage.
		{EOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q", i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexerLogicalConstants(t *testing.T) {
	input := `.T.  .F.  .Y.  .N.  .t.  .f.  .y.  .n.`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{LOGICAL, ".T."},
		{LOGICAL, ".F."},
		{LOGICAL, ".Y."},
		{LOGICAL, ".N."},
		{LOGICAL, ".t."},
		{LOGICAL, ".f."},
		{LOGICAL, ".y."},
		{LOGICAL, ".n."},
		{EOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q", i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexerNumberLookahead(t *testing.T) {
	input := "123.AND."

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{NUMBER, "123"},
		{AND, ".AND."},
		{EOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q", i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexerOperatorsAndParenthesis(t *testing.T) {
	input := `1 + 2 * 3 - 4 / 5 = 6 < 7 > 8 <= 9 >= 10 <> 11 ( 12 , 13 )`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{NUMBER, "1"},
		{PLUS, "+"},
		{NUMBER, "2"},
		{ASTERISK, "*"},
		{NUMBER, "3"},
		{MINUS, "-"},
		{NUMBER, "4"},
		{SLASH, "/"},
		{NUMBER, "5"},
		{EQ, "="},
		{NUMBER, "6"},
		{LT, "<"},
		{NUMBER, "7"},
		{GT, ">"},
		{NUMBER, "8"},
		{LTE, "<="},
		{NUMBER, "9"},
		{GTE, ">="},
		{NUMBER, "10"},
		{NEQ, "<>"},
		{NUMBER, "11"},
		{LPAREN, "("},
		{NUMBER, "12"},
		{COMMA, ","},
		{NUMBER, "13"},
		{RPAREN, ")"},
		{EOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q", i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexerLogicalOperators(t *testing.T) {
	input := `.NOT. x .AND. y .OR. z`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{NOT, ".NOT."},
		{IDENT, "x"},
		{AND, ".AND."},
		{IDENT, "y"},
		{OR, ".OR."},
		{IDENT, "z"},
		{EOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q", i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}
