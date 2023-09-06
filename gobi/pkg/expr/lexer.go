package expr

import "strings"

// Lexer turns one expression string into tokens.
type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
	line         int
	col          int
}

// NewLexer creates a lexer over input. Line and column start at 1.
func NewLexer(input string) *Lexer {
	l := &Lexer{input: input, line: 1, col: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.col++
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		if l.ch == '\n' {
			l.line++
			l.col = 0
		}
		l.readChar()
	}
}

// NextToken returns the next token. Whitespace is skipped.
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()
	tok := Token{Line: l.line, Col: l.col}

	switch l.ch {
	case 0:
		tok.Type = EOF
		tok.Literal = ""
	case '"', '\'':
		return l.lexQuotedString(l.ch, tok)
	case '[':
		return l.lexBracketString(tok)
	case '.':
		if isDotWordStart(l) {
			return l.lexDotWord(tok)
		}
		tok.Type = ILLEGAL
		tok.Literal = "."
		l.readChar()
	case '+':
		tok.Type = PLUS
		tok.Literal = "+"
		l.readChar()
	case '-':
		tok.Type = MINUS
		tok.Literal = "-"
		l.readChar()
	case '*':
		tok.Type = ASTERISK
		tok.Literal = "*"
		l.readChar()
	case '/':
		tok.Type = SLASH
		tok.Literal = "/"
		l.readChar()
	case '(':
		tok.Type = LPAREN
		tok.Literal = "("
		l.readChar()
	case ')':
		tok.Type = RPAREN
		tok.Literal = ")"
		l.readChar()
	case ',':
		tok.Type = COMMA
		tok.Literal = ","
		l.readChar()
	case '=':
		tok.Type = EQ
		tok.Literal = "="
		l.readChar()
	case '#':
		tok.Type = NEQ
		tok.Literal = "#"
		l.readChar()
	case '$':
		tok.Type = CONTAIN
		tok.Literal = "$"
		l.readChar()
	case '<':
		if l.peekChar() == '=' {
			tok.Type = LTE
			tok.Literal = "<="
			l.readChar()
			l.readChar()
		} else if l.peekChar() == '>' {
			tok.Type = NEQ
			tok.Literal = "<>"
			l.readChar()
			l.readChar()
		} else {
			tok.Type = LT
			tok.Literal = "<"
			l.readChar()
		}
	case '>':
		if l.peekChar() == '=' {
			tok.Type = GTE
			tok.Literal = ">="
			l.readChar()
			l.readChar()
		} else {
			tok.Type = GT
			tok.Literal = ">"
			l.readChar()
		}
	default:
		if isDigit(l.ch) {
			tok.Type = NUMBER
			tok.Literal = l.readNumber()
			return tok
		}
		tok.Type = ILLEGAL
		tok.Literal = string(l.ch)
		l.readChar()
	}
	return tok
}

func (l *Lexer) lexQuotedString(quote byte, tok Token) Token {
	l.readChar()
	start := l.position
	for l.ch != quote && l.ch != 0 {
		l.readChar()
	}
	lit := l.input[start:l.position]
	if l.ch == quote {
		l.readChar()
		tok.Type = STRING
		tok.Literal = lit
		return tok
	}
	tok.Type = ILLEGAL
	tok.Literal = lit
	return tok
}

func (l *Lexer) lexBracketString(tok Token) Token {
	l.readChar()
	start := l.position
	for l.ch != ']' && l.ch != 0 {
		l.readChar()
	}
	lit := l.input[start:l.position]
	if l.ch == ']' {
		l.readChar()
		tok.Type = STRING
		tok.Literal = lit
		return tok
	}
	tok.Type = ILLEGAL
	tok.Literal = lit
	return tok
}

func (l *Lexer) lexDotWord(tok Token) Token {
	lit := l.readDotWord()
	tok.Literal = lit
	switch strings.ToUpper(lit) {
	case ".T.", ".Y.":
		tok.Type = LOGICAL
	case ".F.", ".N.":
		tok.Type = LOGICAL
	case ".AND.":
		tok.Type = AND
	case ".OR.":
		tok.Type = OR
	case ".NOT.":
		tok.Type = NOT
	default:
		tok.Type = ILLEGAL
	}
	return tok
}

func (l *Lexer) readNumber() string {
	start := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	// Decimal only when the next byte after '.' is a digit (not .AND.).
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	return l.input[start:l.position]
}

func (l *Lexer) readDotWord() string {
	start := l.position
	l.readChar()
	for isLetter(l.ch) {
		l.readChar()
	}
	if l.ch == '.' {
		l.readChar()
	}
	return l.input[start:l.position]
}

func isDotWordStart(l *Lexer) bool {
	next := l.peekChar()
	return isLetter(next)
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
