package processing

import "fmt"

// Registry holds all registered algorithm processors.
type Registry struct {
	processors map[string]Processor
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		processors: make(map[string]Processor),
	}
}

// Register adds a processor to the registry.
func (r *Registry) Register(p Processor) {
	r.processors[p.Name()] = p
}

// Get returns the processor for the given algorithm name.
func (r *Registry) Get(name string) (Processor, error) {
	p, ok := r.processors[name]
	if !ok {
		return nil, fmt.Errorf("unknown algorithm: %s", name)
	}
	return p, nil
}

// Registered returns the list of registered algorithm names.
func (r *Registry) Registered() []string {
	names := make([]string, 0, len(r.processors))
	for n := range r.processors {
		names = append(names, n)
	}
	return names
}
