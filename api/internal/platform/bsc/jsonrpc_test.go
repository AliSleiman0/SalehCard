package bsc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/platform/tron"
)

const testRecipient = "0x5e0a66cedc7688aab52c87dc02bff97f6575d7dc"

// chainFixture is a scripted BSC chain served over httptest: a latest block,
// per-block timestamps (defaulting to genesisTS + n*blockSecs), and a set of
// logs returned for any getLogs range containing their block.
type chainFixture struct {
	mu        sync.Mutex
	latest       int64
	genesisTS    int64
	blockSecs    int64
	prunedBefore int64 // blocks below this return null (pruned history)
	logs         []fixtureLog
	fail         bool // force HTTP 500 (failover tests)

	getLogsCalls   []map[string]any // captured getLogs filter params
	blockTimeCalls int              // eth_getBlockByNumber count
	requests       int              // total requests served
}

type fixtureLog struct {
	block   int64
	txHash  string
	from    string
	to      string
	value   string // hex data
	removed bool
	topics  []string // optional override
}

func (c *chainFixture) blockTS(n int64) int64 { return c.genesisTS + n*c.blockSecs }

func (c *chainFixture) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.requests++
		if c.fail {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		var req struct {
			ID     int64  `json:"id"`
			Method string `json:"method"`
			Params []any  `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		reply := func(result any) {
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
		}
		switch req.Method {
		case "eth_blockNumber":
			reply(fmt.Sprintf("0x%x", c.latest))
		case "eth_getBlockByNumber":
			c.blockTimeCalls++
			n, _ := hexToInt64(req.Params[0].(string))
			if n < c.prunedBefore {
				reply(nil) // pruned history: nodes answer result:null
				return
			}
			reply(map[string]any{"timestamp": fmt.Sprintf("0x%x", c.blockTS(n))})
		case "eth_getLogs":
			filter := req.Params[0].(map[string]any)
			c.getLogsCalls = append(c.getLogsCalls, filter)
			from, _ := hexToInt64(filter["fromBlock"].(string))
			to, _ := hexToInt64(filter["toBlock"].(string))
			out := []map[string]any{}
			for _, lg := range c.logs {
				if lg.block < from || lg.block > to {
					continue
				}
				topics := lg.topics
				if topics == nil {
					topics = []string{
						transferTopic,
						padTopicAddress(lg.from),
						padTopicAddress(lg.to),
					}
				}
				out = append(out, map[string]any{
					"address":         USDTContractBSC,
					"topics":          topics,
					"data":            lg.value,
					"blockNumber":     fmt.Sprintf("0x%x", lg.block),
					"transactionHash": lg.txHash,
					"removed":         lg.removed,
				})
			}
			reply(out)
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "id": req.ID,
				"error": map[string]any{"code": -32601, "message": "method not found"},
			})
		}
	}
}

// newFixture returns a scripted chain + an adapter pointed at it. Small knobs
// keep the block math easy: 1 block/second from genesisTS.
func newFixture(t *testing.T, latest int64, cfg JSONRPCConfig) (*chainFixture, *jsonrpc, *httptest.Server) {
	t.Helper()
	c := &chainFixture{latest: latest, genesisTS: 1_700_000_000, blockSecs: 1}
	srv := httptest.NewServer(c.handler())
	t.Cleanup(srv.Close)
	cfg.Endpoints = append(cfg.Endpoints, srv.URL)
	j, err := newJSONRPC(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c, j, srv
}

func TestJSONRPCHappyPath(t *testing.T) {
	// latest 20_000, confirmations 15 → safeHead 19_985. since = ts of block
	// 10_000. One matching transfer at block 10_500.
	chain, j, _ := newFixture(t, 20_000, JSONRPCConfig{MinConfirmations: 15})
	chain.logs = []fixtureLog{{
		block: 10_500, txHash: "0xabc", from: "0x1111111111111111111111111111111111111111",
		to: testRecipient, value: "0x0000000000000000000000000000000000000000000000008ac723ed5e8d1000", // 10.000001 USDT
	}}
	since := time.Unix(chain.blockTS(10_000), 0).UTC()

	got, err := j.ListTransfers(context.Background(), testRecipient, since, nil)
	if err != nil {
		t.Fatalf("ListTransfers: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d payments, want 1: %+v", len(got), got)
	}
	want := tron.Payment{
		TxHash: "0xabc", From: "0x1111111111111111111111111111111111111111",
		AmountMicros: 10_000_001, BlockTime: time.Unix(chain.blockTS(10_500), 0).UTC(),
	}
	if got[0] != want {
		t.Errorf("payment = %+v, want %+v", got[0], want)
	}

	// The getLogs filter carries the contract, transfer topic and padded
	// recipient, with hex block bounds.
	if len(chain.getLogsCalls) == 0 {
		t.Fatal("no getLogs calls captured")
	}
	f := chain.getLogsCalls[0]
	if f["address"] != USDTContractBSC {
		t.Errorf("filter address = %v", f["address"])
	}
	topics := f["topics"].([]any)
	if topics[0] != transferTopic || topics[1] != nil || topics[2] != padTopicAddress(testRecipient) {
		t.Errorf("filter topics = %v", topics)
	}
}

func TestJSONRPCFilters(t *testing.T) {
	chain, j, _ := newFixture(t, 20_000, JSONRPCConfig{MinConfirmations: 15})
	okValue := "0xe8d4a51000" // 1 micro
	chain.logs = []fixtureLog{
		// Wrong recipient → the node wouldn't return it, but decode re-checks.
		{block: 10_100, txHash: "0xwrongto", from: "0x1111111111111111111111111111111111111111",
			to: "0x2222222222222222222222222222222222222222", value: okValue},
		// Reorged-out log → skipped.
		{block: 10_200, txHash: "0xremoved", from: "0x1111111111111111111111111111111111111111",
			to: testRecipient, value: okValue, removed: true},
		// Malformed topics → skipped.
		{block: 10_300, txHash: "0xmalformed", to: testRecipient, value: okValue,
			topics: []string{transferTopic}},
		// Dust below 1 micro → skipped.
		{block: 10_400, txHash: "0xdust", from: "0x1111111111111111111111111111111111111111",
			to: testRecipient, value: "0x64"},
		// Before the since window (block 5_000 < since block 10_000) → skipped.
		{block: 5_000, txHash: "0xold", from: "0x1111111111111111111111111111111111111111",
			to: testRecipient, value: okValue},
		// Good one.
		{block: 10_500, txHash: "0xgood", from: "0x1111111111111111111111111111111111111111",
			to: testRecipient, value: okValue},
	}
	since := time.Unix(chain.blockTS(10_000), 0).UTC()

	got, err := j.ListTransfers(context.Background(), testRecipient, since, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].TxHash != "0xgood" {
		t.Errorf("payments = %+v, want only 0xgood", got)
	}
}

func TestJSONRPCConfirmationsBound(t *testing.T) {
	// Transfer 5 blocks from the tip: below 15 confirmations → not scanned.
	chain, j, _ := newFixture(t, 20_000, JSONRPCConfig{MinConfirmations: 15})
	chain.logs = []fixtureLog{{
		block: 19_995, txHash: "0xfresh", from: "0x1111111111111111111111111111111111111111",
		to: testRecipient, value: "0xe8d4a51000",
	}}
	since := time.Unix(chain.blockTS(19_900), 0).UTC()

	got, err := j.ListTransfers(context.Background(), testRecipient, since, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("unconfirmed transfer returned: %+v", got)
	}

	// Chain advances → now ≥15 confirmations → returned.
	chain.mu.Lock()
	chain.latest = 20_050
	chain.mu.Unlock()
	got, err = j.ListTransfers(context.Background(), testRecipient, since, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].TxHash != "0xfresh" {
		t.Errorf("payments after confirmation = %+v", got)
	}
}

func TestJSONRPCChunkingAndCap(t *testing.T) {
	// since block 1_000, safeHead 19_985; chunk 2_000 × cap 3 = 6_000 blocks
	// per tick → first tick covers 1_000..6_999, truncated by the cap.
	chain, j, _ := newFixture(t, 20_000, JSONRPCConfig{
		MinConfirmations: 15, ChunkSize: 2_000, MaxChunksPerTick: 3, OverlapBlocks: 100,
	})
	since := time.Unix(chain.blockTS(1_000), 0).UTC()

	if _, err := j.ListTransfers(context.Background(), testRecipient, since, nil); err != nil {
		t.Fatal(err)
	}
	if len(chain.getLogsCalls) != 3 {
		t.Fatalf("first tick made %d getLogs calls, want 3", len(chain.getLogsCalls))
	}
	assertRange := func(call map[string]any, wantFrom, wantTo int64) {
		t.Helper()
		from, _ := hexToInt64(call["fromBlock"].(string))
		to, _ := hexToInt64(call["toBlock"].(string))
		if from != wantFrom || to != wantTo {
			t.Errorf("range = %d..%d, want %d..%d", from, to, wantFrom, wantTo)
		}
	}
	assertRange(chain.getLogsCalls[0], 1_000, 2_999)
	assertRange(chain.getLogsCalls[1], 3_000, 4_999)
	assertRange(chain.getLogsCalls[2], 5_000, 6_999)

	// Second tick resumes from cursor − overlap (6_999 − 100 + 1 = 6_900).
	chain.mu.Lock()
	chain.getLogsCalls = nil
	chain.mu.Unlock()
	if _, err := j.ListTransfers(context.Background(), testRecipient, since, nil); err != nil {
		t.Fatal(err)
	}
	if len(chain.getLogsCalls) != 3 {
		t.Fatalf("second tick made %d getLogs calls, want 3", len(chain.getLogsCalls))
	}
	assertRange(chain.getLogsCalls[0], 6_900, 8_899)
}

func TestJSONRPCOverlapRedelivery(t *testing.T) {
	chain, j, _ := newFixture(t, 20_000, JSONRPCConfig{MinConfirmations: 15, OverlapBlocks: 1_000})
	chain.logs = []fixtureLog{{
		block: 19_900, txHash: "0xrecent", from: "0x1111111111111111111111111111111111111111",
		to: testRecipient, value: "0xe8d4a51000",
	}}
	since := time.Unix(chain.blockTS(19_000), 0).UTC()

	for i := range 2 {
		got, err := j.ListTransfers(context.Background(), testRecipient, since, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].TxHash != "0xrecent" {
			t.Fatalf("call %d: payments = %+v, want 0xrecent re-delivered", i+1, got)
		}
	}
}

func TestJSONRPCSinceResolutionCachedAndReset(t *testing.T) {
	chain, j, _ := newFixture(t, 100_000, JSONRPCConfig{MinConfirmations: 15})
	since := time.Unix(chain.blockTS(50_000), 0).UTC()

	if _, err := j.ListTransfers(context.Background(), testRecipient, since, nil); err != nil {
		t.Fatal(err)
	}
	firstProbes := chain.blockTimeCalls
	if firstProbes == 0 || firstProbes > 40 {
		t.Fatalf("binary search made %d getBlockByNumber probes, want 1..40", firstProbes)
	}
	// The resolved window must start at the since block.
	from, _ := hexToInt64(chain.getLogsCalls[0]["fromBlock"].(string))
	if from != 50_000 {
		t.Errorf("first window starts at %d, want 50000 (block whose ts == since)", from)
	}

	// Same since again → cursor path, zero extra probes.
	chain.mu.Lock()
	chain.blockTimeCalls = 0
	chain.mu.Unlock()
	if _, err := j.ListTransfers(context.Background(), testRecipient, since, nil); err != nil {
		t.Fatal(err)
	}
	if chain.blockTimeCalls != 0 {
		t.Errorf("cached call made %d probes, want 0", chain.blockTimeCalls)
	}

	// An EARLIER since (cannot happen with today's watcher, honored anyway)
	// resets the cursor and re-resolves.
	earlier := time.Unix(chain.blockTS(40_000), 0).UTC()
	chain.mu.Lock()
	chain.getLogsCalls = nil
	chain.mu.Unlock()
	if _, err := j.ListTransfers(context.Background(), testRecipient, earlier, nil); err != nil {
		t.Fatal(err)
	}
	from, _ = hexToInt64(chain.getLogsCalls[0]["fromBlock"].(string))
	if from != 40_000 {
		t.Errorf("reset window starts at %d, want 40000", from)
	}
}

func TestJSONRPCPrunedHistory(t *testing.T) {
	// The node prunes everything below block 60_000; since maps to block
	// 50_000 — the search must converge onto the oldest servable block
	// instead of erroring out.
	chain, j, _ := newFixture(t, 100_000, JSONRPCConfig{MinConfirmations: 15})
	chain.prunedBefore = 60_000
	since := time.Unix(chain.blockTS(50_000), 0).UTC()

	if _, err := j.ListTransfers(context.Background(), testRecipient, since, nil); err != nil {
		t.Fatalf("pruned history must not fail the scan: %v", err)
	}
	from, _ := hexToInt64(chain.getLogsCalls[0]["fromBlock"].(string))
	if from < 60_000 || from > 60_001 {
		t.Errorf("window starts at %d, want the pruning horizon (~60000)", from)
	}
}

func TestJSONRPCLookbackBound(t *testing.T) {
	// A chain taller than maxLookbackBlocks: the search must never probe
	// blocks older than safeHead − maxLookbackBlocks (free nodes prune them).
	chain, j, _ := newFixture(t, 2_500_000, JSONRPCConfig{MinConfirmations: 15})
	chain.prunedBefore = 1_200_000 // anything older would blow up as null…
	since := time.Unix(chain.blockTS(1_600_000), 0).UTC()

	if _, err := j.ListTransfers(context.Background(), testRecipient, since, nil); err != nil {
		t.Fatalf("ListTransfers: %v", err)
	}
	from, _ := hexToInt64(chain.getLogsCalls[0]["fromBlock"].(string))
	if from != 1_600_000 {
		t.Errorf("window starts at %d, want 1600000", from)
	}
}

func TestJSONRPCEndpointFailover(t *testing.T) {
	bad := &chainFixture{fail: true}
	badSrv := httptest.NewServer(bad.handler())
	defer badSrv.Close()

	good := &chainFixture{latest: 20_000, genesisTS: 1_700_000_000, blockSecs: 1}
	goodSrv := httptest.NewServer(good.handler())
	defer goodSrv.Close()

	j, err := newJSONRPC(JSONRPCConfig{
		Endpoints: []string{badSrv.URL, goodSrv.URL}, MinConfirmations: 15,
	})
	if err != nil {
		t.Fatal(err)
	}
	since := time.Unix(good.blockTS(19_000), 0).UTC()
	if _, err := j.ListTransfers(context.Background(), testRecipient, since, nil); err != nil {
		t.Fatalf("failover call failed: %v", err)
	}
	badBefore := bad.requests

	// Subsequent calls stick to the good endpoint — the bad one sees nothing.
	if _, err := j.ListTransfers(context.Background(), testRecipient, since, nil); err != nil {
		t.Fatal(err)
	}
	if bad.requests != badBefore {
		t.Errorf("bad endpoint hit again after failover (%d → %d requests)", badBefore, bad.requests)
	}
	if good.requests == 0 {
		t.Error("good endpoint never used")
	}
}

func TestJSONRPCEmptyWindowNoGetLogs(t *testing.T) {
	// Cursor is at safeHead already → next tick has nothing new to scan and
	// only the overlap; with overlap larger than the gap the window is still
	// valid, so instead test the no-safe-head case: latest below confirmations.
	chain, j, _ := newFixture(t, 10, JSONRPCConfig{MinConfirmations: 15})
	got, err := j.ListTransfers(context.Background(), testRecipient, time.Unix(chain.blockTS(1), 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("payments = %+v, want nil", got)
	}
	if len(chain.getLogsCalls) != 0 {
		t.Errorf("getLogs called %d times on an empty chain, want 0", len(chain.getLogsCalls))
	}
}

func TestJSONRPCErrorEnvelopeSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": 1,
			"error": map[string]any{"code": -32005, "message": "limit exceeded"},
		})
	}))
	defer srv.Close()
	j, err := newJSONRPC(JSONRPCConfig{Endpoints: []string{srv.URL}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = j.ListTransfers(context.Background(), testRecipient, time.Now().Add(-time.Hour), nil)
	if err == nil || !strings.Contains(err.Error(), "limit exceeded") {
		t.Fatalf("rpc error not surfaced: %v", err)
	}
}

func TestNewJSONRPCProvider(t *testing.T) {
	lister, err := New(Config{Provider: "jsonrpc"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := lister.(*jsonrpc); !ok {
		t.Errorf("provider jsonrpc = %T, want *jsonrpc", lister)
	}
}
