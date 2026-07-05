package idcheck

import "context"

// StubVerifier is the dev/default adapter, used when no real provider is
// configured. It resolves any non-empty player ID to a deterministic placeholder
// username so the verification flow is exercisable locally without RapidAPI
// credentials, and reports ErrIDNotFound for an empty id. It makes no network call.
type StubVerifier struct{}

// Verify returns a deterministic fake account for any non-empty id.
func (StubVerifier) Verify(_ context.Context, slug, playerID string) (Account, error) {
	_ = slug
	if playerID == "" {
		return Account{}, ErrIDNotFound
	}
	return Account{Username: "Player_" + playerID}, nil
}
