package expr

import "strings"

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

// readChar advances the cursor to the next character in the input string.
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

// peekChar returns the next character without advancing the cursor.
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// skipWhitespace discards whitespace characters.
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
	case '.':
		if isDotDelimitedStart(l) {
			lit := l.readDotDelimited()
			tok.Literal = lit
			switch strings.ToUpper(lit) {
			case ".T.", ".F.", ".Y.", ".N.":
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
		} else {
			tok.Type = ILLEGAL
			tok.Literal = "."
			l.readChar()
		}
	default:
		if isDigit(l.ch) {
			tok.Type = NUMBER
			tok.Literal = l.readNumber()
			return tok
		}
		if isLetter(l.ch) {
			tok.Type = IDENT
			tok.Literal = l.readExtendedIdentifier()
			return tok
		}
		// Characters not supported yet are marked as ILLEGAL in this step.
		tok.Type = ILLEGAL
		tok.Literal = string(l.ch)
		l.readChar()
	}

	return tok
}

// readString reads a string literal enclosed in single or double quotes.
func (l *Lexer) readString(quote byte) string {
	startCol := l.col
	l.readChar() // Skip the opening quote
	startPos := l.position

	for l.ch != quote && l.ch != 0 {
		l.readChar()
	}

	literal := l.input[startPos:l.position]
	if l.ch == quote {
		l.readChar() // Skip the closing quote
	} else {
		l.col = startCol // Track error at opening quote
	}
	return literal
}

// readBracketString reads a string literal enclosed in square brackets [like this].
func (l *Lexer) readBracketString() string {
	startCol := l.col
	l.readChar() // Skip the opening bracket '['
	startPos := l.position

	for l.ch != ']' && l.ch != 0 {
		l.readChar()
	}

	literal := l.input[startPos:l.position]
	if l.ch == ']' {
		l.readChar() // Skip the closing bracket ']'
	} else {
		l.col = startCol // Track error at opening bracket
	}
	return literal
}

// readNumber scans a numeric literal, supporting integers and floats.
// It avoids greedily consuming dots if they are part of logical operators (e.g. 5.AND.).
func (l *Lexer) readNumber() string {
	startPos := l.position
	for isDigit(l.ch) {
		l.readChar()
	}

	// Consume dot only if it is a decimal separator followed by a digit.
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar() // Consume the dot
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[startPos:l.position]
}

// readDotDelimited reads a dot-delimited sequence (.XXX.) from the input.
func (l *Lexer) readDotDelimited() string {
	startPos := l.position
	l.readChar() // consume '.'
	for (l.ch >= 'a' && l.ch <= 'z') || (l.ch >= 'A' && l.ch <= 'Z') {
		l.readChar()
	}
	if l.ch == '.' {
		l.readChar() // consume closing '.'
	}
	return l.input[startPos:l.position]
}

// isDotDelimitedStart checks if the current position begins a dot-delimited sequence.
func isDotDelimitedStart(l *Lexer) bool {
	if l.ch != '.' {
		return false
	}
	next := l.peekChar()
	return (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z')
}

// readIdentifier reads a sequence of letters and digits starting at the current position.
func (l *Lexer) readIdentifier() string {
	startPos := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[startPos:l.position]
}

// readExtendedIdentifier reads dBase field names, including JOB:CODE suffixes and P./S. prefixes.
func (l *Lexer) readExtendedIdentifier() string {
	lit := l.readIdentifier()
	if l.ch == ':' {
		l.readChar()
		start := l.position
		for isLetter(l.ch) || isDigit(l.ch) {
			l.readChar()
		}
		lit += ":" + l.input[start:l.position]
	}
	if (lit == "P" || lit == "S" || lit == "p" || lit == "s") && l.ch == '.' && l.peekChar() != '.' {
		l.readChar()
		lit += "." + l.readFieldNameContinued()
	}
	return lit
}

func (l *Lexer) readFieldNameContinued() string {
	startPos := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == ':' {
		l.readChar()
	}
	return l.input[startPos:l.position]
}

// isLetter returns true if the byte represents an ASCII letter.
func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

// isDigit returns true if the byte represents an ASCII digit.
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
