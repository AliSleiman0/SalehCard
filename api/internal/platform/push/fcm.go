package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// fcmScope is the OAuth2 scope required by the FCM HTTP v1 API.
const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

// FCMSender sends pushes via Firebase Cloud Messaging's HTTP v1
// projects.messages:send endpoint, authenticating with a service-account
// token source (the oauth2 client caches and refreshes access tokens).
type FCMSender struct {
	endpoint string // full messages:send URL; overridable in tests
	http     *http.Client
}

// newFCMSender builds an FCM-backed Sender from cfg, erroring when credentials
// are missing or unparsable. Inline JSON wins over a file path; the project ID
// falls back to the one embedded in the service account.
func newFCMSender(cfg FCMConfig) (Sender, error) {
	raw := []byte(cfg.CredentialsJSON)
	if len(raw) == 0 {
		if cfg.CredentialsFile == "" {
			return nil, fmt.Errorf("fcm: credentials JSON or file is required")
		}
		b, err := os.ReadFile(cfg.CredentialsFile)
		if err != nil {
			return nil, fmt.Errorf("fcm: read credentials file: %w", err)
		}
		raw = b
	}

	creds, err := google.CredentialsFromJSON(context.Background(), raw, fcmScope)
	if err != nil {
		return nil, fmt.Errorf("fcm: parse credentials: %w", err)
	}

	projectID := cfg.ProjectID
	if projectID == "" {
		projectID = creds.ProjectID
	}
	if projectID == "" {
		return nil, fmt.Errorf("fcm: project ID missing from config and credentials")
	}

	client := oauth2.NewClient(context.Background(), creds.TokenSource)
	client.Timeout = 15 * time.Second
	return &FCMSender{
		endpoint: fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", projectID),
		http:     client,
	}, nil
}

// fcmRequest is the HTTP v1 send payload (one message per request).
type fcmRequest struct {
	Message fcmMessage `json:"message"`
}

type fcmMessage struct {
	Token        string            `json:"token"`
	Notification fcmNotification   `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
	Android      fcmAndroid        `json:"android"`
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type fcmAndroid struct {
	Priority string `json:"priority"`
}

// Send delivers msg to its device token, returning [ErrUnregistered] when FCM
// reports the token is no longer valid so callers can prune it.
func (f *FCMSender) Send(ctx context.Context, msg Message) error {
	body, err := json.Marshal(fcmRequest{Message: fcmMessage{
		Token:        msg.Token,
		Notification: fcmNotification{Title: msg.Title, Body: msg.Body},
		Data:         msg.Data,
		Android:      fcmAndroid{Priority: "HIGH"},
	}})
	if err != nil {
		return fmt.Errorf("fcm marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.http.Do(req)
	if err != nil {
		return fmt.Errorf("fcm request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		slog.Info("push sent via fcm", "token", truncateToken(msg.Token), "title", msg.Title)
		return nil
	}

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048)) // truncated for the error message
	// FCM signals a stale/uninstalled token as 404 NOT_FOUND with error code
	// UNREGISTERED in the details.
	if resp.StatusCode == http.StatusNotFound || strings.Contains(string(respBody), "UNREGISTERED") {
		return ErrUnregistered
	}
	return fmt.Errorf("fcm status %d: %s", resp.StatusCode, respBody)
}
