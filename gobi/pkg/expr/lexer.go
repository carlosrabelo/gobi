package expr

// Lexer scans a dBase II expression input string and produces tokens.
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // 1-indexed line number
	col          int  // 1-indexed column number
}

// NewLexer creates and initializes a Lexer with the input expression string.
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

// NextToken returns the next lexical Token from the input expression.
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	tok.Line = l.line
	tok.Col = l.col

	switch l.ch {
	case 0:
		tok.Type = EOF
		tok.Literal = ""
	case '"', '\'':
		tok.Type = STRING
		tok.Literal = l.readString(l.ch)
	case '[':
		tok.Type = STRING
		tok.Literal = l.readBracketString()
	case '.':
		tok.Type = ILLEGAL
		tok.Literal = "."
		l.readChar()
	default:
		if isDigit(l.ch) {
			tok.Type = NUMBER
			tok.Literal = l.readNumber()
			return tok
		}
		if isLetter(l.ch) {
			tok.Type = IDENT
			tok.Literal = l.readIdentifier()
			return tok
		}
		tok.Type = ILLEGAL
		tok.Literal = string(l.ch)
		l.readChar()
	}

	return tok
}

func (l *Lexer) readString(quote byte) string {
	startCol := l.col
	l.readChar()
	startPos := l.position

	for l.ch != quote && l.ch != 0 {
		l.readChar()
	}

	literal := l.input[startPos:l.position]
	if l.ch == quote {
		l.readChar()
	} else {
		l.col = startCol
	}
	return literal
}

func (l *Lexer) readBracketString() string {
	startCol := l.col
	l.readChar()
	startPos := l.position

	for l.ch != ']' && l.ch != 0 {
		l.readChar()
	}

	literal := l.input[startPos:l.position]
	if l.ch == ']' {
		l.readChar()
	} else {
		l.col = startCol
	}
	return literal
}

func (l *Lexer) readNumber() string {
	startPos := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	return l.input[startPos:l.position]
}

func (l *Lexer) readIdentifier() string {
	startPos := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[startPos:l.position]
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
