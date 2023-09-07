package expr

// TokenType represents the category of a token in the dBase II expression language.
type TokenType string

const (
	// Special tokens
	EOF     TokenType = "EOF"
	ILLEGAL TokenType = "ILLEGAL"

	// Identifiers (fields, memory variables, functions)
	IDENT TokenType = "IDENT"

	// Literals
	NUMBER  TokenType = "NUMBER"  // e.g. 123, 45.67
	STRING  TokenType = "STRING"  // e.g. "hello", 'world', [brackets]
	LOGICAL TokenType = "LOGICAL" // .T., .F., .Y., .N.

	// Arithmetic Operators
	PLUS     TokenType = "+"
	MINUS    TokenType = "-"
	ASTERISK TokenType = "*"
	SLASH    TokenType = "/"

	// Relational Operators
	EQ  TokenType = "="
	NEQ TokenType = "<>" // or !=
	LT  TokenType = "<"
	GT  TokenType = ">"
	LTE TokenType = "<="
	GTE TokenType = ">="

	// Logical Operators
	AND TokenType = ".AND."
	OR  TokenType = ".OR."
	NOT TokenType = ".NOT."

	// Delimiters
	LPAREN TokenType = "("
	RPAREN TokenType = ")"
	COMMA  TokenType = ","
)

// Token represents a single lexical token in the input expression.
type Token struct {
	Type    TokenType
	Literal string
	Line    int // 1-indexed line number for error reporting
	Col     int // 1-indexed column number for error reporting
}

// String returns a string representation of the token for debugging.
func (t Token) String() string {
	return string(t.Type) + "(" + t.Literal + ")"
}
