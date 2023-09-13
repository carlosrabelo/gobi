package expr

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	_ int = iota
	LOWEST
	OR_PREC
	AND_PREC
	NOT_PREC
	COMPARISON
	SUM
	PRODUCT
	PREFIX
	CALL
)

var precedences = map[TokenType]int{
	OR:       OR_PREC,
	AND:      AND_PREC,
	EQ:       COMPARISON,
	NEQ:      COMPARISON,
	LT:       COMPARISON,
	GT:       COMPARISON,
	LTE:      COMPARISON,
	GTE:      COMPARISON,
	PLUS:     SUM,
	MINUS:    SUM,
	ASTERISK: PRODUCT,
	SLASH:    PRODUCT,
	LPAREN:   CALL,
}

type (
	prefixParseFn func() Expression
	infixParseFn  func(Expression) Expression
)

// Parser transforms a stream of tokens from a Lexer into an AST.
type Parser struct {
	l      *Lexer
	errors []string

	curToken  Token
	peekToken Token

	prefixParseFns map[TokenType]prefixParseFn
	infixParseFns  map[TokenType]infixParseFn
}

// NewParser creates a Parser that reads tokens from the given Lexer.
func NewParser(l *Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	p.prefixParseFns = make(map[TokenType]prefixParseFn)
	p.registerPrefix(NUMBER, p.parseNumberLiteral)
	p.registerPrefix(STRING, p.parseStringLiteral)
	p.registerPrefix(LOGICAL, p.parseBooleanLiteral)
	p.registerPrefix(IDENT, p.parseIdentifier)
	p.registerPrefix(LPAREN, p.parseGroupedExpression)
	p.registerPrefix(MINUS, p.parsePrefixExpression)
	p.registerPrefix(NOT, p.parsePrefixExpression)

	p.infixParseFns = make(map[TokenType]infixParseFn)
	p.registerInfix(PLUS, p.parseInfixExpression)
	p.registerInfix(MINUS, p.parseInfixExpression)
	p.registerInfix(ASTERISK, p.parseInfixExpression)
	p.registerInfix(SLASH, p.parseInfixExpression)
	p.registerInfix(EQ, p.parseInfixExpression)
	p.registerInfix(NEQ, p.parseInfixExpression)
	p.registerInfix(LT, p.parseInfixExpression)
	p.registerInfix(GT, p.parseInfixExpression)
	p.registerInfix(LTE, p.parseInfixExpression)
	p.registerInfix(GTE, p.parseInfixExpression)
	p.registerInfix(AND, p.parseInfixExpression)
	p.registerInfix(OR, p.parseInfixExpression)
	p.registerInfix(LPAREN, p.parseCallExpression)

	p.nextToken()
	p.nextToken()

	return p
}

// Errors returns the list of parse errors encountered.
func (p *Parser) Errors() []string { return p.errors }

func (p *Parser) registerPrefix(t TokenType, fn prefixParseFn) {
	p.prefixParseFns[t] = fn
}

func (p *Parser) registerInfix(t TokenType, fn infixParseFn) {
	p.infixParseFns[t] = fn
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead (line %d, col %d)",
		t, p.peekToken.Type, p.peekToken.Line, p.peekToken.Col)
	p.errors = append(p.errors, msg)
}

func (p *Parser) noPrefixParseFnError(t TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found (line %d, col %d)",
		t, p.curToken.Line, p.curToken.Col)
	p.errors = append(p.errors, msg)
}

func (p *Parser) peekPrecedence() int {
	if pr, ok := precedences[p.peekToken.Type]; ok {
		return pr
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if pr, ok := precedences[p.curToken.Type]; ok {
		return pr
	}
	return LOWEST
}

// ParseExpression parses a single expression from the lexer input.
func (p *Parser) ParseExpression() Expression {
	return p.parseExpression(LOWEST)
}

func (p *Parser) parseExpression(precedence int) Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(EOF) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}
		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() Expression {
	return &Identifier{Token: p.curToken, Name: p.curToken.Literal}
}

func (p *Parser) parseNumberLiteral() Expression {
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as number (line %d, col %d)",
			p.curToken.Literal, p.curToken.Line, p.curToken.Col)
		p.errors = append(p.errors, msg)
		return nil
	}
	return &NumberLiteral{Token: p.curToken, Value: value}
}

func (p *Parser) parseStringLiteral() Expression {
	return &StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() Expression {
	upper := strings.ToUpper(p.curToken.Literal)
	value := upper == ".T." || upper == ".Y."
	return &BooleanLiteral{Token: p.curToken, Value: value}
}

func (p *Parser) parsePrefixExpression() Expression {
	exp := &UnaryExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}
	p.nextToken()
	exp.Right = p.parseExpression(PREFIX)
	return exp
}

func (p *Parser) parseInfixExpression(left Expression) Expression {
	exp := &BinaryExpression{
		Left:     left,
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}
	precedence := p.curPrecedence()
	p.nextToken()
	exp.Right = p.parseExpression(precedence)
	return exp
}

func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.expectPeek(RPAREN) {
		return nil
	}
	return exp
}

func (p *Parser) parseCallExpression(function Expression) Expression {
	ident, ok := function.(*Identifier)
	exp := &CallExpression{
		Token: p.curToken,
	}
	if ok {
		exp.Function = *ident
	}

	p.nextToken()

	if p.curTokenIs(RPAREN) {
		p.nextToken()
		return exp
	}

	exp.Arguments = p.parseCallArguments()

	if !p.expectPeek(RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) parseCallArguments() []Expression {
	var args []Expression
	args = append(args, p.parseExpression(LOWEST))

	for p.peekTokenIs(COMMA) {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	return args
}
