package payments

// Config selects the active payment integration. Provider "mock" enables a
// sandbox provider for the card + usdt methods; "" / "log" leaves them disabled,
// so checkout stays wallet-only — the launch default. A real gateway registers
// itself here (a new adapter + one case in New), no order-flow change required.
type Config struct {
	Provider string
}

// Registry resolves a PaymentProvider by order payment method. An empty registry
// means no non-wallet method is enabled (wallet-only checkout).
type Registry struct {
	byMethod map[string]PaymentProvider
}

// New builds the registry from cfg.
func New(cfg Config) *Registry {
	byMethod := map[string]PaymentProvider{}
	switch cfg.Provider {
	case "mock":
		m := NewMockProvider()
		byMethod["card"] = m
		byMethod["usdt"] = m
	}
	return &Registry{byMethod: byMethod}
}

// Enabled reports whether a payment method has a provider wired.
func (r *Registry) Enabled(method string) bool {
	_, ok := r.byMethod[method]
	return ok
}

// For returns the provider for a method (ok=false when none is configured).
func (r *Registry) For(method string) (PaymentProvider, bool) {
	p, ok := r.byMethod[method]
	return p, ok
}
