package expr

import "testing"

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
