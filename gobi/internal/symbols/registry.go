package symbols

import (
	"fmt"
	"sort"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

const (
	// MaxName is the maximum length of a memory variable name in bytes.
	MaxName = 10
	// MaxVars is the maximum number of memory variables in a session.
	MaxVars = 256
)

// Registry is the session map of memory variables.
type Registry struct {
	vars map[string]expr.Value
}

// New returns an empty registry.
func New() *Registry {
	return &Registry{vars: map[string]expr.Value{}}
}

// Get returns the value for name (case-folded). ok is false if absent.
func (r *Registry) Get(name string) (expr.Value, bool) {
	if r == nil || r.vars == nil {
		return expr.Value{}, false
	}
	v, ok := r.vars[strings.ToUpper(name)]
	return v, ok
}

// Set stores v under name. Names are upper-cased. Errors on length or capacity.
func (r *Registry) Set(name string, v expr.Value) error {
	if r.vars == nil {
		r.vars = map[string]expr.Value{}
	}
	key := strings.ToUpper(strings.TrimSpace(name))
	if key == "" {
		return fmt.Errorf("empty variable name")
	}
	if len(key) > MaxName {
		return fmt.Errorf("variable name longer than %d characters", MaxName)
	}
	if _, exists := r.vars[key]; !exists && len(r.vars) >= MaxVars {
		return fmt.Errorf("too many memory variables")
	}
	r.vars[key] = v
	return nil
}

// Release removes one name. Missing names are ignored.
func (r *Registry) Release(name string) {
	if r == nil || r.vars == nil {
		return
	}
	delete(r.vars, strings.ToUpper(name))
}

// ReleaseAll clears the registry.
func (r *Registry) ReleaseAll() {
	if r == nil {
		return
	}
	r.vars = map[string]expr.Value{}
}

// Len returns the number of stored names.
func (r *Registry) Len() int {
	if r == nil || r.vars == nil {
		return 0
	}
	return len(r.vars)
}

// Names returns sorted upper-case names.
func (r *Registry) Names() []string {
	if r == nil || r.vars == nil {
		return nil
	}
	out := make([]string, 0, len(r.vars))
	for k := range r.vars {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Match reports whether name matches a LIKE pattern (? one char, * rest).
func Match(pattern, name string) bool {
	pattern = strings.ToUpper(pattern)
	name = strings.ToUpper(name)
	return matchPat(pattern, name)
}

func matchPat(pat, s string) bool {
	for len(pat) > 0 {
		switch pat[0] {
		case '*':
			pat = pat[1:]
			if pat == "" {
				return true
			}
			for i := 0; i <= len(s); i++ {
				if matchPat(pat, s[i:]) {
					return true
				}
			}
			return false
		case '?':
			if s == "" {
				return false
			}
			pat = pat[1:]
			s = s[1:]
		default:
			if s == "" || s[0] != pat[0] {
				return false
			}
			pat = pat[1:]
			s = s[1:]
		}
	}
	return s == ""
}

// ReleaseLike removes names matching pattern. like=true means LIKE, false EXCEPT.
func (r *Registry) ReleaseLike(pattern string, like bool) {
	if r == nil || r.vars == nil {
		return
	}
	for _, name := range r.Names() {
		m := Match(pattern, name)
		if like == m {
			delete(r.vars, name)
		}
	}
}

// Replace replaces the entire map (used by RESTORE).
func (r *Registry) Replace(other *Registry) {
	if other == nil {
		r.vars = map[string]expr.Value{}
		return
	}
	r.vars = make(map[string]expr.Value, len(other.vars))
	for k, v := range other.vars {
		r.vars[k] = v
	}
}

// Merge overwrites by name from other (RESTORE ADDITIVE).
func (r *Registry) Merge(other *Registry) error {
	if other == nil {
		return nil
	}
	for _, name := range other.Names() {
		v, _ := other.Get(name)
		if err := r.Set(name, v); err != nil {
			return err
		}
	}
	return nil
}

// Filter returns a new registry with names matching LIKE or EXCEPT.
func (r *Registry) Filter(pattern string, like bool) *Registry {
	out := New()
	for _, name := range r.Names() {
		m := Match(pattern, name)
		if like == m {
			v, _ := r.Get(name)
			_ = out.Set(name, v)
		}
	}
	return out
}
