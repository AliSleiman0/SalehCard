package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newTestTorosfon points a Torosfon adapter at a stub server.
func newTestTorosfon(baseURL string) *Torosfon {
	return &Torosfon{
		id:       14,
		name:     "torosfon",
		currency: "USD",
		baseURL:  strings.TrimRight(baseURL, "/"),
		token:    "test-token",
		http:     &http.Client{Timeout: 2 * time.Second},
	}
}

// torosIn is an order line shaped like production: OrderUUID is a 24-hex
// ObjectID and the per-unit amount arrives as Qty.
func torosIn() FulfillInput {
	return FulfillInput{
		ProductID:  "prod123",
		UpstreamID: "147",
		PlayerID:   "55667788",
		Fields:     map[string]string{"player_id": "55667788"},
		Qty:        10000,
		OrderUUID:  "6af9e7b4c0ffee0123456789",
	}
}

// torosServer answers every request with body at status.
func torosServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestNewTorosfon_Validation(t *testing.T) {
	if _, err := NewTorosfon(TorosfonConfig{ID: 14, Name: "torosfon", BaseURL: "https://x"}); err == nil {
		t.Fatal("missing token must fail")
	}
	tf, err := NewTorosfon(TorosfonConfig{ID: 14, Name: "torosfon", BaseURL: "https://x/", Token: "t"})
	if err != nil {
		t.Fatalf("valid config: %v", err)
	}
	if tf.ID() != 14 || tf.currency != "USD" || tf.baseURL != "https://x" {
		t.Fatalf("defaults not applied: %+v", tf)
	}
	p, err := New(SupplierSpec{ID: 14, Name: "torosfon", Kind: "torosfon", BaseURL: "https://x", Token: "t"})
	if err != nil {
		t.Fatalf("New(kind torosfon): %v", err)
	}
	if _, ok := p.(*Torosfon); !ok {
		t.Fatalf("New built %T, want *Torosfon", p)
	}
	if _, ok := p.(Cataloger); !ok {
		t.Fatal("torosfon must implement Cataloger")
	}
}

func TestTorosfonFulfill_RequestShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("api-token"); got != "test-token" {
			t.Errorf("api-token header = %q", got)
		}
		if r.URL.Path != "/api/PlaceOrder/147/data" {
			t.Errorf("path = %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("amount") != "10000" || q.Get("player_Id") != "55667788" || q.Get("referenceId") != "6af9e7b4c0ffee0123456789" {
			t.Errorf("query = %v", q)
		}
		_, _ = w.Write([]byte(`{"message":"","success":true,"data":{"order_id":123,"order_ex_rate":1,"order_value":1.1,"order_status":"Completed","order_desc":"CODE-XYZ"}}`))
	}))
	defer srv.Close()
	res, err := newTestTorosfon(srv.URL).Fulfill(context.Background(), torosIn())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Reference != "123" || len(res.Codes) != 1 || res.Codes[0] != "CODE-XYZ" {
		t.Fatalf("result = %+v", res)
	}
}

func TestTorosfonFulfill_PlayerFromFieldsWhenSnapshotEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("player_Id"); got != "998877" {
			t.Errorf("player_Id = %q, want the field value", got)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"order_id":5,"order_status":"Pending"}}`))
	}))
	defer srv.Close()
	in := torosIn()
	in.PlayerID = ""
	in.Fields = map[string]string{"player_id": "998877"}
	_, _ = newTestTorosfon(srv.URL).Fulfill(context.Background(), in)
}

// Bodies below are the shapes the live API returned on 2026-11-17.
func TestTorosfonFulfill_OutcomeTable(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    string
		want    string
		wantRef string // "" = no ref expected; "search" = a search: ref
	}{
		{"completed with code", 200, `{"message":"","success":true,"data":{"order_id":123,"order_status":"Completed","order_desc":"FFE334CE8989"}}`, "ok", "123"},
		{"completed no code", 200, `{"message":"","success":true,"data":{"order_id":124,"order_status":"Completed","order_desc":""}}`, "ok", "124"},
		{"pending", 200, `{"message":"","success":true,"data":{"order_id":50,"order_status":"Pending","order_desc":""}}`, "pending", "50"},
		{"unknown status with id polls, never re-sends", 200, `{"success":true,"data":{"order_id":51,"order_status":"Mystery"}}`, "pending", "51"},
		{"live reject (success false + real id)", 200, `{"success":false,"message":"An external error has occurred with this product, almost rejected-- Order Cancelled Id:1511645 User not found!","data":{"order_id":1511645,"order_ex_rate":1,"order_value":0.0154,"order_status":"Cancelled","balanceBefore":-1880.8,"order_desc":"User not found!"}}`, "hard", ""},
		{"documented reject (success true)", 200, `{"message":"user id not found","success":true,"data":{"order_id":55,"order_status":"Cancelled","order_desc":"user id not found"}}`, "hard", ""},
		{"product not found", 200, `{"success":false,"message":"Product Not Found","data":{"order_id":null,"order_ex_rate":null,"order_value":null,"order_status":"Cancelled","balanceBefore":null,"order_desc":"Product Not Found"}}`, "refused", ""},
		{"product inactive", 200, `{"success":false,"message":"The product is not active you can't purchase","data":{"order_id":null,"order_status":"Cancelled"}}`, "refused", ""},
		{"qty out of range", 200, `{"success":false,"message":"The requested quantity is not in allowed range between 1000 and 10000000","data":{"order_id":null,"order_status":"Cancelled"}}`, "refused", ""},
		{"documented generic failure", 200, `{"message":"some things","success":false,"data":""}`, "refused", ""},
		{"validation 400", 400, `{"type":"https://tools.ietf.org/html/rfc9110#section-15.5.1","title":"One or more validation errors occurred.","status":400,"errors":{"amount":["The value 'x' is not valid."]}}`, "refused", ""},
		{"401", 401, ``, "refused", ""},
		{"429", 429, ``, "refused", ""},
		{"500 empty", 500, ``, "pending", "search"},
		{"500 failure json", 500, `{"success":false,"message":"boom","data":""}`, "pending", "search"},
		{"502 html", 502, `<html>bad gateway</html>`, "pending", "search"},
		{"200 garbage", 200, `not json`, "pending", "search"},
		{"success without order id", 200, `{"success":true,"message":"","data":{"order_id":null}}`, "pending", "search"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := torosServer(t, tc.status, tc.body)
			res, err := newTestTorosfon(srv.URL).Fulfill(context.Background(), torosIn())
			if got := outcome(err); got != tc.want {
				t.Fatalf("outcome = %s, want %s (err %v)", got, tc.want, err)
			}
			switch tc.wantRef {
			case "search":
				if res.Reference != "search:torosfon:6af9e7b4c0ffee0123456789" {
					t.Fatalf("ref = %q, want the search ref", res.Reference)
				}
			case "":
				if tc.want == "pending" {
					t.Fatal("a pending outcome must carry a ref")
				}
			default:
				if res.Reference != tc.wantRef {
					t.Fatalf("ref = %q, want %q", res.Reference, tc.wantRef)
				}
			}
			// Torosfon has no dedupe: no outcome may be a retryable
			// "unavailable" that isn't a definite refusal.
			if errors.Is(err, ErrUnavailable) && !Refused(err) {
				t.Fatalf("ambiguous outcome returned as re-sendable unavailable: %v", err)
			}
		})
	}
}

func TestTorosfonFulfill_TransportFailureIsHeldNotRetried(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()
	res, err := newTestTorosfon(url).Fulfill(context.Background(), torosIn())
	if got := outcome(err); got != "pending" || res.Reference != "search:torosfon:6af9e7b4c0ffee0123456789" {
		t.Fatalf("outcome = %s ref %q, want pending with a search ref (err %v)", got, res.Reference, err)
	}
}

func TestTorosfonFulfill_TimeoutIsHeldNotRetried(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte(`{"success":true,"data":{"order_id":1,"order_status":"Completed"}}`))
	}))
	defer srv.Close()
	tf := newTestTorosfon(srv.URL)
	tf.http.Timeout = 50 * time.Millisecond
	res, err := tf.Fulfill(context.Background(), torosIn())
	if got := outcome(err); got != "pending" || !strings.HasPrefix(res.Reference, "search:") {
		t.Fatalf("outcome = %s ref %q, want pending search ref", got, res.Reference)
	}
}

func TestTorosfonFulfill_EmptyUpstreamIsRefused(t *testing.T) {
	in := torosIn()
	in.UpstreamID = ""
	_, err := newTestTorosfon("http://127.0.0.1:1").Fulfill(context.Background(), in)
	if got := outcome(err); got != "refused" {
		t.Fatalf("outcome = %s, want refused", got)
	}
}

func TestTorosfonCheckStatus(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		body      string
		wantState string
		wantErr   bool
		wantCodes int
	}{
		{"completed", 200, `{"message":"","success":true,"data":{"order_id":123,"order_status":"Completed","order_desc":"CODE-1"}}`, "accept", false, 1},
		{"completed empty desc", 200, `{"success":true,"data":{"order_id":123,"order_status":"Completed","order_desc":""}}`, "accept", false, 0},
		{"pending", 200, `{"success":true,"data":{"order_id":123,"order_status":"Pending","order_desc":""}}`, "wait", false, 0},
		{"unknown status waits", 200, `{"success":true,"data":{"order_id":123,"order_status":"Mystery"}}`, "wait", false, 0},
		{"live cancelled", 200, `{"success":false,"message":"Automatic Closing Because of ::  errMsg => User not found!","data":{"order_id":123,"order_ex_rate":1,"order_value":0.015,"order_status":"Cancelled","balanceBefore":-1880.8,"order_desc":"Automatic Closing Because of ::  errMsg => User not found!"}}`, "reject", false, 0},
		{"not found is never a reject", 200, `{"success":false,"message":"order is not found","data":{"order_id":0,"order_ex_rate":null,"order_value":null,"order_status":"Cancelled","balanceBefore":null,"order_desc":"order is not found"}}`, "", true, 0},
		{"a different order id is not ours", 200, `{"success":true,"data":{"order_id":999,"order_status":"Cancelled"}}`, "", true, 0},
		{"generic failure", 200, `{"message":"some things","success":false,"data":""}`, "", true, 0},
		{"500", 500, ``, "", true, 0},
		{"garbage", 200, `nope`, "", true, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var path string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path = r.URL.RequestURI()
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			st, err := newTestTorosfon(srv.URL).CheckStatus(context.Background(), "123")
			if path != "/api/CheckOrder?order_id=123" {
				t.Errorf("request = %q", path)
			}
			if tc.wantErr {
				if err == nil || !errors.Is(err, ErrUnavailable) || Refused(err) {
					t.Fatalf("err = %v (state %q), want plain ErrUnavailable", err, st.State)
				}
				return
			}
			if err != nil || st.State != tc.wantState || len(st.Codes) != tc.wantCodes {
				t.Fatalf("status = %+v err %v, want %s with %d codes", st, err, tc.wantState, tc.wantCodes)
			}
		})
	}
}

func TestTorosfonCheckStatus_SearchRefWaitsWithoutCalling(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer srv.Close()
	st, err := newTestTorosfon(srv.URL).CheckStatus(context.Background(), "search:torosfon:6af9e7b4c0ffee0123456789")
	if err != nil || st.State != "wait" || calls.Load() != 0 {
		t.Fatalf("status = %+v err %v calls %d, want wait with no request", st, err, calls.Load())
	}
}

func TestTorosfonCheckStatus_NonNumericRefIsNotSent(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer srv.Close()
	_, err := newTestTorosfon(srv.URL).CheckStatus(context.Background(), "ID_abc")
	if !errors.Is(err, ErrUnavailable) || calls.Load() != 0 {
		t.Fatalf("err = %v calls %d, want ErrUnavailable with no request", err, calls.Load())
	}
}

func TestTorosfonProfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/GetMyInfo" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"message":"","data":{"user_id":"a3d0","user_balance":-1797.032925518608,"user_currency":"$","user_email":"x@y.z"}}`))
	}))
	defer srv.Close()
	acc, err := newTestTorosfon(srv.URL).Profile(context.Background())
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if acc.Balance != -1797.032925518608 || acc.Currency != "USD" || acc.Email != "x@y.z" {
		t.Fatalf("account = %+v", acc)
	}
}

func TestTorosfonProbeErrors(t *testing.T) {
	cases := []struct {
		status int
		body   string
		health string
	}{
		{401, ``, "auth_error"},
		{403, ``, "auth_error"},
		{200, `{"message":"some things","success":false,"data":""}`, "unreachable"},
		{500, `<html/>`, "unreachable"},
	}
	for _, tc := range cases {
		srv := torosServer(t, tc.status, tc.body)
		_, err := newTestTorosfon(srv.URL).Profile(context.Background())
		if got := HealthFromError(err); got != tc.health {
			t.Errorf("status %d body %q: health = %s, want %s (err %v)", tc.status, tc.body, got, tc.health, err)
		}
	}
}

func TestTorosfonListProducts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/AllProducts" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"message":"","data":[
			{"product_id":147,"product_name":"سول ستار","product_name_translated":"SoulStar ","product_price":0.0001097163,"min_qty":9000,"max_qty":10000000,"is_active":true},
			{"product_id":8,"product_name":"PUBG 60 UC GLOBAL","product_name_translated":"","product_price":0.9264491999999999,"min_qty":1,"max_qty":1,"is_active":true},
			{"product_id":36,"product_name":"PUBG 60 UC TR","product_name_translated":"PUBG 60 UC TR","product_price":0.869,"min_qty":1,"max_qty":1,"is_active":false},
			{"product_id":372,"product_name":"Asha Live","product_name_translated":"Asha Live","product_price":0,"min_qty":10000,"max_qty":1000000,"is_active":true}
		]}`))
	}))
	defer srv.Close()
	got, err := newTestTorosfon(srv.URL).ListProducts(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("got %d products, want 4", len(got))
	}
	ss := got[0]
	if ss.UpstreamID != "147" || ss.Name != "SoulStar" || ss.Price != 0.0001097163 || ss.BasePrice != ss.Price || ss.Currency != "USD" ||
		!ss.Available || ss.QtyMin == nil || *ss.QtyMin != 9000 || ss.QtyMax == nil || *ss.QtyMax != 10000000 ||
		len(ss.Params) != 1 {
		t.Fatalf("SoulStar = %+v", ss)
	}
	if got[1].Name != "PUBG 60 UC GLOBAL" {
		t.Fatalf("empty translated name must fall back to product_name, got %q", got[1].Name)
	}
	if !AcceptsQty(got[1], 1) || AcceptsQty(got[1], 2) {
		t.Fatal("a fixed pack accepts exactly qty 1")
	}
	if got[2].Available {
		t.Fatal("is_active false must be unavailable")
	}
	if got[3].Price != 0 {
		t.Fatal("price 0 passes through unchanged")
	}
}

func TestTorosfonListProducts_FailureIsNotAnEmptyCatalog(t *testing.T) {
	srv := torosServer(t, 200, `{"message":"no Products are found","success":false,"data":""}`)
	got, err := newTestTorosfon(srv.URL).ListProducts(context.Background())
	if err == nil || got != nil {
		t.Fatalf("got %v err %v, want an error (never an empty known catalog)", got, err)
	}
}

// TestTorosfonCodes pins the live order_desc shapes (real order 1511858 on
// 2026-11-17 delivered `{"replay":["…"]}` as a string) so the customer never
// sees raw JSON as their code.
func TestTorosfonCodes(t *testing.T) {
	cases := []struct {
		name string
		desc string
		want []string
	}{
		{"empty", "", nil},
		{"blank", "   ", nil},
		{"live replay object", `{"replay":["J9ZC-TEST"]}`, []string{"J9ZC-TEST"}},
		{"replay several", `{"replay":["A","","B"]}`, []string{"A", "B"}},
		{"replay empty", `{"replay":[]}`, nil},
		{"other object", `{"note":"done"}`, nil},
		{"flat array", `["X","Y"]`, []string{"X", "Y"}},
		{"nested array", `[{"replay":["Z"]}]`, []string{"Z"}},
		{"plain code", " CODE-XYZ ", []string{"CODE-XYZ"}},
		{"broken json is text", `{"replay":[`, []string{`{"replay":[`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := torosfonCodes(tc.desc)
			if len(got) != len(tc.want) {
				t.Fatalf("torosfonCodes(%q) = %q, want %q", tc.desc, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("torosfonCodes(%q) = %q, want %q", tc.desc, got, tc.want)
				}
			}
		})
	}
}
