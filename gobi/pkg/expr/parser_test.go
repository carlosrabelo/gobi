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
