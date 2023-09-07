package expr

import "testing"

func TestParseLiteralsAndIdents(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		kind string
	}{
		{name: "integer", in: "42", want: "42", kind: "number"},
		{name: "float", in: "45.67", want: "45.67", kind: "number"},
		{name: "string", in: `"hello"`, want: "hello", kind: "string"},
		{name: "true Y", in: ".Y.", want: ".T.", kind: "logical"},
		{name: "false N", in: ".N.", want: ".F.", kind: "logical"},
		{name: "ident", in: "NAME", want: "NAME", kind: "ident"},
		{name: "alias field", in: "CUSTOMER->NAME", want: "CUSTOMER->NAME", kind: "ident"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errs := Parse(tt.in)
			if len(errs) != 0 {
				t.Fatalf("errors: %v", errs)
			}
			if got == nil {
				t.Fatal("nil expression")
			}
			if got.String() != tt.want {
				t.Fatalf("String: got %q, want %q", got.String(), tt.want)
			}
			switch tt.kind {
			case "number":
				if _, ok := got.(*NumberNode); !ok {
					t.Fatalf("got %T", got)
				}
			case "string":
				if _, ok := got.(*StringNode); !ok {
					t.Fatalf("got %T", got)
				}
			case "logical":
				if _, ok := got.(*LogicalNode); !ok {
					t.Fatalf("got %T", got)
				}
			case "ident":
				if _, ok := got.(*IdentNode); !ok {
					t.Fatalf("got %T", got)
				}
			}
		})
	}
}

func TestParseRejectsTrailingToken(t *testing.T) {
	_, errs := Parse("1 2")
	if len(errs) == 0 {
		t.Fatal("expected error for second expression")
	}
}

func TestParsePrecedence(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "left assoc subtract", in: "1-2-3", want: "((1 - 2) - 3)"},
		{name: "NOT binds over AND", in: ".NOT. .T. .AND. .F.", want: "((.NOT. .T.) .AND. .F.)"},
		{name: "compare over AND", in: "1 = 2 .AND. 3 = 4", want: "((1 = 2) .AND. (3 = 4))"},
		{name: "product over sum", in: "1+2*3", want: "(1 + (2 * 3))"},
		{name: "parens", in: "(1+2)*3", want: "((1 + 2) * 3)"},
		{name: "contain same as compare", in: `"A" $ "AB" .AND. .T.`, want: `((A $ AB) .AND. .T.)`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errs := Parse(tt.in)
			if len(errs) != 0 {
				t.Fatalf("errors: %v", errs)
			}
			if got == nil {
				t.Fatal("nil expression")
			}
			if got.String() != tt.want {
				t.Fatalf("got %q, want %q", got.String(), tt.want)
			}
		})
	}
}
