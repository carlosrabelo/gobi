package repl

import (
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func TestParseStoreExpression(t *testing.T) {
	exp, err := parseStoreExpression("NUMBER + 9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp == nil {
		t.Fatal("expected parsed expression")
	}
}

func TestParseStoreExpressionInvalid(t *testing.T) {
	_, err := parseStoreExpression("NUMBER +")
	if err == nil || !strings.Contains(err.Error(), "Syntax error") {
		t.Fatalf("expected syntax error, got %v", err)
	}
}

func TestParseStoreMemvars(t *testing.T) {
	memvars, err := parseStoreMemvars("i, j, k")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(memvars) != 3 {
		t.Fatalf("expected 3 memvars, got %#v", memvars)
	}
}

func TestParseStoreMemvarsMissing(t *testing.T) {
	_, err := parseStoreMemvars("")
	if err == nil || !strings.Contains(err.Error(), "TO clause") {
		t.Fatalf("expected TO clause error, got %v", err)
	}
}

func TestObjectToStoredValueNumber(t *testing.T) {
	val, err := objectToStoredValue(&expr.NumberObject{Value: 12})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.(float64) != 12 {
		t.Fatalf("expected 12, got %v", val)
	}
}

func TestObjectToStoredValueString(t *testing.T) {
	val, err := objectToStoredValue(&expr.StringObject{Value: "HOWARD"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.(string) != "HOWARD" {
		t.Fatalf("expected HOWARD, got %v", val)
	}
}
