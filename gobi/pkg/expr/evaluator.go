package expr

import (
	"fmt"
)

// Environment provides access to external data during expression evaluation.
type Environment interface {
	GetField(name string) (Object, bool)
	GetVariable(name string) (Object, bool)
	CallFunction(name string, args []Object) (Object, error)
}

// Eval evaluates an AST expression node and returns a runtime Object.
func Eval(node Expression, env Environment) (Object, error) {
	if node == nil {
		return nil, fmt.Errorf("evaluator: cannot evaluate nil node")
	}

	switch n := node.(type) {
	case *NumberLiteral:
		return &NumberObject{Value: n.Value}, nil
	case *StringLiteral:
		return &StringObject{Value: n.Value}, nil
	case *BooleanLiteral:
		return &BooleanObject{Value: n.Value}, nil
	default:
		return nil, fmt.Errorf("evaluator: unsupported node type %T", n)
	}
}
