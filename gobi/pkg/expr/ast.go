package expr

import (
	"fmt"
	"strconv"
)

// Node is the base interface for all AST nodes.
type Node interface {
	String() string
}

// Expression represents a node that produces a value when evaluated.
type Expression interface {
	Node
	expressionNode()
}

// NumberLiteral represents a numeric literal (integer or float).
type NumberLiteral struct {
	Token Token
	Value float64
}

func (n *NumberLiteral) expressionNode() {}

func (n *NumberLiteral) String() string {
	return strconv.FormatFloat(n.Value, 'f', -1, 64)
}

// StringLiteral represents a string literal.
type StringLiteral struct {
	Token Token
	Value string
}

func (s *StringLiteral) expressionNode() {}

func (s *StringLiteral) String() string {
	return `"` + s.Value + `"`
}

// BooleanLiteral represents a logical constant (.T., .F., .Y., .N.).
type BooleanLiteral struct {
	Token Token
	Value bool
}

func (b *BooleanLiteral) expressionNode() {}

func (b *BooleanLiteral) String() string {
	if b.Value {
		return ".T."
	}
	return ".F."
}

// Identifier represents a named reference (field, memory variable, or function name).
type Identifier struct {
	Token Token
	Name  string
}

func (i *Identifier) expressionNode() {}

func (i *Identifier) String() string { return i.Name }

// UnaryExpression represents a unary operation (e.g., -x, .NOT. x).
type UnaryExpression struct {
	Token    Token
	Operator string
	Right    Expression
}

func (u *UnaryExpression) expressionNode() {}

func (u *UnaryExpression) String() string {
	return fmt.Sprintf("(%s%s)", u.Operator, u.Right.String())
}

// BinaryExpression represents a binary operation (e.g., a + b, x .AND. y).
type BinaryExpression struct {
	Left     Expression
	Token    Token
	Operator string
	Right    Expression
}

func (b *BinaryExpression) expressionNode() {}

func (b *BinaryExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left.String(), b.Operator, b.Right.String())
}

// CallExpression represents a function call (e.g., TRIM(x), SUBSTR(s, 1, 3)).
type CallExpression struct {
	Token     Token
	Function  Identifier
	Arguments []Expression
}

func (c *CallExpression) expressionNode() {}

func (c *CallExpression) String() string {
	args := ""
	for i, arg := range c.Arguments {
		if i > 0 {
			args += ", "
		}
		args += arg.String()
	}
	return fmt.Sprintf("%s(%s)", c.Function.String(), args)
}
