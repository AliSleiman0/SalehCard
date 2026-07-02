package push

import (
	"context"
	"log/slog"
)

// LogSender is the development fallback: it logs the push instead of sending
// it, so the notification flow is fully testable without a Firebase project.
type LogSender struct{}

// Send logs the would-be push at info level.
func (LogSender) Send(_ context.Context, msg Message) error {
	slog.Info("push (dev, not sent)", "token", truncateToken(msg.Token), "title", msg.Title, "body", msg.Body)
	return nil
}

// truncateToken keeps log lines short and avoids dumping full device tokens.
func truncateToken(token string) string {
	if len(token) > 12 {
		return token[:12] + "…"
	}
	return token
}
