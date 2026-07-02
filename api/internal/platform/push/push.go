// Package push is the outbound mobile push-notification port together with its
// provider adapters (FCM, log). The notification module depends only on the
// [Sender] port; [New] selects an adapter from [Config] so providers swap purely
// by configuration (ports & adapters / hexagonal, mirroring platform/sms).
// Adding a provider is a new adapter file plus one case in [New].
package push

import (
	"context"
	"errors"
	"fmt"
)

// ErrUnregistered is returned by a [Sender] when the provider reports the device
// token is stale (e.g. FCM UNREGISTERED / 404) — callers should prune the token.
var ErrUnregistered = errors.New("push: device token unregistered")

// Message is one push notification addressed to a single device token.
type Message struct {
	Token string
	Title string
	Body  string
	Data  map[string]string // provider data payload; FCM requires string values
}

// Sender is the outbound port: it delivers a push message to one device token.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// Config selects and configures the active push adapter.
type Config struct {
	// Provider is one of "fcm" or "log" (default "log").
	Provider string
	FCM      FCMConfig
}

// FCMConfig holds Firebase Cloud Messaging (HTTP v1) credentials. Exactly one
// of CredentialsJSON (inline service-account JSON, preferred for Azure app
// settings) or CredentialsFile (path, local dev) must be set.
type FCMConfig struct {
	CredentialsJSON string
	CredentialsFile string
	ProjectID       string // optional; derived from the service account when empty
}

// New builds the [Sender] for cfg.Provider. An empty or "log" provider returns
// the dev [LogSender]; an unknown provider, or a selected provider missing
// required credentials, returns an error.
func New(cfg Config) (Sender, error) {
	switch cfg.Provider {
	case "", "log":
		return LogSender{}, nil
	case "fcm":
		return newFCMSender(cfg.FCM)
	default:
		return nil, fmt.Errorf("unknown push provider %q", cfg.Provider)
	}
}
