// Package sms delivers text messages for the OTP flow. It abstracts the mip SMS
// gateway behind a small Sender interface so the user service stays unaware of
// the transport, and provides a logging fallback for local development.
package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Sender delivers a message to a phone number in E.164 form.
type Sender interface {
	Send(ctx context.Context, phoneE164, message string) error
}

// LogSender is the development fallback: it logs the message instead of sending
// it, so the OTP flow is fully testable without a gateway or Twilio account.
type LogSender struct{}

// Send logs the would-be SMS at info level.
func (LogSender) Send(_ context.Context, phoneE164, message string) error {
	slog.Info("sms (dev, not sent)", "to", phoneE164, "message", message)
	return nil
}

// GatewayClient sends via the mip SMS gateway's POST {base}/api/sms/send.
type GatewayClient struct {
	baseURL  string
	provider string
	token    string // optional Bearer, used only if the gateway requires auth
	http     *http.Client
}

// NewGatewayClient builds a client for the gateway at baseURL (e.g.
// "http://localhost:8081/SMS_GATEWAY_API"). provider selects the downstream
// channel ("twilio"); token is an optional Bearer for a security-enabled gateway.
func NewGatewayClient(baseURL, provider, token string) *GatewayClient {
	return &GatewayClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		provider: provider,
		token:    token,
		http:     &http.Client{Timeout: 15 * time.Second},
	}
}

// sendRequest mirrors the gateway's SmsRequestDto.
type sendRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	Message     string `json:"message"`
	Provider    string `json:"provider,omitempty"`
}

// sendResponse mirrors the gateway's envelope (time/message/code/messageSid).
type sendResponse struct {
	Message    string `json:"message"`
	Code       int    `json:"code"`
	MessageSid string `json:"messageSid"`
}

// Send posts the message to the gateway, returning an error on a non-2xx HTTP
// status or a non-200 envelope code.
func (c *GatewayClient) Send(ctx context.Context, phoneE164, message string) error {
	body, err := json.Marshal(sendRequest{PhoneNumber: phoneE164, Message: message, Provider: c.provider})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/sms/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("sms gateway request: %w", err)
	}
	defer resp.Body.Close()

	var parsed sendResponse
	_ = json.NewDecoder(resp.Body).Decode(&parsed) // tolerate empty/non-JSON bodies

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sms gateway status %d: %s", resp.StatusCode, parsed.Message)
	}
	if parsed.Code != 0 && parsed.Code != http.StatusOK {
		return fmt.Errorf("sms gateway error %d: %s", parsed.Code, parsed.Message)
	}
	slog.Info("sms sent via gateway", "to", phoneE164, "sid", parsed.MessageSid)
	return nil
}
