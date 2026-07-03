// Package email is the outbound transactional/bulk email port together with its
// provider adapters (SMTP, SendGrid, log). Callers depend only on the [Sender]
// port; [New] selects an adapter from [Config] so providers swap purely by
// configuration (ports & adapters / hexagonal, mirroring platform/sms + push).
// Adding a provider is a new adapter file plus one case in [New].
package email

import (
	"context"
	"fmt"
)

// Message is one email addressed to a single recipient (plain text body).
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender is the outbound port: it delivers one email.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// Config selects and configures the active email adapter.
type Config struct {
	// Provider is one of "smtp", "sendgrid", or "log" (default "log").
	Provider string
	SMTP     SMTPConfig
	SendGrid SendGridConfig
}

// SMTPConfig holds SMTP relay credentials. Host and From are required when the
// smtp provider is selected; Port defaults to 587; auth is used when Username set.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// SendGridConfig holds the SendGrid v3 API credentials. APIKey and From are
// required when the sendgrid provider is selected.
type SendGridConfig struct {
	APIKey string
	From   string
}

// New builds the [Sender] for cfg.Provider. An empty or "log" provider returns
// the dev [LogSender]; an unknown provider, or a selected provider missing
// required credentials, returns an error.
func New(cfg Config) (Sender, error) {
	switch cfg.Provider {
	case "", "log":
		return LogSender{}, nil
	case "smtp":
		return newSMTPSender(cfg.SMTP)
	case "sendgrid":
		return newSendGridSender(cfg.SendGrid)
	default:
		return nil, fmt.Errorf("unknown email provider %q", cfg.Provider)
	}
}
