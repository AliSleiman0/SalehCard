// Package idcheck is the game-account ID-verification port together with its
// provider adapters (RapidAPI ID Game Checker, stub). Callers depend only on the
// [Verifier] port; [New] selects an adapter from [Config] so providers swap purely
// by configuration (ports & adapters / hexagonal). Adding a provider is a new
// adapter file plus one case in [New] — nothing outside this package changes.
package idcheck

import (
	"context"
	"errors"
	"fmt"
)

// ErrIDNotFound is returned when a provider positively reports that a player ID
// does not exist (a clean "bad ID"), as distinct from a transport/availability
// error. Callers block the purchase on this; on any other error they fail open.
var ErrIDNotFound = errors.New("idcheck: id not found")

// Account is a resolved game account. Banned is best-effort — some games do not
// report ban status, in which case it stays false.
type Account struct {
	Username string
	Banned   bool
}

// Verifier is the outbound port: it resolves a player ID under a game slug to the
// account it belongs to, or ErrIDNotFound when no such account exists.
type Verifier interface {
	Verify(ctx context.Context, slug, playerID string) (Account, error)
}

// Config selects and configures the active verification adapter.
type Config struct {
	// Provider is one of "rapidapi" or "stub" (default "stub").
	Provider string
	// RapidAPIKey authenticates the RapidAPI adapter (the x-rapidapi-key header).
	// Required when Provider is "rapidapi"; server-side only — never shipped to
	// clients.
	RapidAPIKey string
}

// New builds the [Verifier] for cfg.Provider. An empty or "stub" provider returns
// the dev [StubVerifier]; "rapidapi" requires a key; an unknown provider errors.
func New(cfg Config) (Verifier, error) {
	switch cfg.Provider {
	case "", "stub":
		return StubVerifier{}, nil
	case "rapidapi":
		return newRapidAPIVerifier(cfg.RapidAPIKey)
	default:
		return nil, fmt.Errorf("unknown ID-check provider %q", cfg.Provider)
	}
}
