package expr

import (
	"strings"
	"testing"
)

type testEnv struct {
	mem    map[string]Value
	fields map[string]Value // key "ALIAS.NAME" or ".NAME" for active area
	exact  *bool
}

func (e *testEnv) Memory(name string) (Value, bool) {
	v, ok := e.mem[strings.ToUpper(name)]
	return v, ok
}

func (e *testEnv) Field(alias, name string) (Value, bool) {
	key := strings.ToUpper(alias) + "." + strings.ToUpper(name)
	v, ok := e.fields[key]
	return v, ok
}

func (e *testEnv) Call(string, []Value) (Value, error) {
	return Value{}, nil
}

func (e *testEnv) Exact() (bool, bool) {
	if e.exact == nil {
		return false, false
	}
	return *e.exact, true
}

func TestEvalLiterals(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		kind Kind
	}{
		{name: "integer", in: "42", want: "42", kind: KindNumber},
		{name: "float", in: "45.67", want: "45.67", kind: KindNumber},
		{name: "string", in: `"hello"`, want: "hello", kind: KindString},
		{name: "true", in: ".T.", want: ".T.", kind: KindLogical},
		{name: "false YN", in: ".N.", want: ".F.", kind: KindLogical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, errs := Parse(tt.in)
			if len(errs) != 0 {
				t.Fatalf("parse: %v", errs)
			}
			got, err := Eval(tree, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got.Kind != tt.kind {
				t.Fatalf("kind: got %v, want %v", got.Kind, tt.kind)
			}
			if got.String() != tt.want {
				t.Fatalf("String: got %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestEvalNilTree(t *testing.T) {
	_, err := Eval(nil, &testEnv{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEvalIdentNilEnv(t *testing.T) {
	tree, errs := Parse("NAME")
	if len(errs) != 0 {
		t.Fatalf("parse: %v", errs)
	}
	_, err := Eval(tree, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEvalMemoryThenField(t *testing.T) {
	env := &testEnv{
		mem: map[string]Value{
			"X": NumberVal(1),
		},
		fields: map[string]Value{
			".X":            NumberVal(2),
			".Y":            StringVal("field"),
			"CUSTOMER.NAME": StringVal("Acme"),
		},
	}

	t.Run("memory", func(t *testing.T) {
		got := mustEval(t, "X", env)
		if got.Num != 1 {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("memory wins over field", func(t *testing.T) {
		got := mustEval(t, "x", env)
		if got.Num != 1 {
			t.Fatalf("got %v, want memory 1", got)
		}
	})
	t.Run("field when no memory", func(t *testing.T) {
		got := mustEval(t, "Y", env)
		if got.Str != "field" {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("alias arrow field", func(t *testing.T) {
		got := mustEval(t, "customer->name", env)
		if got.Str != "Acme" {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("missing", func(t *testing.T) {
		tree, errs := Parse("MISSING")
		if len(errs) != 0 {
			t.Fatalf("parse: %v", errs)
		}
		_, err := Eval(tree, env)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestEvalArithmetic(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want float64
	}{
		{name: "add", in: "1+2", want: 3},
		{name: "sub", in: "5-3", want: 2},
		{name: "mul", in: "2*3", want: 6},
		{name: "div", in: "8/2", want: 4},
		{name: "product over sum", in: "1+2*3", want: 7},
		{name: "unary minus", in: "-5", want: -5},
		{name: "unary then product", in: "-2*3", want: -6},
		{name: "string coerce", in: `"3"+2`, want: 5},
		{name: "empty string is zero", in: `""+1`, want: 1},
		{name: "trim coerce", in: `" 4 "+1`, want: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mustEval(t, tt.in, nil)
			if got.Kind != KindNumber || got.Num != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvalDivByZero(t *testing.T) {
	tree, errs := Parse("1/0")
	if len(errs) != 0 {
		t.Fatalf("parse: %v", errs)
	}
	_, err := Eval(tree, nil)
	if err == nil {
		t.Fatal("expected division by zero")
	}
}

func TestEvalBadNumberString(t *testing.T) {
	tree, errs := Parse(`"abc"+1`)
	if len(errs) != 0 {
		t.Fatalf("parse: %v", errs)
	}
	_, err := Eval(tree, nil)
	if err == nil {
		t.Fatal("expected coerce error")
	}
}

func TestEvalCompareNumeric(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "lt", in: "1 < 2", want: true},
		{name: "gt", in: "3 > 1", want: true},
		{name: "eq", in: "2 = 2", want: true},
		{name: "neq hash", in: "1 # 2", want: true},
		{name: "neq angles", in: "1 <> 2", want: true},
		{name: "lte", in: "2 <= 2", want: true},
		{name: "gte", in: "2 >= 1", want: true},
		{name: "eq false", in: "1 = 2", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mustEval(t, tt.in, nil)
			if got.Kind != KindLogical || got.Log != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvalCompareExact(t *testing.T) {
	off := false
	on := true
	expr := `"Smith" = "Sm"`

	t.Run("exact off Smith equals Sm", func(t *testing.T) {
		got := mustEval(t, expr, &testEnv{exact: &off})
		if !got.Log {
			t.Fatalf("got %v, want .T.", got)
		}
	})
	t.Run("exact on full match", func(t *testing.T) {
		got := mustEval(t, expr, &testEnv{exact: &on})
		if got.Log {
			t.Fatalf("got %v, want .F.", got)
		}
	})
	t.Run("unreported mode is exact", func(t *testing.T) {
		got := mustEval(t, expr, &testEnv{})
		if got.Log {
			t.Fatalf("got %v, want .F.", got)
		}
	})
}

func mustEval(t *testing.T, in string, env Env) Value {
	t.Helper()
	tree, errs := Parse(in)
	if len(errs) != 0 {
		t.Fatalf("parse: %v", errs)
	}
	got, err := Eval(tree, env)
	if err != nil {
		t.Fatal(err)
	}
	return got
}
