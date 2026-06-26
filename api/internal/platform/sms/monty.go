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

// defaultMontyBaseURL is Monty Mobile's SendSMS API host.
const defaultMontyBaseURL = "https://sms.montymobile.com"

// MontySender sends SMS via Monty Mobile's GET /API/SendSMS endpoint (the Lebanon
// provider). Auth is the X-Access-Token header plus a username/apiId in the query;
// Source must be a pre-registered alphanumeric Sender ID approved on the account.
type MontySender struct {
	baseURL     string
	username    string
	apiID       string
	accessToken string
	source      string
	campaign    string
	http        *http.Client
}

// newMontySender builds a Monty-backed Sender from cfg, erroring if any required
// credential is missing. BaseURL defaults to [defaultMontyBaseURL]; Campaign
// defaults to Username.
func newMontySender(cfg MontyConfig) (Sender, error) {
	if cfg.Username == "" || cfg.APIID == "" || cfg.AccessToken == "" || cfg.SenderID == "" {
		return nil, fmt.Errorf("monty: username, api id, access token, and sender ID are required")
	}
	base := cfg.BaseURL
	if base == "" {
		base = defaultMontyBaseURL
	}
	campaign := cfg.Campaign
	if campaign == "" {
		campaign = cfg.Username
	}
	return &MontySender{
		baseURL:     strings.TrimRight(base, "/"),
		username:    cfg.Username,
		apiID:       cfg.APIID,
		accessToken: cfg.AccessToken,
		source:      cfg.SenderID,
		campaign:    campaign,
		http:        &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// montyResponse is Monty's SendSMS reply (ErrorCode 0 = success).
type montyResponse struct {
	ErrorCode    int    `json:"ErrorCode"`
	Description  string `json:"Description"`
	ID           string `json:"Id"`
	MessageCount int    `json:"MessageCount"`
}

// Send delivers message to phoneE164 via Monty, returning an error on a non-2xx
// HTTP status or a non-zero ErrorCode (e.g. -8 "Invalid Source").
func (m *MontySender) Send(ctx context.Context, phoneE164, message string) error {
	q := url.Values{}
	q.Set("username", m.username)
	q.Set("apiId", m.apiID)
	q.Set("json", "True")
	q.Set("destination", phoneE164) // Monty accepts E.164 with the leading +
	q.Set("source", m.source)
	q.Set("campaignname", m.campaign)
	q.Set("text", message)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.baseURL+"/API/SendSMS?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Access-Token", m.accessToken)

	resp, err := m.http.Do(req)
	if err != nil {
		return fmt.Errorf("monty request: %w", err)
	}
	defer resp.Body.Close()

	var parsed montyResponse
	_ = json.NewDecoder(resp.Body).Decode(&parsed) // tolerate empty/non-JSON bodies

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("monty status %d: %s", resp.StatusCode, parsed.Description)
	}
	if parsed.ErrorCode != 0 {
		return fmt.Errorf("monty error %d: %s", parsed.ErrorCode, parsed.Description)
	}
	slog.Info("sms sent via monty", "to", phoneE164, "id", parsed.ID, "parts", parsed.MessageCount)
	return nil
}
