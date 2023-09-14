package expr

import "fmt"

// ObjectType represents the runtime type of a value in the expression language.
type ObjectType string

const (
	StringType  ObjectType = "STRING"
	NumberType  ObjectType = "NUMBER"
	BooleanType ObjectType = "BOOLEAN"
)

// Object is the runtime value interface returned by the evaluator.
type Object interface {
	Type() ObjectType
	String() string
}

// StringObject wraps a string value.
type StringObject struct {
	Value string
}

func (o *StringObject) Type() ObjectType { return StringType }

func (o *StringObject) String() string { return o.Value }

// NumberObject wraps a float64 value.
type NumberObject struct {
	Value float64
}

func (o *NumberObject) Type() ObjectType { return NumberType }

func (o *NumberObject) String() string {
	s := fmt.Sprintf("%v", o.Value)
	return s
}

// BooleanObject wraps a bool value.
type BooleanObject struct {
	Value bool
}

func (o *BooleanObject) Type() ObjectType { return BooleanType }

func (o *BooleanObject) String() string {
	if o.Value {
		return ".T."
	}
	return ".F."
}
