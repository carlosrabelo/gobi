package expr

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser builds an expression tree from tokens.
type Parser struct {
	l      *Lexer
	errors []string

	curToken  Token
	peekToken Token
}

// Parse reads one expression from input and stops at EOF.
func Parse(input string) (Expression, []string) {
	p := &Parser{l: NewLexer(input)}
	p.nextToken()
	p.nextToken()
	expr := p.parsePrimary()
	if p.curToken.Type != EOF {
		p.errors = append(p.errors, fmt.Sprintf("unexpected token %s", p.curToken.Type))
	}
	return expr, p.errors
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) parsePrimary() Expression {
	switch p.curToken.Type {
	case NUMBER:
		return p.parseNumber()
	case STRING:
		return p.parseString()
	case LOGICAL:
		return p.parseLogical()
	case IDENT:
		return p.parseIdent()
	case EOF:
		p.errors = append(p.errors, "expected expression")
		return nil
	default:
		p.errors = append(p.errors, fmt.Sprintf("unexpected token %s", p.curToken.Type))
		return nil
	}
}

func (p *Parser) parseNumber() Expression {
	v, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("bad number %q", p.curToken.Literal))
		return nil
	}
	n := &NumberNode{Value: v}
	p.nextToken()
	return n
}

func (p *Parser) parseString() Expression {
	n := &StringNode{Value: p.curToken.Literal}
	p.nextToken()
	return n
}

func (p *Parser) parseLogical() Expression {
	upper := strings.ToUpper(p.curToken.Literal)
	v := upper == ".T." || upper == ".Y."
	n := &LogicalNode{Value: v}
	p.nextToken()
	return n
}

func (p *Parser) parseIdent() Expression {
	n := &IdentNode{Name: p.curToken.Literal}
	p.nextToken()
	return n
}
