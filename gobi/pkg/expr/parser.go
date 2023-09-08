package expr

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	_ int = iota
	precLowest
	precOr
	precAnd
	precCompare
	precSum
	precProduct
	precPrefix
	precCall
)

var precedences = map[TokenType]int{
	OR:       precOr,
	AND:      precAnd,
	EQ:       precCompare,
	NEQ:      precCompare,
	LT:       precCompare,
	GT:       precCompare,
	LTE:      precCompare,
	GTE:      precCompare,
	CONTAIN:  precCompare,
	PLUS:     precSum,
	MINUS:    precSum,
	ASTERISK: precProduct,
	SLASH:    precProduct,
	LPAREN:   precCall,
}

type (
	prefixParseFn func() Expression
	infixParseFn  func(Expression) Expression
)

// Parser builds an expression tree from tokens.
type Parser struct {
	l      *Lexer
	errors []string

	curToken  Token
	peekToken Token

	prefixFns map[TokenType]prefixParseFn
	infixFns  map[TokenType]infixParseFn
}

// Parse reads one expression from input and stops at EOF.
func Parse(input string) (Expression, []string) {
	p := newParser(input)
	expr := p.parseExpression(precLowest)
	if p.peekToken.Type != EOF {
		p.errors = append(p.errors, fmt.Sprintf("unexpected token %s", p.peekToken.Type))
	}
	return expr, p.errors
}

func newParser(input string) *Parser {
	p := &Parser{
		l:         NewLexer(input),
		prefixFns: map[TokenType]prefixParseFn{},
		infixFns:  map[TokenType]infixParseFn{},
	}
	p.registerPrefix(NUMBER, p.parseNumber)
	p.registerPrefix(STRING, p.parseString)
	p.registerPrefix(LOGICAL, p.parseLogical)
	p.registerPrefix(IDENT, p.parseIdent)
	p.registerPrefix(LPAREN, p.parseGrouped)
	p.registerPrefix(MINUS, p.parsePrefix)
	p.registerPrefix(NOT, p.parsePrefix)

	for _, t := range []TokenType{
		PLUS, MINUS, ASTERISK, SLASH,
		EQ, NEQ, LT, GT, LTE, GTE, CONTAIN,
		AND, OR,
	} {
		p.registerInfix(t, p.parseInfix)
	}
	p.registerInfix(LPAREN, p.parseCall)

	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) registerPrefix(t TokenType, fn prefixParseFn) {
	p.prefixFns[t] = fn
}

func (p *Parser) registerInfix(t TokenType, fn infixParseFn) {
	p.infixFns[t] = fn
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) peekPrecedence() int {
	if pr, ok := precedences[p.peekToken.Type]; ok {
		return pr
	}
	return precLowest
}

func (p *Parser) curPrecedence() int {
	if pr, ok := precedences[p.curToken.Type]; ok {
		return pr
	}
	return precLowest
}

func (p *Parser) parseExpression(precedence int) Expression {
	prefix := p.prefixFns[p.curToken.Type]
	if prefix == nil {
		p.errors = append(p.errors, fmt.Sprintf("unexpected token %s", p.curToken.Type))
		return nil
	}
	left := prefix()

	for p.peekToken.Type != EOF && precedence < p.peekPrecedence() {
		infix := p.infixFns[p.peekToken.Type]
		if infix == nil {
			return left
		}
		p.nextToken()
		left = infix(left)
	}
	return left
}

func (p *Parser) parseNumber() Expression {
	v, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("bad number %q", p.curToken.Literal))
		return nil
	}
	return &NumberNode{Value: v}
}

func (p *Parser) parseString() Expression {
	return &StringNode{Value: p.curToken.Literal}
}

func (p *Parser) parseLogical() Expression {
	upper := strings.ToUpper(p.curToken.Literal)
	return &LogicalNode{Value: upper == ".T." || upper == ".Y."}
}

func (p *Parser) parseIdent() Expression {
	return &IdentNode{Name: p.curToken.Literal}
}

func (p *Parser) parsePrefix() Expression {
	op := p.curToken.Literal
	p.nextToken()
	return &UnaryNode{Op: op, Operand: p.parseExpression(precPrefix)}
}

func (p *Parser) parseInfix(left Expression) Expression {
	op := p.curToken.Literal
	prec := p.curPrecedence()
	p.nextToken()
	right := p.parseExpression(prec)
	return &BinaryNode{Left: left, Op: op, Right: right}
}

func (p *Parser) parseGrouped() Expression {
	p.nextToken()
	expr := p.parseExpression(precLowest)
	if p.peekToken.Type != RPAREN {
		p.errors = append(p.errors, "expected )")
		return expr
	}
	p.nextToken()
	return expr
}

func (p *Parser) parseCall(left Expression) Expression {
	ident, ok := left.(*IdentNode)
	if !ok {
		p.errors = append(p.errors, "call requires identifier")
		return left
	}
	args := []Expression{}
	if p.peekToken.Type == RPAREN {
		p.nextToken()
		return &CallNode{Callee: ident.Name, Args: args}
	}
	p.nextToken()
	for {
		if p.curToken.Type == COMMA || p.curToken.Type == RPAREN {
			p.errors = append(p.errors, "expected expression")
			if p.curToken.Type == RPAREN {
				return &CallNode{Callee: ident.Name, Args: args}
			}
			p.nextToken()
			continue
		}
		args = append(args, p.parseExpression(precLowest))
		if p.peekToken.Type != COMMA {
			break
		}
		p.nextToken()
		p.nextToken()
	}
	if p.peekToken.Type != RPAREN {
		p.errors = append(p.errors, "expected )")
		return &CallNode{Callee: ident.Name, Args: args}
	}
	p.nextToken()
	return &CallNode{Callee: ident.Name, Args: args}
}
