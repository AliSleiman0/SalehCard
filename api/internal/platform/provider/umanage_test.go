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

// newTestUmanage points a Umanage adapter at a stub server. A non-zero storeID
// pre-resolves the store (storeSet true) so /stores is only hit by the tests
// that explicitly exercise resolution (storeID 0).
func newTestUmanage(baseURL string, storeID int) *Umanage {
	return &Umanage{
		id:        13,
		name:      "umanage",
		currency:  "LBP",
		baseURL:   strings.TrimRight(baseURL, "/"),
		apiKey:    "key",
		apiSecret: "secret",
		http:      &http.Client{Timeout: 5 * time.Second},
		storeID:   storeID,
		storeSet:  storeID > 0,
	}
}

// umIn is the canonical order line for umanage tests.
func umIn(upstreamID string) FulfillInput {
	return FulfillInput{
		ProductID:  "prodX",
		UpstreamID: upstreamID,
		Fields:     map[string]string{"secondary_number": "03123456", "recipient_phone": "03123456"},
		Qty:        1,
		OrderUUID:  "uuid-abc",
	}
}

// --- Profile / store resolution ---

func TestUmanageProfile_SuccessResolvesStore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stores" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "key" || r.Header.Get("X-API-Secret") != "secret" {
			t.Errorf("missing auth headers: %v", r.Header)
		}
		_, _ = w.Write([]byte(`{"success":true,"stores":[{"store_id":24,"store_name":"Main","balance_lbp":5000000}]}`))
	}))
	defer srv.Close()

	// StoreID 0 → the probe must resolve it from /stores.
	acc, err := newTestUmanage(srv.URL, 0).Profile(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if acc.Balance != 5000000 {
		t.Errorf("balance = %v, want 5000000", acc.Balance)
	}
	if acc.Currency != "LBP" {
		t.Errorf("currency = %q, want LBP", acc.Currency)
	}
}

func TestUmanageProfile_AuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"INVALID_API_KEY","message":"bad key"}}`))
	}))
	defer srv.Close()

	_, err := newTestUmanage(srv.URL, 0).Profile(context.Background())
	if !errors.Is(err, ErrProbeAuth) {
		t.Fatalf("error = %v, want ErrProbeAuth", err)
	}
}

// --- Fulfill ---

func TestUmanageFulfill_BundleCompleted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/stores/24/orders" {
			t.Errorf("unexpected %s %q", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"order":{"order_id":98765,"status":"completed","app_user_reference":"uuid-abc"}}`))
	}))
	defer srv.Close()

	res, err := newTestUmanage(srv.URL, 24).Fulfill(context.Background(), umIn("bundle:123"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Reference != "bundle:98765" {
		t.Errorf("reference = %q, want bundle:98765", res.Reference)
	}
}

func TestUmanageFulfill_BundleProcessingParks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"order":{"order_id":98765,"status":"processing"}}`))
	}))
	defer srv.Close()

	res, err := newTestUmanage(srv.URL, 24).Fulfill(context.Background(), umIn("bundle:123"))
	if !errors.Is(err, ErrPending) {
		t.Fatalf("error = %v, want ErrPending", err)
	}
	if res.Reference != "bundle:98765" {
		t.Errorf("pending reference = %q, want bundle:98765", res.Reference)
	}
}

func TestUmanageFulfill_BundleEnvironmentalParks(t *testing.T) {
	// INSUFFICIENT_BALANCE, no order created → environmental → ErrUnavailable.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"INSUFFICIENT_BALANCE","message":"low"}}`))
	}))
	defer srv.Close()

	_, err := newTestUmanage(srv.URL, 24).Fulfill(context.Background(), umIn("bundle:123"))
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("error = %v, want ErrUnavailable", err)
	}
}

func TestUmanageFulfill_BundleOutOfStockHardFails(t *testing.T) {
	// BUNDLE_OUT_OF_STOCK is order-specific → a plain error (compensate), NOT
	// ErrUnavailable and NOT ErrPending.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"BUNDLE_OUT_OF_STOCK","message":"gone"}}`))
	}))
	defer srv.Close()

	_, err := newTestUmanage(srv.URL, 24).Fulfill(context.Background(), umIn("bundle:123"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if errors.Is(err, ErrUnavailable) || errors.Is(err, ErrPending) {
		t.Fatalf("BUNDLE_OUT_OF_STOCK must hard-fail, got %v", err)
	}
}

func TestUmanageFulfill_SMSCompleted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stores/24/sms-orders" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"order":{"order_id":4242,"status":"completed","sms_transaction":{"confirmation_message":"sent"}}}`))
	}))
	defer srv.Close()

	res, err := newTestUmanage(srv.URL, 24).Fulfill(context.Background(), umIn("sms:credit:5.5:alfa"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Reference != "sms:4242" {
		t.Errorf("reference = %q, want sms:4242", res.Reference)
	}
	if res.Meta["confirmation"] != "sent" {
		t.Errorf("meta confirmation = %q, want sent", res.Meta["confirmation"])
	}
}

func TestUmanageFulfill_TelecomVoucherCompleted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stores/24/telecom-orders" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"order":{"order_id":55,"status":"completed","voucher":{"code":"1234-5678-9012","serial":"SER1","expiry_date":"2027-01-01"}}}`))
	}))
	defer srv.Close()

	res, err := newTestUmanage(srv.URL, 24).Fulfill(context.Background(), umIn("telecom:88"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Reference != "telecom:55" {
		t.Errorf("reference = %q, want telecom:55", res.Reference)
	}
	if len(res.Codes) != 1 || res.Codes[0] != "1234-5678-9012" {
		t.Errorf("codes = %v, want [1234-5678-9012]", res.Codes)
	}
}

func TestUmanageFulfill_TelecomRefundedFailure(t *testing.T) {
	// success:false with an auto-refunded order → definitively failed → plain
	// error (compensate), NOT ErrUnavailable / ErrPending.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"order":{"order_id":56,"status":"failed","refunded":true,"error":"network down"}}`))
	}))
	defer srv.Close()

	_, err := newTestUmanage(srv.URL, 24).Fulfill(context.Background(), umIn("telecom:88"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if errors.Is(err, ErrUnavailable) || errors.Is(err, ErrPending) {
		t.Fatalf("refunded failure must hard-fail, got %v", err)
	}
}

func TestUmanageFulfill_TransportAmbiguityParksToSearch(t *testing.T) {
	// Server accepts then closes without a valid HTTP reply → http.Client.Do
	// errors → ambiguous → ErrPending with a "search:"-prefixed reference.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("server does not support hijacking")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("hijack: %v", err)
		}
		_ = conn.Close() // slam the connection with no response
	}))
	defer srv.Close()

	res, err := newTestUmanage(srv.URL, 24).Fulfill(context.Background(), umIn("bundle:123"))
	if !errors.Is(err, ErrPending) {
		t.Fatalf("error = %v, want ErrPending", err)
	}
	if res.Reference != "search:bundle:uuid-abc" {
		t.Errorf("reference = %q, want search:bundle:uuid-abc", res.Reference)
	}
}

func TestUmanageFulfill_UndecodableBodyParksToSearch(t *testing.T) {
	// A real HTTP response with a garbage body is still ambiguous → search park.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html>gateway error</html>`))
	}))
	defer srv.Close()

	res, err := newTestUmanage(srv.URL, 24).Fulfill(context.Background(), umIn("bundle:123"))
	if !errors.Is(err, ErrPending) {
		t.Fatalf("error = %v, want ErrPending", err)
	}
	if res.Reference != "search:bundle:uuid-abc" {
		t.Errorf("reference = %q, want search:bundle:uuid-abc", res.Reference)
	}
}

// --- CheckStatus ---

func TestUmanageCheckStatus_DirectRef(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stores/24/orders/98765" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"order":{"order_id":98765,"status":"completed"}}`))
	}))
	defer srv.Close()

	st, err := newTestUmanage(srv.URL, 24).CheckStatus(context.Background(), "bundle:98765")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.State != "accept" {
		t.Errorf("state = %q, want accept", st.State)
	}
}

func TestUmanageCheckStatus_SearchRefMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The search path hits the family LIST endpoint, not a detail id.
		if r.URL.Path != "/stores/24/orders" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"orders":[
			{"order_id":1,"status":"completed","app_user_reference":"other"},
			{"order_id":2,"status":"completed","app_user_reference":"uuid-abc","voucher":{"code":"V-9"}}
		]}`))
	}))
	defer srv.Close()

	st, err := newTestUmanage(srv.URL, 24).CheckStatus(context.Background(), "search:bundle:uuid-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.State != "accept" {
		t.Errorf("state = %q, want accept", st.State)
	}
	if len(st.Codes) != 1 || st.Codes[0] != "V-9" {
		t.Errorf("codes = %v, want [V-9]", st.Codes)
	}
}

func TestUmanageCheckStatus_SearchRefNotFoundWaits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// A short (< pageSize) list with no match → last page → not found → wait.
		_, _ = w.Write([]byte(`{"success":true,"orders":[{"order_id":1,"status":"completed","app_user_reference":"someone-else"}]}`))
	}))
	defer srv.Close()

	st, err := newTestUmanage(srv.URL, 24).CheckStatus(context.Background(), "search:bundle:uuid-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.State != "wait" {
		t.Errorf("state = %q, want wait", st.State)
	}
}

// --- ListProducts ---

func TestUmanageListProducts_MergesFamilies(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/stores/24/bundles", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"bundles":[{"bundle_id":123,"bundle_size":5,"bundle_type":"data","bundle_days":30,"price_lbp":250000,"is_available":1}]}`))
	})
	mux.HandleFunc("/stores/24/sms-products", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"has_sms_service":true,"credits":[{"carrier_type":"alfa","pricing":[{"amount_usd":"5.5","price_lbp":500000}]}],"gifts":[{"product_code":"MI7","name":"Alfa Gift MI7","carrier_type":"alfa","price_lbp":700000}]}`))
	})
	mux.HandleFunc("/stores/24/telecom-products", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"products":[{"product_id":88,"name":"Touch Voucher 10","brand":"Touch","product_type":"voucher","category":"Vouchers","price_lbp":900000}]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	prods, err := newTestUmanage(srv.URL, 24).ListProducts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	byID := map[string]CatalogProduct{}
	for _, p := range prods {
		byID[p.UpstreamID] = p
	}
	for _, want := range []string{"bundle:123", "sms:credit:5.5:alfa", "sms:gift:MI7:alfa", "telecom:88"} {
		if _, ok := byID[want]; !ok {
			t.Errorf("missing upstream id %q; got %v", want, keysOf(byID))
		}
	}
	if b := byID["bundle:123"]; !b.Available || b.Price != 250000 || b.Currency != "LBP" {
		t.Errorf("bundle merged wrong: %+v", b)
	}
	if g := byID["sms:gift:MI7:alfa"]; g.Name != "Alfa Gift MI7" || g.ProductType != "gift" {
		t.Errorf("sms gift merged wrong: %+v", g)
	}
	if tp := byID["telecom:88"]; tp.Name != "Touch Voucher 10" || tp.Category != "Touch Vouchers" {
		t.Errorf("telecom merged wrong: %+v", tp)
	}
}

func keysOf(m map[string]CatalogProduct) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestNewUmanage_Validation(t *testing.T) {
	valid := UmanageConfig{ID: 13, Name: "umanage", BaseURL: "https://api.umanageapp.uk/", APIKey: "k", APISecret: "s", StoreID: 24}
	u, err := NewUmanage(valid)
	if err != nil {
		t.Fatalf("valid config: %v", err)
	}
	if u.baseURL != "https://api.umanageapp.uk" {
		t.Errorf("trailing slash not trimmed: %q", u.baseURL)
	}
	if !u.storeSet || u.storeID != 24 {
		t.Errorf("StoreID>0 should pre-resolve: set=%v id=%d", u.storeSet, u.storeID)
	}
	for name, cfg := range map[string]UmanageConfig{
		"no secret": {ID: 13, Name: "umanage", BaseURL: "https://x", APIKey: "k"},
		"no key":    {ID: 13, Name: "umanage", BaseURL: "https://x", APISecret: "s"},
		"no url":    {ID: 13, Name: "umanage", APIKey: "k", APISecret: "s"},
		"bad id":    {ID: 0, Name: "umanage", BaseURL: "https://x", APIKey: "k", APISecret: "s"},
	} {
		if _, err := NewUmanage(cfg); err == nil {
			t.Errorf("%s: expected a constructor error", name)
		}
	}
}
