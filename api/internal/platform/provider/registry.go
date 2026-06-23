package provider

// Registry resolves a numeric provider id to its adapter. Unknown or nil ids
// resolve to a StubProvider, so an api-mode order always has something to call
// and parks cleanly until a real adapter is registered.
type Registry struct {
	adapters map[int]Provider
}

// NewRegistry builds a Registry from zero or more adapters, keyed by ID().
func NewRegistry(adapters ...Provider) *Registry {
	m := make(map[int]Provider, len(adapters))
	for _, a := range adapters {
		m[a.ID()] = a
	}
	return &Registry{adapters: m}
}

// Resolve returns the adapter registered for id, or a StubProvider when id is
// nil or no adapter is registered for it.
func (r *Registry) Resolve(id *int) Provider {
	if id != nil {
		if a, ok := r.adapters[*id]; ok {
			return a
		}
		return NewStub(*id)
	}
	return NewStub(0)
}
