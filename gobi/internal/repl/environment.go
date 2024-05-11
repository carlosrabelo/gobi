package repl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

// replEnvironment adapts context.Context to act as an expr.Environment.
type replEnvironment struct {
	ctx *context.Context
}

// newReplEnvironment creates a new Environment tied to the active REPL context.
func newReplEnvironment(ctx *context.Context) expr.Environment {
	return &replEnvironment{ctx: ctx}
}

// ExactComparison reports the SET EXACT mode for string comparisons.
func (env *replEnvironment) ExactComparison() bool {
	return env.ctx.Config.Exact
}

func (env *replEnvironment) GetVariable(name string) (expr.Object, bool) {
	val, ok := env.ctx.Variables.Get(name)
	if !ok {
		return nil, false
	}
	switch v := val.(type) {
	case string:
		return &expr.StringObject{Value: v}, true
	case float64:
		return &expr.NumberObject{Value: v}, true
	case int:
		return &expr.NumberObject{Value: float64(v)}, true
	case bool:
		return &expr.BooleanObject{Value: v}, true
	case expr.Object:
		return v, true
	default:
		return &expr.StringObject{Value: fmt.Sprintf("%v", v)}, true
	}
}

func (env *replEnvironment) GetField(name string) (expr.Object, bool) {
	area := env.ctx.GetActiveArea()
	if area == nil || area.Table == nil || area.ActiveRecord == nil {
		return nil, false
	}

	fd, idx := area.Table.FieldByName(strings.ToUpper(name))
	if fd == nil {
		return nil, false
	}

	val, err := area.ActiveRecord.DecodeField(area.Table, idx)
	if err != nil {
		return nil, false
	}

	switch v := val.(type) {
	case string:
		return &expr.StringObject{Value: v}, true
	case float64:
		return &expr.NumberObject{Value: v}, true
	case bool:
		return &expr.BooleanObject{Value: v}, true
	default:
		return nil, false
	}
}

func (env *replEnvironment) CallFunction(name string, args []expr.Object) (expr.Object, error) {
	switch strings.ToUpper(name) {
	case "EOF":
		if len(args) != 0 {
			return nil, fmt.Errorf("EOF expects 0 arguments")
		}
		area := env.ctx.GetActiveArea()
		if area == nil || area.Table == nil {
			return &expr.BooleanObject{Value: true}, nil
		}
		isEOF := area.RecordNo >= int(area.Table.Header.RecordCount)
		return &expr.BooleanObject{Value: isEOF}, nil

	case "RECNO":
		if len(args) != 0 {
			return nil, fmt.Errorf("RECNO expects 0 arguments")
		}
		area := env.ctx.GetActiveArea()
		if area == nil || area.Table == nil {
			return &expr.NumberObject{Value: 0}, nil
		}
		// dBase II is 1-based for users
		return &expr.NumberObject{Value: float64(area.RecordNo + 1)}, nil

	case "DELETED":
		if len(args) != 0 {
			return nil, fmt.Errorf("DELETED expects 0 arguments")
		}
		area := env.ctx.GetActiveArea()
		if area == nil || area.ActiveRecord == nil {
			return &expr.BooleanObject{Value: false}, nil
		}
		return &expr.BooleanObject{Value: area.ActiveRecord.Deleted}, nil

	case "FOUND":
		if len(args) != 0 {
			return nil, fmt.Errorf("FOUND expects 0 arguments")
		}
		area := env.ctx.GetActiveArea()
		if area == nil {
			return &expr.BooleanObject{Value: false}, nil
		}
		return &expr.BooleanObject{Value: area.Found}, nil

	case "CHR":
		if len(args) != 1 {
			return nil, fmt.Errorf("CHR expects 1 argument")
		}
		numObj, ok := args[0].(*expr.NumberObject)
		if !ok {
			return nil, fmt.Errorf("CHR expects numeric argument")
		}
		n := int(numObj.Value)
		if n < 0 || n > 255 {
			return nil, fmt.Errorf("CHR argument out of range (0-255)")
		}
		// Single-byte semantics, as in dBase II's ASCII character set.
		return &expr.StringObject{Value: string([]byte{byte(n)})}, nil

	case "TRIM":
		if len(args) != 1 {
			return nil, fmt.Errorf("TRIM expects 1 argument")
		}
		return &expr.StringObject{Value: strings.TrimRight(args[0].String(), " ")}, nil

	case "UPPER":
		if len(args) != 1 {
			return nil, fmt.Errorf("UPPER expects 1 argument")
		}
		return &expr.StringObject{Value: strings.ToUpper(args[0].String())}, nil

	case "LOWER":
		if len(args) != 1 {
			return nil, fmt.Errorf("LOWER expects 1 argument")
		}
		return &expr.StringObject{Value: strings.ToLower(args[0].String())}, nil

	case "LEN":
		if len(args) != 1 {
			return nil, fmt.Errorf("LEN expects 1 argument")
		}
		return &expr.NumberObject{Value: float64(len(args[0].String()))}, nil

	case "VAL":
		if len(args) != 1 {
			return nil, fmt.Errorf("VAL expects 1 argument")
		}
		s := strings.TrimSpace(args[0].String())
		if s == "" {
			return &expr.NumberObject{Value: 0}, nil
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return &expr.NumberObject{Value: 0}, nil
		}
		return &expr.NumberObject{Value: v}, nil

	case "STR":
		if len(args) < 1 || len(args) > 3 {
			return nil, fmt.Errorf("STR expects 1 to 3 arguments")
		}
		numObj, ok := args[0].(*expr.NumberObject)
		if !ok {
			return nil, fmt.Errorf("STR expects numeric first argument")
		}
		length := 10
		decimals := 0
		if len(args) >= 2 {
			lObj, ok := args[1].(*expr.NumberObject)
			if !ok {
				return nil, fmt.Errorf("STR expects numeric second argument")
			}
			length = int(lObj.Value)
		}
		if len(args) == 3 {
			dObj, ok := args[2].(*expr.NumberObject)
			if !ok {
				return nil, fmt.Errorf("STR expects numeric third argument")
			}
			decimals = int(dObj.Value)
		}
		fmtStr := fmt.Sprintf("%%%d.%df", length, decimals)
		valStr := fmt.Sprintf(fmtStr, numObj.Value)
		return &expr.StringObject{Value: valStr}, nil

	case "SUBSTR":
		if len(args) != 3 {
			return nil, fmt.Errorf("SUBSTR expects 3 arguments")
		}
		strVal := args[0].String()
		startObj, ok := args[1].(*expr.NumberObject)
		if !ok {
			return nil, fmt.Errorf("SUBSTR expects numeric second argument")
		}
		lenObj, ok := args[2].(*expr.NumberObject)
		if !ok {
			return nil, fmt.Errorf("SUBSTR expects numeric third argument")
		}
		start := int(startObj.Value) - 1 // 1-based in dBase
		length := int(lenObj.Value)
		if start < 0 {
			start = 0
		}
		if start >= len(strVal) || length <= 0 {
			return &expr.StringObject{Value: ""}, nil
		}
		end := start + length
		if end > len(strVal) {
			end = len(strVal)
		}
		return &expr.StringObject{Value: strVal[start:end]}, nil

	default:
		return nil, fmt.Errorf("unknown function: %s", name)
	}
}
