// Package tron is the on-chain payment-detection port for USDT (TRC20)
// deposits together with its adapters (TronGrid, stub). The payment module
// depends only on the [Reader] port; [New] selects an adapter from [Config] so
// providers swap purely by configuration (ports & adapters, like platform/sms).
// Adding a chain data source is a new adapter file plus one case in [New].
//
// The package also owns watch-only HD address derivation ([DeriveAddress]):
// deposit addresses are derived from an account xpub, so no private key ever
// touches the server.
package tron

import "fmt"

// USDTContract is the mainnet USDT (Tether) TRC20 contract address.
const USDTContract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

// Config selects and configures the active chain-reader adapter.
type Config struct {
	// Provider is one of "trongrid" or "stub" (default "stub" — dev needs
	// zero chain config; the stub auto-pays after a short delay).
	Provider string
	TronGrid TronGridConfig
	Stub     StubConfig
}

// New builds the [Reader] for cfg.Provider. An empty or "stub" provider
// returns the dev [Stub]; an unknown provider, or a selected provider missing
// required settings, returns an error.
func New(cfg Config) (Reader, error) {
	switch cfg.Provider {
	case "", "stub":
		return NewStub(cfg.Stub), nil
	case "trongrid":
		return newTronGrid(cfg.TronGrid)
	default:
		return nil, fmt.Errorf("unknown USDT chain provider %q", cfg.Provider)
	}
}
