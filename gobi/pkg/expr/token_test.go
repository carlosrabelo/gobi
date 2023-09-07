package expr

import "testing"

func TestTokenString(t *testing.T) {
	tests := []struct {
		token    Token
		expected string
	}{
		{Token{Type: PLUS, Literal: "+"}, "+(+)"},
		{Token{Type: IDENT, Literal: "NOME"}, "IDENT(NOME)"},
		{Token{Type: NUMBER, Literal: "12.34"}, "NUMBER(12.34)"},
		{Token{Type: LOGICAL, Literal: ".T."}, "LOGICAL(.T.)"},
		{Token{Type: EOF, Literal: ""}, "EOF()"},
	}

	for _, tt := range tests {
		result := tt.token.String()
		if result != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, result)
		}
	}
}
