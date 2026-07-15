package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestPanel points a Panel at a stub server.
func newTestPanel(baseURL string) *Panel {
	return &Panel{
		id:      10,
		name:    "jentel",
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   "test-token",
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// fulfillIn is the canonical order line used across tests.
func fulfillIn() FulfillInput {
	return FulfillInput{
		ProductID:  "prod123",
		UpstreamID: "364",
		PlayerID:   "player-9",
		Fields:     map[string]string{"playerId": "player-9", "zoneId": "77"},
		Qty:        2,
		OrderUUID:  "ecbdd545-e616-4aee-8770-7eefa977bcd",
	}
}

func TestPanelFulfill_Accept(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("api-token"); got != "test-token" {
			t.Errorf("missing/wrong api-token header: %q", got)
		}
		if r.URL.Path != "/client/api/newOrder/364/params" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("qty") != "2" || q.Get("order_uuid") != "ecbdd545-e616-4aee-8770-7eefa977bcd" {
			t.Errorf("qty/order_uuid missing: %v", q)
		}
		if q.Get("playerId") != "player-9" || q.Get("zoneId") != "77" {
			t.Errorf("field params missing: %v", q)
		}
		// Mirrors the documented newOrder response (nested replay_api).
		_, _ = w.Write([]byte(`{"status":"OK","data":{"order_id":"ID_9fffb0d849a45215","status":"accept","price":1.26,"data":{"playerId":"player-9"},"replay_api":[{"replay":["CODE-1","CODE-2"]}]}}`))
	}))
	defer srv.Close()

	res, err := newTestPanel(srv.URL).Fulfill(context.Background(), fulfillIn())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Reference != "ID_9fffb0d849a45215" {
		t.Errorf("reference = %q", res.Reference)
	}
	if len(res.Codes) != 2 || res.Codes[0] != "CODE-1" || res.Codes[1] != "CODE-2" {
		t.Errorf("codes = %v, want [CODE-1 CODE-2]", res.Codes)
	}
}

func TestPanelFulfill_AcceptNullReplay(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"OK","data":{"order_id":"ID_1","status":"accept","replay_api":null}}`))
	}))
	defer srv.Close()

	res, err := newTestPanel(srv.URL).Fulfill(context.Background(), fulfillIn())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Reference != "ID_1" || len(res.Codes) != 0 {
		t.Errorf("got %+v, want reference ID_1 with no codes", res)
	}
}

func TestPanelFulfill_NumericOrderID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"OK","data":{"order_id":991245,"status":"accept"}}`))
	}))
	defer srv.Close()

	res, err := newTestPanel(srv.URL).Fulfill(context.Background(), fulfillIn())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Reference != "991245" {
		t.Errorf("reference = %q, want 991245", res.Reference)
	}
}

func TestPanelFulfill_WaitIsPendingWithReference(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"OK","data":{"order_id":"ID_wait1","status":"wait","replay_api":null}}`))
	}))
	defer srv.Close()

	res, err := newTestPanel(srv.URL).Fulfill(context.Background(), fulfillIn())
	if !errors.Is(err, ErrPending) {
		t.Fatalf("error = %v, want ErrPending", err)
	}
	if res.Reference != "ID_wait1" {
		t.Errorf("pending result must carry the upstream reference, got %q", res.Reference)
	}
}

func TestPanelFulfill_RejectHardFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"OK","data":{"order_id":"ID_r","status":"reject"}}`))
	}))
	defer srv.Close()

	_, err := newTestPanel(srv.URL).Fulfill(context.Background(), fulfillIn())
	if err == nil {
		t.Fatal("expected an error on reject")
	}
	if errors.Is(err, ErrPending) || errors.Is(err, ErrUnavailable) || errors.Is(err, ErrNotImplemented) {
		t.Fatalf("reject must be a plain hard-fail error, got %v", err)
	}
}

func TestPanelFulfill_EnvironmentalCodesPark(t *testing.T) {
	// 100 insufficient balance, 111 throttle, 123 IP not allowed, 130 maintenance,
	// and an unknown code — all environmental → ErrUnavailable (park).
	for _, code := range []string{"100", "111", "123", "130", "9999"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"status":"error","code":` + code + `,"message":"nope"}`))
		}))
		_, err := newTestPanel(srv.URL).Fulfill(context.Background(), fulfillIn())
		srv.Close()
		if !errors.Is(err, ErrUnavailable) {
			t.Errorf("code %s: error = %v, want ErrUnavailable", code, err)
		}
	}
}

func TestPanelFulfill_OrderSpecificCodesHardFail(t *testing.T) {
	// 105/106/112/113 invalid qty, 107 blocked player — about THIS order → compensate.
	for _, code := range []string{"105", "106", "107", "112", "113"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"status":"error","code":` + code + `,"message":"bad order"}`))
		}))
		_, err := newTestPanel(srv.URL).Fulfill(context.Background(), fulfillIn())
		srv.Close()
		if err == nil {
			t.Errorf("code %s: expected an error", code)
			continue
		}
		if errors.Is(err, ErrUnavailable) || errors.Is(err, ErrPending) {
			t.Errorf("code %s: must hard-fail, got %v", code, err)
		}
	}
}

func TestPanelFulfill_TransportAmbiguityParks(t *testing.T) {
	// HTTP 500 with garbage body.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<html>gateway error</html>`))
	}))
	if _, err := newTestPanel(srv.URL).Fulfill(context.Background(), fulfillIn()); !errors.Is(err, ErrUnavailable) {
		t.Errorf("HTTP 500 garbage: error = %v, want ErrUnavailable", err)
	}
	srv.Close()

	// Connection refused (server closed).
	if _, err := newTestPanel(srv.URL).Fulfill(context.Background(), fulfillIn()); !errors.Is(err, ErrUnavailable) {
		t.Errorf("connection refused: error = %v, want ErrUnavailable", err)
	}

	// HTTP 200 with an undecodable body.
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json at all`))
	}))
	defer srv2.Close()
	if _, err := newTestPanel(srv2.URL).Fulfill(context.Background(), fulfillIn()); !errors.Is(err, ErrUnavailable) {
		t.Errorf("garbage 200 body: error = %v, want ErrUnavailable", err)
	}
}

func TestPanelFulfill_EmptyUpstreamIDParksWithoutCall(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer srv.Close()

	in := fulfillIn()
	in.UpstreamID = ""
	_, err := newTestPanel(srv.URL).Fulfill(context.Background(), in)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("error = %v, want ErrUnavailable", err)
	}
	if called {
		t.Error("no HTTP call should be made without an upstream id")
	}
}

func TestPanelCheckStatus(t *testing.T) {
	cases := []struct {
		name, body, wantState string
		wantCodes             int
	}{
		// The check endpoint returns replay_api FLAT (unlike newOrder's nested shape).
		{"accept", `{"status":"OK","data":[{"order_id":"ID_1","status":"accept","replay_api":["CODE-9"]}]}`, "accept", 1},
		{"wait", `{"status":"OK","data":[{"order_id":"ID_1","status":"wait","replay_api":null}]}`, "wait", 0},
		{"reject", `{"status":"OK","data":[{"order_id":"ID_1","status":"reject"}]}`, "reject", 0},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/client/api/check" {
				t.Errorf("%s: unexpected path %q", tc.name, r.URL.Path)
			}
			if got := r.URL.Query().Get("orders"); got != "[ID_1]" {
				t.Errorf("%s: orders param = %q, want [ID_1]", tc.name, got)
			}
			if got := r.Header.Get("api-token"); got != "test-token" {
				t.Errorf("%s: missing api-token header", tc.name)
			}
			_, _ = w.Write([]byte(tc.body))
		}))
		st, err := newTestPanel(srv.URL).CheckStatus(context.Background(), "ID_1")
		srv.Close()
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
			continue
		}
		if st.State != tc.wantState || len(st.Codes) != tc.wantCodes {
			t.Errorf("%s: got %+v, want state=%s codes=%d", tc.name, st, tc.wantState, tc.wantCodes)
		}
	}
}

func TestPanelCheckStatus_TransportParks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	if _, err := newTestPanel(srv.URL).CheckStatus(context.Background(), "ID_1"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("error = %v, want ErrUnavailable", err)
	}
}

func TestNewPanel_Validation(t *testing.T) {
	valid := PanelConfig{ID: 10, Name: "jentel", BaseURL: "https://api.jentel-cash.com/", Token: "tok"}

	p, err := NewPanel(valid)
	if err != nil {
		t.Fatalf("valid config: %v", err)
	}
	if p.baseURL != "https://api.jentel-cash.com" {
		t.Errorf("trailing slash not trimmed: %q", p.baseURL)
	}
	if p.ID() != 10 {
		t.Errorf("ID() = %d", p.ID())
	}

	for name, cfg := range map[string]PanelConfig{
		"no token": {ID: 10, Name: "jentel", BaseURL: "https://x"},
		"no url":   {ID: 10, Name: "jentel", Token: "tok"},
		"no name":  {ID: 10, BaseURL: "https://x", Token: "tok"},
		"bad id":   {ID: 0, Name: "jentel", BaseURL: "https://x", Token: "tok"},
	} {
		if _, err := NewPanel(cfg); err == nil {
			t.Errorf("%s: expected a constructor error", name)
		}
	}
}

func TestPanelVerify_NotImplemented(t *testing.T) {
	p := newTestPanel("http://unused")
	if _, err := p.Verify(context.Background(), fulfillIn()); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("error = %v, want ErrNotImplemented", err)
	}
}
