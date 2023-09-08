package expr

import (
	"fmt"
	"strings"
)

// Eval evaluates an expression tree using env for names and calls.
func Eval(expr Expression, env Env) (Value, error) {
	if expr == nil {
		return Value{}, fmt.Errorf("nil expression")
	}
	switch n := expr.(type) {
	case *NumberNode:
		return NumberVal(n.Value), nil
	case *StringNode:
		return StringVal(n.Value), nil
	case *LogicalNode:
		return LogicalVal(n.Value), nil
	case *IdentNode:
		return evalIdent(n.Name, env)
	default:
		return Value{}, fmt.Errorf("unsupported expression %T", expr)
	}
}

func evalIdent(name string, env Env) (Value, error) {
	if env == nil {
		return Value{}, fmt.Errorf("nil environment")
	}
	if i := strings.Index(name, "->"); i >= 0 {
		alias := name[:i]
		field := name[i+2:]
		if v, ok := env.Field(alias, field); ok {
			return v, nil
		}
		return Value{}, fmt.Errorf("unknown field %s", name)
	}
	if v, ok := env.Memory(name); ok {
		return v, nil
	}
	if v, ok := env.Field("", name); ok {
		return v, nil
	}
	return Value{}, fmt.Errorf("unknown name %s", name)
}
