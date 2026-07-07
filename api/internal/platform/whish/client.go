package whish

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	initiateTimeout = 10 * time.Second
	statusTimeout   = 5 * time.Second
)

// Client is a thin wrapper around the Whish HTTP API. It owns a reusable
// http.Client and sets the four Whish headers (channel, secret, websiteUrl,
// User-Agent) on every call.
type Client struct {
	cfg  ClientConfig
	http *http.Client
}

// NewClient constructs a Client, validating and defaulting cfg.
func NewClient(cfg ClientConfig) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: initiateTimeout},
	}, nil
}

// Initiate calls POST /payment/whish and returns the collectUrl.
func (c *Client) Initiate(ctx context.Context, req initiateRequest) (string, error) {
	var resp initiateResponse
	if err := c.do(ctx, http.MethodPost, "/payment/whish", req, &resp, initiateTimeout); err != nil {
		return "", err
	}
	if !resp.Status {
		return "", fmt.Errorf("whish initiate rejected: code=%q", resp.Code)
	}
	if resp.Data.CollectURL == "" {
		return "", fmt.Errorf("whish initiate returned empty collectUrl")
	}
	return resp.Data.CollectURL, nil
}

// CollectStatus calls POST /payment/collect/status, returning the normalized
// status string and the payer phone (decoded leniently — Whish returns it as a
// number or a string depending on the account).
func (c *Client) CollectStatus(ctx context.Context, req statusRequest) (string, string, error) {
	var resp statusResponse
	if err := c.do(ctx, http.MethodPost, "/payment/collect/status", req, &resp, statusTimeout); err != nil {
		return "", "", err
	}
	if !resp.Status {
		return "", "", fmt.Errorf("whish status rejected: code=%q", resp.Code)
	}
	phone := ""
	switch v := resp.Data.PayerPhoneNumber.(type) {
	case string:
		phone = v
	case float64:
		phone = fmt.Sprintf("%.0f", v)
	}
	return resp.Data.CollectStatus, phone, nil
}

func (c *Client) do(ctx context.Context, method, path string, body, out any, timeout time.Duration) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("whish marshal: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, method, c.cfg.BaseURL+path, bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("whish new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("channel", c.cfg.Channel)
	req.Header.Set("secret", c.cfg.Secret)
	req.Header.Set("websiteUrl", c.cfg.WebsiteURL)
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("whish http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("whish read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("whish http %d: %s", resp.StatusCode, string(respBody))
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("whish decode: %w (body=%s)", err, string(respBody))
	}
	return nil
}
