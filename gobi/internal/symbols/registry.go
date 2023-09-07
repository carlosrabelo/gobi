package symbols

// Registry is a stub symbol table; the mature implementation lands with
// the memory-variable milestone.
type Registry struct{}

func NewRegistry() *Registry {
	return &Registry{}
}
