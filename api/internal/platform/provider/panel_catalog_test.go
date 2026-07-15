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

// newTestPanelCat is a Panel with a currency set (the plain newTestPanel leaves
// it empty), so the Cataloger probes report USD like the real config.
func newTestPanelCat(baseURL string) *Panel {
	return &Panel{
		id:       10,
		name:     "jentel",
		currency: "USD",
		baseURL:  strings.TrimRight(baseURL, "/"),
		token:    "test-token",
		http:     &http.Client{Timeout: 5 * time.Second},
	}
}

func TestPanelProfile_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/client/api/profile" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("api-token"); got != "test-token" {
			t.Errorf("missing/wrong api-token header: %q", got)
		}
		// The success body is the bare object, NOT the {status:OK,data} envelope.
		_, _ = w.Write([]byte(`{"balance":"8788.683","email":"x"}`))
	}))
	defer srv.Close()

	acc, err := newTestPanelCat(srv.URL).Profile(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if acc.Balance != 8788.683 {
		t.Errorf("balance = %v, want 8788.683", acc.Balance)
	}
	if acc.Currency != "USD" {
		t.Errorf("currency = %q, want USD", acc.Currency)
	}
	if acc.Email != "x" {
		t.Errorf("email = %q, want x", acc.Email)
	}
}

func TestPanelProfile_ErrorEnvelopeClassification(t *testing.T) {
	cases := []struct {
		name string
		code int
		want error
	}{
		{"ip blocked", 123, ErrProbeIPBlocked},
		{"auth", 121, ErrProbeAuth},
		{"maintenance", 130, ErrProbeMaintenance},
		{"unknown → unavailable", 999, ErrUnavailable},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"status":"error","code":` + itoa(tc.code) + `,"message":"nope"}`))
		}))
		_, err := newTestPanelCat(srv.URL).Profile(context.Background())
		srv.Close()
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: error = %v, want %v", tc.name, err, tc.want)
		}
	}
}

func TestPanelListProducts_Success(t *testing.T) {
	// Three rows exercising the three qty_values shapes: null, discrete list,
	// and {min,max} range (max arrives as a string upstream).
	body := `[
		{"id":"1","name":"Alpha","price":1.5,"base_price":1.2,"available":true,"product_type":"amount","category_name":"Cat A","parent_id":"0","params":["playerId"],"qty_values":null},
		{"id":"2","name":"Beta","price":2.0,"base_price":1.8,"available":true,"product_type":"package","qty_values":["110","150"]},
		{"id":"3","name":"Gamma","price":3.0,"available":false,"product_type":"amount","qty_values":{"min":1,"max":"15000"}}
	]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/client/api/products" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("api-token"); got != "test-token" {
			t.Errorf("missing api-token header: %q", got)
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	prods, err := newTestPanelCat(srv.URL).ListProducts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prods) != 3 {
		t.Fatalf("got %d products, want 3", len(prods))
	}

	// Row 0: null qty_values → no constraints, currency USD, fields carried.
	p0 := prods[0]
	if p0.UpstreamID != "1" || p0.Name != "Alpha" || p0.Category != "Cat A" || p0.ParentID != "0" {
		t.Errorf("row0 identity = %+v", p0)
	}
	if p0.Price != 1.5 || p0.BasePrice != 1.2 || p0.Currency != "USD" || !p0.Available {
		t.Errorf("row0 pricing/availability = %+v", p0)
	}
	if len(p0.Params) != 1 || p0.Params[0] != "playerId" {
		t.Errorf("row0 params = %v", p0.Params)
	}
	if p0.QtyValues != nil || p0.QtyMin != nil || p0.QtyMax != nil {
		t.Errorf("row0 null qty_values must yield no constraints, got values=%v min=%v max=%v", p0.QtyValues, p0.QtyMin, p0.QtyMax)
	}

	// Row 1: discrete list → QtyValues, no min/max.
	p1 := prods[1]
	if len(p1.QtyValues) != 2 || p1.QtyValues[0] != "110" || p1.QtyValues[1] != "150" {
		t.Errorf("row1 qty_values = %v, want [110 150]", p1.QtyValues)
	}
	if p1.QtyMin != nil || p1.QtyMax != nil {
		t.Errorf("row1 must not set min/max, got min=%v max=%v", p1.QtyMin, p1.QtyMax)
	}

	// Row 2: {min,max} range → QtyMin/QtyMax (max was a string), no values list.
	p2 := prods[2]
	if p2.QtyMin == nil || *p2.QtyMin != 1 {
		t.Errorf("row2 qtyMin = %v, want 1", p2.QtyMin)
	}
	if p2.QtyMax == nil || *p2.QtyMax != 15000 {
		t.Errorf("row2 qtyMax = %v, want 15000", p2.QtyMax)
	}
	if p2.QtyValues != nil {
		t.Errorf("row2 must not set a values list, got %v", p2.QtyValues)
	}
	if p2.Available {
		t.Errorf("row2 available = true, want false")
	}
}

func TestPanelListProducts_ErrorEnvelopeClassification(t *testing.T) {
	cases := []struct {
		name string
		code int
		want error
	}{
		{"ip blocked", 123, ErrProbeIPBlocked},
		{"auth", 120, ErrProbeAuth},
		{"maintenance", 130, ErrProbeMaintenance},
		{"unknown → unavailable", 500, ErrUnavailable},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// An error body is the {status:error,code} envelope, not an array.
			_, _ = w.Write([]byte(`{"status":"error","code":` + itoa(tc.code) + `,"message":"nope"}`))
		}))
		_, err := newTestPanelCat(srv.URL).ListProducts(context.Background())
		srv.Close()
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: error = %v, want %v", tc.name, err, tc.want)
		}
	}
}

// itoa is a tiny local int→string to keep the envelope-building inline without
// importing strconv into the test (the adapter already covers the parsing path).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
