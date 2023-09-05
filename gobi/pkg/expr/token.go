package expr

// TokenType is the kind of a lexical token.
type TokenType string

const (
	EOF     TokenType = "EOF"
	ILLEGAL TokenType = "ILLEGAL"

	NUMBER  TokenType = "NUMBER"
	STRING  TokenType = "STRING"
	LOGICAL TokenType = "LOGICAL"
	IDENT   TokenType = "IDENT"

	AND TokenType = "AND"
	OR  TokenType = "OR"
	NOT TokenType = "NOT"

	PLUS     TokenType = "PLUS"
	MINUS    TokenType = "MINUS"
	ASTERISK TokenType = "ASTERISK"
	SLASH    TokenType = "SLASH"

	EQ      TokenType = "EQ"
	NEQ     TokenType = "NEQ"
	LT      TokenType = "LT"
	GT      TokenType = "GT"
	LTE     TokenType = "LTE"
	GTE     TokenType = "GTE"
	CONTAIN TokenType = "CONTAIN"

	LPAREN TokenType = "LPAREN"
	RPAREN TokenType = "RPAREN"
	COMMA  TokenType = "COMMA"
)

// Token is one lexical unit with 1-based line and column.
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}
