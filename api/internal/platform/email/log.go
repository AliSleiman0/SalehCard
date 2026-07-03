package email

import (
	"context"
	"log/slog"
)

// LogSender is the development fallback: it logs the email instead of sending
// it, so the bulk-email flow is fully testable without a real mail provider.
type LogSender struct{}

// Send logs the would-be email at info level (body omitted to keep logs short).
func (LogSender) Send(_ context.Context, msg Message) error {
	slog.Info("email (dev, not sent)", "to", msg.To, "subject", msg.Subject)
	return nil
}
