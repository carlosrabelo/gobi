package expr

import (
	"fmt"
	"strconv"
	"strings"
)

// Expression is one node in a parsed expression tree.
type Expression interface {
	String() string
}

// NumberNode is a numeric literal.
type NumberNode struct {
	Value float64
}

func (n *NumberNode) String() string {
	return strconv.FormatFloat(n.Value, 'g', -1, 64)
}

// StringNode is a string literal.
type StringNode struct {
	Value string
}

func (n *StringNode) String() string {
	return n.Value
}

// LogicalNode is a logical literal.
type LogicalNode struct {
	Value bool
}

func (n *LogicalNode) String() string {
	if n.Value {
		return ".T."
	}
	return ".F."
}

// IdentNode is a field or memory-variable name as lexed.
type IdentNode struct {
	Name string
}

func (n *IdentNode) String() string {
	return n.Name
}

// UnaryNode is a prefix operator and its operand.
type UnaryNode struct {
	Op      string
	Operand Expression
}

func (n *UnaryNode) String() string {
	return fmt.Sprintf("(%s %s)", n.Op, n.Operand.String())
}

// BinaryNode is an infix operator with two operands.
type BinaryNode struct {
	Left  Expression
	Op    string
	Right Expression
}

func (n *BinaryNode) String() string {
	return fmt.Sprintf("(%s %s %s)", n.Left.String(), n.Op, n.Right.String())
}

// CallNode is a function call.
type CallNode struct {
	Callee string
	Args   []Expression
}

func (n *CallNode) String() string {
	parts := make([]string, len(n.Args))
	for i, a := range n.Args {
		parts[i] = a.String()
	}
	return fmt.Sprintf("%s(%s)", n.Callee, strings.Join(parts, ", "))
}
