// Package sms is the outbound SMS port for the OTP flow together with its
// provider adapters (Monty, Twilio, log). The user service depends only on the
// [Sender] port; [New] selects an adapter from [Config] so providers swap purely
// by configuration (ports & adapters / hexagonal). Adding a provider is a new
// adapter file plus one case in [New].
package sms

import (
	"context"
	"fmt"
)

// Sender is the outbound port: it delivers a message to a phone number in E.164.
type Sender interface {
	Send(ctx context.Context, phoneE164, message string) error
}

// Config selects and configures the active SMS adapter.
type Config struct {
	// Provider is one of "monty", "twilio", or "log" (default "log").
	Provider string
	Monty    MontyConfig
	Twilio   TwilioConfig
}

// MontyConfig holds Monty Mobile (Lebanon) credentials and settings for the
// sms.montymobile.com/API/SendSMS endpoint.
type MontyConfig struct {
	BaseURL     string // e.g. https://sms.montymobile.com
	Username    string
	APIID       string // the apiId query parameter
	AccessToken string // X-Access-Token header
	SenderID    string // registered alphanumeric Source (e.g. "SalehCard")
	Campaign    string // optional campaignname; defaults to Username
}

// TwilioConfig holds Twilio (international) credentials and settings.
type TwilioConfig struct {
	AccountSID string
	AuthToken  string
	From       string // E.164 sender number or Messaging Service SID
}

// New builds the [Sender] for cfg.Provider. An empty or "log" provider returns
// the dev [LogSender]; an unknown provider, or a selected provider missing
// required credentials, returns an error.
func New(cfg Config) (Sender, error) {
	switch cfg.Provider {
	case "", "log":
		return LogSender{}, nil
	case "monty":
		return newMontySender(cfg.Monty)
	case "twilio":
		return newTwilioSender(cfg.Twilio)
	default:
		return nil, fmt.Errorf("unknown SMS provider %q", cfg.Provider)
	}
}
