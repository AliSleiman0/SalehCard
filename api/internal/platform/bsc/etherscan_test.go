package bsc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/platform/tron"
)

func TestWeiToMicros(t *testing.T) {
	cases := []struct {
		value  string
		micros int64
		ok     bool
	}{
		{"1000000000000", 1, true},                    // exactly 1 micro
		{"10000001000000000000", 10_000_001, true},    // 10.000001 USDT — salt digits intact
		{"10000001000000400321", 10_000_001, true},    // sub-micro dust floors away
		{"10000000000000000000000", 10_000_000_000, true}, // 10,000 USDT = 1e22 wei > int64
		{"999999999999", 0, false},                    // below 1 micro → quotient 0
		{"0", 0, false},
		{"-1000000000000", 0, false},
		{"not-a-number", 0, false},
		// quotient itself past int64 (absurd amount) → rejected, not wrapped
		{"10000000000000000000000000000000", 0, false},
	}
	for _, c := range cases {
		got, ok := weiToMicros(c.value)
		if ok != c.ok || got != c.micros {
			t.Errorf("weiToMicros(%q) = (%d, %v), want (%d, %v)", c.value, got, ok, c.micros, c.ok)
		}
	}
}

// testRow builds a tokentx result row paying `wei` to `to`.
func testRow(hash, to, contract, wei string, ts int64, confirmations string) map[string]string {
	return map[string]string{
		"hash":            hash,
		"from":            "0x1111111111111111111111111111111111111111",
		"to":              to,
		"contractAddress": contract,
		"value":           wei,
		"timeStamp":       fmt.Sprintf("%d", ts),
		"confirmations":   confirmations,
	}
}

// newTestServer serves getblocknobytime (fixed block) and tokentx (the given
// rows), capturing the tokentx query for assertions.
func newTestServer(t *testing.T, rows []map[string]string, capture *map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch q.Get("action") {
		case "getblocknobytime":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "1", "message": "OK", "result": "36000000"})
		case "tokentx":
			if capture != nil {
				m := map[string]string{}
				for k := range q {
					m[k] = q.Get(k)
				}
				*capture = m
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "1", "message": "OK", "result": rows})
		default:
			t.Errorf("unexpected action %q", q.Get("action"))
			http.Error(w, "bad action", http.StatusBadRequest)
		}
	}))
}

func TestEtherscanListTransfers(t *testing.T) {
	addr := "0x5e0a66cedc7688aab52c87dc02bff97f6575d7dc"
	since := time.Unix(1_700_000_000, 0).UTC()

	rows := []map[string]string{
		// Good: checksummed to/contract must match case-insensitively.
		testRow("0xgood", "0x5E0a66CEDC7688AAB52c87dc02bff97f6575d7dc", USDTContractBSC, "10000001000000000000", 1_700_000_100, "99"),
		// Wrong recipient → skipped.
		testRow("0xwrongto", "0x2222222222222222222222222222222222222222", USDTContractBSC, "5000000000000000000", 1_700_000_100, "99"),
		// Wrong token contract → skipped.
		testRow("0xwrongtoken", addr, "0x3333333333333333333333333333333333333333", "5000000000000000000", 1_700_000_100, "99"),
		// Before the window → skipped.
		testRow("0xold", addr, USDTContractBSC, "5000000000000000000", 1_699_999_999, "99"),
		// Too fresh (confirmations below the gate) → skipped this tick.
		testRow("0xfresh", addr, USDTContractBSC, "5000000000000000000", 1_700_000_200, "3"),
		// Dust below one micro → skipped.
		testRow("0xdust", addr, USDTContractBSC, "999", 1_700_000_100, "99"),
	}
	var captured map[string]string
	srv := newTestServer(t, rows, &captured)
	defer srv.Close()

	lister, err := New(Config{Provider: "etherscan", Etherscan: EtherscanConfig{BaseURL: srv.URL, APIKey: "k", MinConfirmations: 15}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := lister.ListTransfers(context.Background(), addr, since, nil)
	if err != nil {
		t.Fatalf("ListTransfers: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d payments, want 1: %+v", len(got), got)
	}
	want := tron.Payment{TxHash: "0xgood", From: "0x1111111111111111111111111111111111111111",
		AmountMicros: 10_000_001, BlockTime: time.Unix(1_700_000_100, 0).UTC()}
	if got[0] != want {
		t.Errorf("payment = %+v, want %+v", got[0], want)
	}

	// The tokentx call carries the multichain + windowing params.
	if captured["chainid"] != "56" || captured["contractaddress"] != USDTContractBSC ||
		captured["address"] != addr || captured["startblock"] != "36000000" ||
		captured["sort"] != "asc" || captured["apikey"] != "k" {
		t.Errorf("tokentx query: %+v", captured)
	}
}

func TestEtherscanNoTransactionsFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") == "getblocknobytime" {
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "1", "message": "OK", "result": "36000000"})
			return
		}
		// Etherscan reports an empty window as status 0, result string.
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "0", "message": "No transactions found", "result": []any{}})
	}))
	defer srv.Close()

	lister, err := New(Config{Provider: "etherscan", Etherscan: EtherscanConfig{BaseURL: srv.URL}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := lister.ListTransfers(context.Background(), "0x5e0a66cedc7688aab52c87dc02bff97f6575d7dc", time.Now().Add(-time.Hour), nil)
	if err != nil {
		t.Fatalf("empty window must not error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d payments, want 0", len(got))
	}
}

func TestEtherscanErrorStatusSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") == "getblocknobytime" {
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "1", "message": "OK", "result": "36000000"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "0", "message": "NOTOK", "result": "Max rate limit reached"})
	}))
	defer srv.Close()

	lister, err := New(Config{Provider: "etherscan", Etherscan: EtherscanConfig{BaseURL: srv.URL}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lister.ListTransfers(context.Background(), "0x5e0a66cedc7688aab52c87dc02bff97f6575d7dc", time.Now().Add(-time.Hour), nil); err == nil {
		t.Fatal("rate-limit status must surface as an error")
	}
}

func TestEtherscanBlockLookupFailureAborts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "0", "message": "NOTOK", "result": "Error! No closest block found"})
	}))
	defer srv.Close()

	lister, err := New(Config{Provider: "etherscan", Etherscan: EtherscanConfig{BaseURL: srv.URL}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lister.ListTransfers(context.Background(), "0x5e0a66cedc7688aab52c87dc02bff97f6575d7dc", time.Now().Add(-time.Hour), nil); err == nil {
		t.Fatal("block lookup failure must surface as an error")
	}
}

func TestNewStubProvider(t *testing.T) {
	lister, err := New(Config{}) // default provider = stub
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := lister.(*tron.Stub); !ok {
		t.Errorf("default provider = %T, want *tron.Stub", lister)
	}
	if _, err := New(Config{Provider: "nope"}); err == nil {
		t.Error("unknown provider must error")
	}
}
