package expr

import "testing"

func TestOperatorsAndDotLogic(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Token
	}{
		{
			name: "arithmetic and parens",
			in:   "+ - * / ( ) ,",
			want: []Token{
				{Type: PLUS, Literal: "+"},
				{Type: MINUS, Literal: "-"},
				{Type: ASTERISK, Literal: "*"},
				{Type: SLASH, Literal: "/"},
				{Type: LPAREN, Literal: "("},
				{Type: RPAREN, Literal: ")"},
				{Type: COMMA, Literal: ","},
				{Type: EOF},
			},
		},
		{
			name: "comparisons and contain",
			in:   `= <> # < > <= >= $`,
			want: []Token{
				{Type: EQ, Literal: "="},
				{Type: NEQ, Literal: "<>"},
				{Type: NEQ, Literal: "#"},
				{Type: LT, Literal: "<"},
				{Type: GT, Literal: ">"},
				{Type: LTE, Literal: "<="},
				{Type: GTE, Literal: ">="},
				{Type: CONTAIN, Literal: "$"},
				{Type: EOF},
			},
		},
		{
			name: "NOT AND OR chain",
			in:   ".NOT. .T. .AND. .F. .OR. .Y.",
			want: []Token{
				{Type: NOT, Literal: ".NOT."},
				{Type: LOGICAL, Literal: ".T."},
				{Type: AND, Literal: ".AND."},
				{Type: LOGICAL, Literal: ".F."},
				{Type: OR, Literal: ".OR."},
				{Type: LOGICAL, Literal: ".Y."},
				{Type: EOF},
			},
		},
		{
			name: "bang is not not-equal",
			in:   "!=",
			want: []Token{
				{Type: ILLEGAL, Literal: "!"},
				{Type: EQ, Literal: "="},
				{Type: EOF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer(tt.in)
			for i, want := range tt.want {
				got := l.NextToken()
				if got.Type != want.Type || got.Literal != want.Literal {
					t.Fatalf("token %d: got {%s %q}, want {%s %q}", i, got.Type, got.Literal, want.Type, want.Literal)
				}
			}
		})
	}
}

func TestNumberStringLogical(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Token
	}{
		{
			name: "integer then AND",
			in:   "123.AND.",
			want: []Token{
				{Type: NUMBER, Literal: "123"},
				{Type: AND, Literal: ".AND."},
				{Type: EOF},
			},
		},
		{
			name: "float",
			in:   "45.67",
			want: []Token{
				{Type: NUMBER, Literal: "45.67"},
				{Type: EOF},
			},
		},
		{
			name: "double and single quotes",
			in:   `"hello" 'world'`,
			want: []Token{
				{Type: STRING, Literal: "hello"},
				{Type: STRING, Literal: "world"},
				{Type: EOF},
			},
		},
		{
			name: "bracket string",
			in:   "[brackets]",
			want: []Token{
				{Type: STRING, Literal: "brackets"},
				{Type: EOF},
			},
		},
		{
			name: "logicals case insensitive",
			in:   ".T. .f. .Y. .n.",
			want: []Token{
				{Type: LOGICAL, Literal: ".T."},
				{Type: LOGICAL, Literal: ".f."},
				{Type: LOGICAL, Literal: ".Y."},
				{Type: LOGICAL, Literal: ".n."},
				{Type: EOF},
			},
		},
		{
			name: "unclosed quote",
			in:   `"abc`,
			want: []Token{
				{Type: ILLEGAL, Literal: "abc"},
				{Type: EOF},
			},
		},
		{
			name: "unclosed bracket",
			in:   "[abc",
			want: []Token{
				{Type: ILLEGAL, Literal: "abc"},
				{Type: EOF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer(tt.in)
			for i, want := range tt.want {
				got := l.NextToken()
				if got.Type != want.Type || got.Literal != want.Literal {
					t.Fatalf("token %d: got {%s %q}, want {%s %q}", i, got.Type, got.Literal, want.Type, want.Literal)
				}
			}
		})
	}
}
