package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// outcome is how a Fulfill error is classified for the order engine.
func outcome(err error) string {
	switch {
	case err == nil:
		return "ok"
	case Refused(err) && errors.Is(err, ErrUnavailable):
		return "refused"
	case Refused(err):
		return "refused-without-unavailable" // never valid: breaks existing parks
	case errors.Is(err, ErrPending):
		return "pending"
	case errors.Is(err, ErrUnavailable):
		return "unavailable"
	default:
		return "hard"
	}
}

func TestRefusedWrapsUnavailable(t *testing.T) {
	err := refusedf("panel x: refused (code %d)", 100)
	if !errors.Is(err, ErrRefused) || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("refusal must match both sentinels: %v", err)
	}
	if wrapped := fmt.Errorf("outer: %w", err); !Refused(wrapped) {
		t.Fatalf("a wrapped refusal is still a refusal")
	}
	if Refused(fmt.Errorf("x: %w", ErrUnavailable)) {
		t.Fatalf("plain unavailable is not a refusal")
	}
}

func TestPanelFulfill_RefusalTable(t *testing.T) {
	cases := []struct {
		body string
		want string
	}{
		{`{"status":"error","code":100,"message":"Insufficient balance"}`, "refused"},
		{`{"status":"error","code":108,"message":"2FA"}`, "refused"},
		{`{"status":"error","code":109,"message":"Product Not Found"}`, "refused"},
		{`{"status":"error","code":110,"message":"Product not available"}`, "refused"},
		{`{"status":"error","code":111,"message":"Too many requests"}`, "refused"},
		{`{"status":"error","code":120,"message":"auth"}`, "refused"},
		{`{"status":"error","code":121,"message":"auth"}`, "refused"},
		{`{"status":"error","code":122,"message":"auth"}`, "refused"},
		{`{"status":"error","code":123,"message":"ip"}`, "refused"},
		{`{"status":"error","code":130,"message":"maintenance"}`, "refused"},
		{`{"status":"error","code":999,"message":"unknown"}`, "unavailable"},
		{`{"status":"error","code":105,"message":"Qty is Small"}`, "hard"},
		{`{"status":"error","code":107,"message":"blocked"}`, "hard"},
		{`{"status":"OK","data":{"order_id":"1","status":"mystery"}}`, "unavailable"},
		{`not json`, "unavailable"},
		{`{"status":"OK","data":{"order_id":"1","status":"wait"}}`, "pending"},
		{`{"status":"OK","data":{"order_id":"1","status":"reject"}}`, "hard"},
	}
	for _, tc := range cases {
		t.Run(tc.body, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			_, err := newTestPanel(srv.URL).Fulfill(context.Background(), fulfillIn())
			if got := outcome(err); got != tc.want {
				t.Fatalf("outcome = %s, want %s (err %v)", got, tc.want, err)
			}
		})
	}
}

func TestPanelFulfill_TransportFailureIsNotRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing listens any more
	_, err := newTestPanel(url).Fulfill(context.Background(), fulfillIn())
	if got := outcome(err); got != "unavailable" {
		t.Fatalf("outcome = %s, want unavailable (err %v)", got, err)
	}
}

func TestPanelFulfill_EmptyUpstreamIsRefused(t *testing.T) {
	in := fulfillIn()
	in.UpstreamID = ""
	_, err := newTestPanel("http://127.0.0.1:1").Fulfill(context.Background(), in)
	if got := outcome(err); got != "refused" {
		t.Fatalf("outcome = %s, want refused", got)
	}
}

func TestPanelCheckStatus_ErrorIsNeverRefused(t *testing.T) {
	// The order already exists upstream: a status-check error must not look
	// like a fallback-safe refusal.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"error","code":100,"message":"Insufficient balance"}`))
	}))
	defer srv.Close()
	_, err := newTestPanel(srv.URL).CheckStatus(context.Background(), "ID_1")
	if Refused(err) || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("check error = %v, want plain ErrUnavailable", err)
	}
}

func TestUmanageFulfill_RefusalTable(t *testing.T) {
	cases := []struct {
		code string
		want string
	}{
		{"INSUFFICIENT_BALANCE", "refused"},
		{"STORE_INSUFFICIENT_BALANCE", "refused"},
		{"SMS_SERVICE_UNAVAILABLE", "refused"},
		{"RATE_LIMIT_EXCEEDED", "refused"},
		{"API_KEY_MISSING", "refused"},
		{"INVALID_API_KEY", "refused"},
		{"AUTH_REQUIRED", "refused"},
		{"AUTH_ERROR", "refused"},
		{"STORE_ACCESS_DENIED", "refused"},
		{"INVALID_STORE_ID", "refused"},
		{"INTERNAL_ERROR", "unavailable"},
		{"ORDER_FAILED", "unavailable"},
		{"SMS_ORDER_FAILED", "unavailable"},
		{"VALIDATION_ERROR", "unavailable"},
		{"BUNDLE_OUT_OF_STOCK", "hard"},
		{"INVALID_PHONE", "hard"},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"` + tc.code + `","message":"m"}}`))
			}))
			defer srv.Close()
			_, err := newTestUmanage(srv.URL, 24).Fulfill(context.Background(), umIn("bundle:123"))
			if got := outcome(err); got != tc.want {
				t.Fatalf("outcome = %s, want %s (err %v)", got, tc.want, err)
			}
		})
	}
}

func TestUmanageFulfill_BadIDsAreRefused(t *testing.T) {
	for _, up := range []string{"", "bundle", "bundle:", "nofamily:1"} {
		t.Run(up, func(t *testing.T) {
			_, err := newTestUmanage("http://127.0.0.1:1", 24).Fulfill(context.Background(), umIn(up))
			if got := outcome(err); got != "refused" {
				t.Fatalf("outcome = %s, want refused (err %v)", got, err)
			}
		})
	}
}

func TestUmanageFulfill_SearchResultIsPendingNotRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`garbage`))
	}))
	defer srv.Close()
	res, err := newTestUmanage(srv.URL, 24).Fulfill(context.Background(), umIn("bundle:123"))
	if got := outcome(err); got != "pending" || res.Reference == "" {
		t.Fatalf("outcome = %s ref %q, want pending with a search ref", got, res.Reference)
	}
}
