package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// sendGridSender delivers mail through the SendGrid v3 HTTP API.
type sendGridSender struct {
	apiKey string
	from   string
	client *http.Client
}

// newSendGridSender validates the SendGrid config and returns a Sender.
func newSendGridSender(cfg SendGridConfig) (Sender, error) {
	if cfg.APIKey == "" || cfg.From == "" {
		return nil, fmt.Errorf("sendgrid email: SENDGRID_API_KEY and EMAIL_FROM are required")
	}
	return &sendGridSender{apiKey: cfg.APIKey, from: cfg.From, client: &http.Client{Timeout: 15 * time.Second}}, nil
}

// Send posts one plain-text email to the SendGrid mail-send endpoint.
func (s *sendGridSender) Send(ctx context.Context, msg Message) error {
	payload := map[string]any{
		"personalizations": []map[string]any{{"to": []map[string]string{{"email": msg.To}}}},
		"from":             map[string]string{"email": s.from},
		"subject":          msg.Subject,
		"content":          []map[string]string{{"type": "text/plain", "value": msg.Body}},
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("sendgrid: unexpected status %d", resp.StatusCode)
	}
	return nil
}
