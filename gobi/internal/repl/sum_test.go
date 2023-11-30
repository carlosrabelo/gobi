package repl

import (
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func TestParseSumExpressionsSingleField(t *testing.T) {
	expressions, err := parseAggregateExpressions("AGE", "SUM")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expressions) != 1 {
		t.Fatalf("expected 1 expression, got %d", len(expressions))
	}
}

func TestParseSumExpressionsMultiple(t *testing.T) {
	expressions, err := parseAggregateExpressions("AGE, AGE * 2", "SUM")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expressions) != 2 {
		t.Fatalf("expected 2 expressions, got %d", len(expressions))
	}
}

func TestParseSumExpressionsWithScope(t *testing.T) {
	expressions, err := parseAggregateExpressions("AGE ALL", "SUM")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expressions) != 1 {
		t.Fatalf("expected 1 expression, got %d", len(expressions))
	}
}

func TestParseSumExpressionsMissing(t *testing.T) {
	_, err := parseAggregateExpressions("", "SUM")
	if err == nil || !strings.Contains(err.Error(), "numeric expression") {
		t.Fatalf("expected missing expression error, got %v", err)
	}
}

func TestParseSumExpressionsTooMany(t *testing.T) {
	_, err := parseAggregateExpressions("A, B, C, D, E, F", "SUM")
	if err == nil || !strings.Contains(err.Error(), "5 FIELDS") {
		t.Fatalf("expected max fields error, got %v", err)
	}
}

func TestParseSumMemvars(t *testing.T) {
	memvars, err := parseAggregateMemvars("TOTAL, DOUBLE", 2, "SUM")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(memvars) != 2 || memvars[0] != "TOTAL" || memvars[1] != "DOUBLE" {
		t.Fatalf("unexpected memvars: %#v", memvars)
	}
}

func TestParseSumMemvarsMismatch(t *testing.T) {
	_, err := parseAggregateMemvars("TOTAL", 2, "SUM")
	if err == nil || !strings.Contains(err.Error(), "memory variable") {
		t.Fatalf("expected memvar mismatch error, got %v", err)
	}
}

func TestEvalNumericExpression(t *testing.T) {
	env := newReplEnvironment(testCtx())
	exp := &expr.NumberLiteral{Value: 12.5}

	val, err := evalNumericExpression(env, exp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 12.5 {
		t.Fatalf("expected 12.5, got %v", val)
	}
}

func TestEvalNumericExpressionNonNumeric(t *testing.T) {
	env := newReplEnvironment(testCtx())
	exp := &expr.StringLiteral{Value: "abc"}

	_, err := evalNumericExpression(env, exp)
	if err == nil || !strings.Contains(err.Error(), "non-numeric") {
		t.Fatalf("expected non-numeric error, got %v", err)
	}
}
