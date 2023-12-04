package symbols

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Registry stores global memory variable names and values for a Gobi session.
// Names are normalized to upper case on insert and lookup.
type Registry struct {
	vars map[string]interface{}
}

// NewRegistry returns an empty symbol registry.
func NewRegistry() *Registry {
	return &Registry{
		vars: make(map[string]interface{}),
	}
}

// NormalizeName converts a memory variable name to its canonical registry form.
func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("symbols: variable name cannot be empty")
	}
	return strings.ToUpper(name), nil
}

// Set stores value under name, creating or replacing the symbol.
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

// Get returns the value stored under name.
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

// Delete removes the named symbol. It reports whether the symbol existed.
func (r *Registry) Delete(name string) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("symbols: nil registry")
	}
	key, err := NormalizeName(name)
	if err != nil {
		return false, err
	}
	if r.vars == nil {
		return false, nil
	}
	if _, ok := r.vars[key]; !ok {
		return false, nil
	}
	delete(r.vars, key)
	return true, nil
}

// Clear removes every symbol from the registry.
func (r *Registry) Clear() {
	if r == nil {
		return
	}
	r.vars = make(map[string]interface{})
}

// Len returns the number of symbols currently stored.
func (r *Registry) Len() int {
	if r == nil || r.vars == nil {
		return 0
	}
	return len(r.vars)
}

// Names returns the stored symbol names sorted in ascending order.
func (r *Registry) Names() []string {
	if r == nil || len(r.vars) == 0 {
		return nil
	}
	names := make([]string, 0, len(r.vars))
	for name := range r.vars {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ValidateName reports whether name is acceptable for a memory variable.
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("symbols: variable name cannot be empty")
	}
	first := rune(name[0])
	if !unicode.IsLetter(first) && first != '_' {
		return fmt.Errorf("symbols: variable name must start with a letter")
	}
	for _, ch := range name {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' {
			continue
		}
		return fmt.Errorf("symbols: invalid character in variable name: %q", string(ch))
	}
	return nil
}
