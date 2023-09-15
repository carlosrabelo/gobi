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
	case *Identifier:
		if env == nil {
			return nil, fmt.Errorf("evaluator: cannot resolve identifier %q without environment", n.Name)
		}
		obj, ok := env.GetVariable(n.Name)
		if ok {
			return obj, nil
		}
		obj, ok = env.GetField(n.Name)
		if !ok {
			return nil, fmt.Errorf("evaluator: identifier %q not found", n.Name)
		}
		return obj, nil
	default:
		return nil, fmt.Errorf("evaluator: unsupported node type %T", n)
	}
}
