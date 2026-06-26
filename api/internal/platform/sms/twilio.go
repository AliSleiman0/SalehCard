package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TwilioSender sends SMS directly via the Twilio Messages REST API — no gateway
// or database required. Configure it with a Twilio Account SID, Auth Token, and
// a sender number (or Messaging Service SID) in E.164 form.
type TwilioSender struct {
	accountSID string
	authToken  string
	from       string
	http       *http.Client
}

// newTwilioSender builds a Twilio-backed Sender from cfg, erroring if any
// required credential is missing.
func newTwilioSender(cfg TwilioConfig) (Sender, error) {
	if cfg.AccountSID == "" || cfg.AuthToken == "" || cfg.From == "" {
		return nil, fmt.Errorf("twilio: account SID, auth token, and from-number are required")
	}
	return &TwilioSender{
		accountSID: cfg.AccountSID,
		authToken:  cfg.AuthToken,
		from:       cfg.From,
		http:       &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// Send delivers message to phoneE164 via Twilio, returning an error on a non-2xx
// response (Twilio replies 201 on success).
func (t *TwilioSender) Send(ctx context.Context, phoneE164, message string) error {
	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)

	form := url.Values{}
	form.Set("To", phoneE164)
	form.Set("From", t.from)
	form.Set("Body", message)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.http.Do(req)
	if err != nil {
		return fmt.Errorf("twilio request: %w", err)
	}
	defer resp.Body.Close()

	var parsed struct {
		SID     string `json:"sid"`
		Status  string `json:"status"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&parsed)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("twilio status %d (code %d): %s", resp.StatusCode, parsed.Code, parsed.Message)
	}
	slog.Info("sms sent via twilio", "to", phoneE164, "sid", parsed.SID, "status", parsed.Status)
	return nil
}
