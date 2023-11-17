package expr

import (
	"errors"
	"fmt"
	"strconv"
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
	case *UnaryNode:
		return evalUnary(n, env)
	case *BinaryNode:
		return evalBinary(n, env)
	case *CallNode:
		return evalCall(n, env)
	default:
		return Value{}, fmt.Errorf("unsupported expression %T", expr)
	}
}

func evalCall(n *CallNode, env Env) (Value, error) {
	if env == nil {
		return Value{}, fmt.Errorf("nil environment")
	}
	args := make([]Value, len(n.Args))
	for i, a := range n.Args {
		v, err := Eval(a, env)
		if err != nil {
			return Value{}, err
		}
		args[i] = v
	}
	return env.Call(n.Callee, args)
}

func evalUnary(n *UnaryNode, env Env) (Value, error) {
	switch strings.ToUpper(n.Op) {
	case "-":
		v, err := Eval(n.Operand, env)
		if err != nil {
			return Value{}, err
		}
		num, err := asNumber(v)
		if err != nil {
			return Value{}, err
		}
		return NumberVal(-num), nil
	case ".NOT.":
		v, err := Eval(n.Operand, env)
		if err != nil {
			return Value{}, err
		}
		b, err := asLogical(v)
		if err != nil {
			return Value{}, err
		}
		return LogicalVal(!b), nil
	default:
		return Value{}, fmt.Errorf("unsupported unary %s", n.Op)
	}
}

func evalBinary(n *BinaryNode, env Env) (Value, error) {
	op := strings.ToUpper(n.Op)
	switch op {
	case "+", "-", "*", "/":
		return evalArithmetic(n, env)
	case "=", "<>", "#", "<", ">", "<=", ">=":
		return evalCompare(n, env)
	case "$":
		return evalContain(n, env)
	case ".AND.", ".OR.":
		return evalLogic(n, env, op)
	default:
		return Value{}, fmt.Errorf("unsupported operator %s", n.Op)
	}
}

func evalContain(n *BinaryNode, env Env) (Value, error) {
	left, err := Eval(n.Left, env)
	if err != nil {
		return Value{}, err
	}
	right, err := Eval(n.Right, env)
	if err != nil {
		return Value{}, err
	}
	return LogicalVal(strings.Contains(right.String(), left.String())), nil
}

func evalLogic(n *BinaryNode, env Env, op string) (Value, error) {
	left, err := Eval(n.Left, env)
	if err != nil {
		return Value{}, err
	}
	lb, err := asLogical(left)
	if err != nil {
		return Value{}, err
	}
	if op == ".AND." && !lb {
		return LogicalVal(false), nil
	}
	if op == ".OR." && lb {
		return LogicalVal(true), nil
	}
	right, err := Eval(n.Right, env)
	if err != nil {
		return Value{}, err
	}
	rb, err := asLogical(right)
	if err != nil {
		return Value{}, err
	}
	return LogicalVal(rb), nil
}

func asLogical(v Value) (bool, error) {
	if v.Kind != KindLogical {
		return false, fmt.Errorf("expected logical")
	}
	return v.Log, nil
}

func evalCompare(n *BinaryNode, env Env) (Value, error) {
	left, err := Eval(n.Left, env)
	if err != nil {
		return Value{}, err
	}
	right, err := Eval(n.Right, env)
	if err != nil {
		return Value{}, err
	}
	var cmp int
	if left.Kind == KindNumber && right.Kind == KindNumber {
		switch {
		case left.Num < right.Num:
			cmp = -1
		case left.Num > right.Num:
			cmp = 1
		}
	} else {
		ls := strings.ToUpper(strings.TrimSpace(left.String()))
		rs := strings.ToUpper(strings.TrimSpace(right.String()))
		if !exactMode(env) && len(ls) > len(rs) {
			ls = ls[:len(rs)]
		}
		cmp = strings.Compare(ls, rs)
	}
	var ok bool
	switch n.Op {
	case "=":
		ok = cmp == 0
	case "<>", "#":
		ok = cmp != 0
	case "<":
		ok = cmp < 0
	case ">":
		ok = cmp > 0
	case "<=":
		ok = cmp <= 0
	case ">=":
		ok = cmp >= 0
	}
	return LogicalVal(ok), nil
}

func exactMode(env Env) bool {
	if env == nil {
		return true
	}
	on, ok := env.Exact()
	if !ok {
		return true
	}
	return on
}

func evalArithmetic(n *BinaryNode, env Env) (Value, error) {
	left, err := Eval(n.Left, env)
	if err != nil {
		return Value{}, err
	}
	right, err := Eval(n.Right, env)
	if err != nil {
		return Value{}, err
	}
	switch n.Op {
	case "+":
		if left.Kind == KindString && right.Kind != KindNumber {
			return StringVal(left.String() + right.String()), nil
		}
		if right.Kind == KindString && left.Kind != KindNumber {
			return StringVal(left.String() + right.String()), nil
		}
		a, err := asNumber(left)
		if err != nil {
			return Value{}, err
		}
		b, err := asNumber(right)
		if err != nil {
			return Value{}, err
		}
		return NumberVal(a + b), nil
	case "-":
		if left.Kind == KindString && right.Kind == KindString {
			return StringVal(stringMinus(left.Str, right.Str)), nil
		}
		a, err := asNumber(left)
		if err != nil {
			return Value{}, err
		}
		b, err := asNumber(right)
		if err != nil {
			return Value{}, err
		}
		return NumberVal(a - b), nil
	case "*":
		a, err := asNumber(left)
		if err != nil {
			return Value{}, err
		}
		b, err := asNumber(right)
		if err != nil {
			return Value{}, err
		}
		return NumberVal(a * b), nil
	case "/":
		a, err := asNumber(left)
		if err != nil {
			return Value{}, err
		}
		b, err := asNumber(right)
		if err != nil {
			return Value{}, err
		}
		if b == 0 {
			return Value{}, fmt.Errorf("division by zero")
		}
		return NumberVal(a / b), nil
	default:
		return Value{}, fmt.Errorf("unsupported operator %s", n.Op)
	}
}

// stringMinus concatenates and moves trailing spaces of the left string to the end.
func stringMinus(left, right string) string {
	trimmed := strings.TrimRight(left, " ")
	spaces := left[len(trimmed):]
	return trimmed + right + spaces
}

func asNumber(v Value) (float64, error) {
	switch v.Kind {
	case KindNumber:
		return v.Num, nil
	case KindString:
		s := strings.TrimSpace(v.Str)
		if s == "" {
			return 0, nil
		}
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("not a number %q", v.Str)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("expected number")
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
	v, err := env.Call(name, nil)
	if err == nil {
		return v, nil
	}
	if errors.Is(err, ErrUnknownFunction) {
		return Value{}, fmt.Errorf("unknown name %s", name)
	}
	return Value{}, err
}
