package tron

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testAddr = "TUEZSdKsoDHQMeZwihtdoBiN46zxhGWYdH"

// transferJSON renders one TronGrid trc20 transfer row.
func transferJSON(txID, from, to, value, contract string, blockMs int64) string {
	return fmt.Sprintf(`{
		"transaction_id": %q, "from": %q, "to": %q, "value": %q,
		"block_timestamp": %d, "token_info": {"address": %q}
	}`, txID, from, to, value, blockMs, contract)
}

func newTestServer(t *testing.T, wantAPIKey string, rows ...string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wantAPIKey != "" && r.Header.Get("TRON-PRO-API-KEY") != wantAPIKey {
			t.Errorf("missing/wrong TRON-PRO-API-KEY header: %q", r.Header.Get("TRON-PRO-API-KEY"))
		}
		if got := r.URL.Query().Get("only_confirmed"); got != "true" {
			t.Errorf("only_confirmed = %q, want true", got)
		}
		if got := r.URL.Query().Get("contract_address"); got != USDTContract {
			t.Errorf("contract_address = %q, want %q", got, USDTContract)
		}
		w.Header().Set("Content-Type", "application/json")
		body := `{"success": true, "data": [` + strings.Join(rows, ",") + `]}`
		_, _ = w.Write([]byte(body))
	}))
}

func newTestReader(t *testing.T, baseURL, apiKey string) Reader {
	t.Helper()
	r, err := New(Config{Provider: "trongrid", TronGrid: TronGridConfig{BaseURL: baseURL, APIKey: apiKey}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

func TestTronGridFindPayment(t *testing.T) {
	now := time.Now()
	watch := Watch{Address: testAddr, CreatedAt: now.Add(-time.Hour), ExpectedMicros: 50_000_000}

	t.Run("match returns earliest transfer", func(t *testing.T) {
		srv := newTestServer(t, "key123",
			transferJSON("tx1", "TSender111", testAddr, "50000000", USDTContract, now.UnixMilli()),
			transferJSON("tx2", "TSender222", testAddr, "1000000", USDTContract, now.UnixMilli()+1),
		)
		defer srv.Close()
		p, err := newTestReader(t, srv.URL, "key123").FindPayment(context.Background(), watch)
		if err != nil {
			t.Fatalf("FindPayment: %v", err)
		}
		if p == nil {
			t.Fatal("expected a payment, got nil")
		}
		if p.TxHash != "tx1" || p.From != "TSender111" || p.AmountMicros != 50_000_000 {
			t.Errorf("unexpected payment: %+v", p)
		}
	})

	t.Run("no transfers yields nil,nil", func(t *testing.T) {
		srv := newTestServer(t, "")
		defer srv.Close()
		p, err := newTestReader(t, srv.URL, "").FindPayment(context.Background(), watch)
		if err != nil || p != nil {
			t.Fatalf("want nil,nil; got %+v, %v", p, err)
		}
	})

	t.Run("wrong token contract is skipped", func(t *testing.T) {
		srv := newTestServer(t, "",
			transferJSON("txX", "TSender111", testAddr, "50000000", "TWrongContract0000000000000000000", now.UnixMilli()),
		)
		defer srv.Close()
		p, err := newTestReader(t, srv.URL, "").FindPayment(context.Background(), watch)
		if err != nil || p != nil {
			t.Fatalf("want nil,nil for wrong contract; got %+v, %v", p, err)
		}
	})

	t.Run("outbound transfer is skipped", func(t *testing.T) {
		srv := newTestServer(t, "",
			transferJSON("txO", testAddr, "TSomeoneElse11111111111111111111", "50000000", USDTContract, now.UnixMilli()),
		)
		defer srv.Close()
		p, err := newTestReader(t, srv.URL, "").FindPayment(context.Background(), watch)
		if err != nil || p != nil {
			t.Fatalf("want nil,nil for outbound tx; got %+v, %v", p, err)
		}
	})

	t.Run("unparseable value is skipped", func(t *testing.T) {
		srv := newTestServer(t, "",
			transferJSON("txB", "TSender111", testAddr, "not-a-number", USDTContract, now.UnixMilli()),
		)
		defer srv.Close()
		p, err := newTestReader(t, srv.URL, "").FindPayment(context.Background(), watch)
		if err != nil || p != nil {
			t.Fatalf("want nil,nil for bad value; got %+v, %v", p, err)
		}
	})

	t.Run("http error surfaces", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "quota exceeded", http.StatusTooManyRequests)
		}))
		defer srv.Close()
		if _, err := newTestReader(t, srv.URL, "").FindPayment(context.Background(), watch); err == nil {
			t.Fatal("expected an error for HTTP 429, got nil")
		}
	})
}

func TestNewUnknownProvider(t *testing.T) {
	if _, err := New(Config{Provider: "etherscan"}); err == nil {
		t.Fatal("expected an error for an unknown provider")
	}
}

func TestStubPaysAfterDelay(t *testing.T) {
	s := NewStub(StubConfig{Delay: time.Minute})
	base := time.Now()
	w := Watch{Address: testAddr, CreatedAt: base, ExpectedMicros: 25_000_000}

	s.now = func() time.Time { return base.Add(30 * time.Second) }
	if p, err := s.FindPayment(context.Background(), w); err != nil || p != nil {
		t.Fatalf("before delay: want nil,nil; got %+v, %v", p, err)
	}

	s.now = func() time.Time { return base.Add(2 * time.Minute) }
	p, err := s.FindPayment(context.Background(), w)
	if err != nil || p == nil {
		t.Fatalf("after delay: want payment; got %+v, %v", p, err)
	}
	if p.AmountMicros != 25_000_000 || p.TxHash != "stub-"+testAddr {
		t.Errorf("unexpected stub payment: %+v", p)
	}
}
