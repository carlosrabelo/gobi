package expr

import "testing"

func TestASTString(t *testing.T) {
	tests := []struct {
		node     Node
		expected string
	}{
		{
			&NumberLiteral{Value: 123},
			"123",
		},
		{
			&NumberLiteral{Value: 45.67},
			"45.67",
		},
		{
			&StringLiteral{Value: "hello"},
			`"hello"`,
		},
		{
			&BooleanLiteral{Value: true},
			".T.",
		},
		{
			&BooleanLiteral{Value: false},
			".F.",
		},
		{
			&Identifier{Name: "NOME"},
			"NOME",
		},
		{
			&UnaryExpression{
				Operator: "-",
				Right:    &NumberLiteral{Value: 5},
			},
			"(-5)",
		},
		{
			&UnaryExpression{
				Operator: ".NOT.",
				Right:    &BooleanLiteral{Value: true},
			},
			"(.NOT..T.)",
		},
		{
			&BinaryExpression{
				Left:     &NumberLiteral{Value: 1},
				Operator: "+",
				Right:    &NumberLiteral{Value: 2},
			},
			"(1 + 2)",
		},
		{
			&BinaryExpression{
				Left:     &Identifier{Name: "x"},
				Operator: ".AND.",
				Right:    &Identifier{Name: "y"},
			},
			"(x .AND. y)",
		},
		{
			&CallExpression{
				Function:  Identifier{Name: "TRIM"},
				Arguments: []Expression{&Identifier{Name: "x"}},
			},
			"TRIM(x)",
		},
		{
			&CallExpression{
				Function: Identifier{Name: "SUBSTR"},
				Arguments: []Expression{
					&StringLiteral{Value: "hello"},
					&NumberLiteral{Value: 1},
					&NumberLiteral{Value: 3},
				},
			},
			`SUBSTR("hello", 1, 3)`,
		},
	}

	for i, tt := range tests {
		result := tt.node.String()
		if result != tt.expected {
			t.Errorf("tests[%d] - String() wrong. expected=%q, got=%q", i, tt.expected, result)
		}
	}
}

func TestASTExpressionNodeMarker(t *testing.T) {
	exprs := []Expression{
		&NumberLiteral{},
		&StringLiteral{},
		&BooleanLiteral{},
		&Identifier{},
		&UnaryExpression{},
		&BinaryExpression{},
		&CallExpression{},
	}

	if len(exprs) != 7 {
		t.Fatalf("expected 7 expression types, got %d", len(exprs))
	}
}
