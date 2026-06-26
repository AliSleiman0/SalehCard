package sms

import (
	"context"
	"log/slog"
)

// LogSender is the development fallback: it logs the message instead of sending
// it, so the OTP flow is fully testable without a provider account.
type LogSender struct{}

// Send logs the would-be SMS at info level.
func (LogSender) Send(_ context.Context, phoneE164, message string) error {
	slog.Info("sms (dev, not sent)", "to", phoneE164, "message", message)
	return nil
}
