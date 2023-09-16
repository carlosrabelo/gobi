package expr

import (
	"fmt"
	"strconv"
	"strings"
)

// Environment provides access to external data during expression evaluation.
type Environment interface {
	GetField(name string) (Object, bool)
	GetVariable(name string) (Object, bool)
	CallFunction(name string, args []Object) (Object, error)
}

// Eval evaluates an AST expression node and returns a runtime Object.
// The env parameter is required only when the expression contains identifiers
// that reference fields or memory variables.
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
	case *BinaryExpression:
		return evalBinaryExpression(n, env)
	default:
		return nil, fmt.Errorf("evaluator: unsupported node type %T", n)
	}
}

func evalBinaryExpression(exp *BinaryExpression, env Environment) (Object, error) {
	op := strings.ToUpper(exp.Operator)

	switch op {
	case ".AND.":
		left, err := Eval(exp.Left, env)
		if err != nil {
			return nil, err
		}
		leftBool, err := toBoolean(left)
		if err != nil {
			return nil, err
		}
		if !leftBool {
			return &BooleanObject{Value: false}, nil
		}
		right, err := Eval(exp.Right, env)
		if err != nil {
			return nil, err
		}
		rightBool, err := toBoolean(right)
		if err != nil {
			return nil, err
		}
		return &BooleanObject{Value: rightBool}, nil

	case ".OR.":
		left, err := Eval(exp.Left, env)
		if err != nil {
			return nil, err
		}
		leftBool, err := toBoolean(left)
		if err != nil {
			return nil, err
		}
		if leftBool {
			return &BooleanObject{Value: true}, nil
		}
		right, err := Eval(exp.Right, env)
		if err != nil {
			return nil, err
		}
		rightBool, err := toBoolean(right)
		if err != nil {
			return nil, err
		}
		return &BooleanObject{Value: rightBool}, nil

	case "=", "<>", "<", ">", "<=", ">=":
		left, err := Eval(exp.Left, env)
		if err != nil {
			return nil, err
		}
		right, err := Eval(exp.Right, env)
		if err != nil {
			return nil, err
		}
		cmp, err := compareValues(left, right, exactMode(env))
		if err != nil {
			return nil, err
		}
		result, err := comparisonResult(cmp, op)
		if err != nil {
			return nil, err
		}
		return &BooleanObject{Value: result}, nil

	case "+", "-", "*", "/":
		left, err := Eval(exp.Left, env)
		if err != nil {
			return nil, err
		}
		right, err := Eval(exp.Right, env)
		if err != nil {
			return nil, err
		}
		leftNum, err := toNumber(left)
		if err != nil {
			return nil, err
		}
		rightNum, err := toNumber(right)
		if err != nil {
			return nil, err
		}
		switch op {
		case "+":
			return &NumberObject{Value: leftNum + rightNum}, nil
		case "-":
			return &NumberObject{Value: leftNum - rightNum}, nil
		case "*":
			return &NumberObject{Value: leftNum * rightNum}, nil
		case "/":
			if rightNum == 0 {
				return nil, fmt.Errorf("evaluator: division by zero")
			}
			return &NumberObject{Value: leftNum / rightNum}, nil
		default:
			return nil, fmt.Errorf("evaluator: unsupported arithmetic operator %q", exp.Operator)
		}

	default:
		return nil, fmt.Errorf("evaluator: unsupported binary operator %q", exp.Operator)
	}
}

// exactMode is a stub until SET EXACT lands; comparisons are exact.
func exactMode(env Environment) bool {
	return true
}

func compareValues(left, right Object, exact bool) (int, error) {
	leftNum, leftIsNum := left.(*NumberObject)
	rightNum, rightIsNum := right.(*NumberObject)
	if leftIsNum && rightIsNum {
		switch {
		case leftNum.Value < rightNum.Value:
			return -1, nil
		case leftNum.Value > rightNum.Value:
			return 1, nil
		default:
			return 0, nil
		}
	}

	leftStr := strings.ToUpper(strings.TrimSpace(left.String()))
	rightStr := strings.ToUpper(strings.TrimSpace(right.String()))
	// dBase II EXACT OFF: the comparison stops at the end of the right
	// operand, so any string matches a prefix of itself.
	if !exact && len(leftStr) > len(rightStr) {
		leftStr = leftStr[:len(rightStr)]
	}
	switch strings.Compare(leftStr, rightStr) {
	case -1:
		return -1, nil
	case 1:
		return 1, nil
	default:
		return 0, nil
	}
}

func comparisonResult(cmp int, op string) (bool, error) {
	switch op {
	case "=":
		return cmp == 0, nil
	case "<>":
		return cmp != 0, nil
	case "<":
		return cmp < 0, nil
	case ">":
		return cmp > 0, nil
	case "<=":
		return cmp <= 0, nil
	case ">=":
		return cmp >= 0, nil
	default:
		return false, fmt.Errorf("evaluator: unsupported comparison operator %q", op)
	}
}

func toBoolean(obj Object) (bool, error) {
	switch o := obj.(type) {
	case *BooleanObject:
		return o.Value, nil
	default:
		return false, fmt.Errorf("evaluator: expected boolean operand, got %s", obj.Type())
	}
}

func toNumber(obj Object) (float64, error) {
	switch o := obj.(type) {
	case *NumberObject:
		return o.Value, nil
	case *StringObject:
		trimmed := strings.TrimSpace(o.Value)
		if trimmed == "" {
			return 0, nil
		}
		val, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, fmt.Errorf("evaluator: expected numeric operand, got %s", obj.Type())
		}
		return val, nil
	default:
		return 0, fmt.Errorf("evaluator: expected numeric operand, got %s", obj.Type())
	}
}
