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
