package symbols

import (
	"fmt"
	"strings"
)

// Registry is a stub symbol table; the mature implementation lands with
// the memory-variable milestone.
type Registry struct {
	vars map[string]interface{}
}

func NewRegistry() *Registry {
	return &Registry{vars: make(map[string]interface{})}
}

func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("symbols: variable name cannot be empty")
	}
	return strings.ToUpper(name), nil
}

func (r *Registry) Set(name string, value interface{}) error {
	if r == nil {
		return fmt.Errorf("symbols: nil registry")
	}
	key, err := NormalizeName(name)
	if err != nil {
		return err
	}
	if r.vars == nil {
		r.vars = make(map[string]interface{})
	}
	r.vars[key] = value
	return nil
}

func (r *Registry) Get(name string) (interface{}, bool) {
	if r == nil || r.vars == nil {
		return nil, false
	}
	key, err := NormalizeName(name)
	if err != nil {
		return nil, false
	}
	val, ok := r.vars[key]
	return val, ok
}
