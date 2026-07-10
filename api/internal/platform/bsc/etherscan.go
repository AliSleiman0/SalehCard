package bsc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/platform/tron"
)

const (
	defaultEtherscanBaseURL = "https://api.etherscan.io/v2/api"
	etherscanTimeout        = 10 * time.Second
	// bscChainID selects BNB Smart Chain mainnet on the Etherscan V2
	// multichain API.
	bscChainID = 56
	// listLimit is the tokentx page size. Exactly listLimit rows back means
	// possible truncation (warn-logged; older transfers are picked up as the
	// window's `since` advances) — same known limitation as trongrid.
	listLimit = 200
	// defaultMinConfirmations gates how settled a transfer must be before it
	// counts. BSC blocks land every ~1-3s, so 15 confirmations adds only
	// seconds of latency against a 25s watcher tick.
	defaultMinConfirmations = 15
)

// EtherscanConfig configures the Etherscan V2 adapter.
type EtherscanConfig struct {
	BaseURL string // default https://api.etherscan.io/v2/api
	// APIKey is required in prod. The free tier (5 req/s, 100k/day) is plenty:
	// the watcher makes at most two calls per 25s tick.
	APIKey           string
	ChainID          int    // default 56 (BSC mainnet)
	Contract         string // token contract to match (default USDTContractBSC)
	MinConfirmations int64  // default 15
}

// etherscan reads confirmed BEP20 USDT transfers from the Etherscan V2 REST
// API. It owns no state beyond configuration, a reusable http.Client, and a
// one-entry start-block cache.
type etherscan struct {
	cfg  EtherscanConfig
	http *http.Client

	// getblocknobytime is deterministic for a past timestamp, and the watcher
	// asks with the same `since` every tick while the same intents stay open —
	// cache the answer so most ticks make a single tokentx call.
	mu          sync.Mutex
	cachedSince int64 // unix seconds the cached start block was resolved for
	cachedBlock string
}

func newEtherscan(cfg EtherscanConfig) (*etherscan, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultEtherscanBaseURL
	}
	if cfg.ChainID == 0 {
		cfg.ChainID = bscChainID
	}
	if cfg.Contract == "" {
		cfg.Contract = USDTContractBSC
	}
	if cfg.MinConfirmations <= 0 {
		cfg.MinConfirmations = defaultMinConfirmations
	}
	return &etherscan{cfg: cfg, http: &http.Client{Timeout: etherscanTimeout}}, nil
}

// etherscanResponse is the V2 envelope. Result is an array on success but a
// bare string on errors (rate limits, bad params), hence the RawMessage.
type etherscanResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

// tokenTxRow mirrors one module=account&action=tokentx result row (all fields
// arrive as strings).
type tokenTxRow struct {
	Hash            string `json:"hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	ContractAddress string `json:"contractAddress"`
	Value           string `json:"value"` // integer string in 18-decimal base units
	TimeStamp       string `json:"timeStamp"`
	Confirmations   string `json:"confirmations"`
}

// ListTransfers returns every sufficiently-confirmed USDT transfer into
// address at or after since, oldest first (hints are a stub-only concept,
// ignored here).
func (e *etherscan) ListTransfers(ctx context.Context, address string, since time.Time, _ []tron.AmountHint) ([]tron.Payment, error) {
	start, err := e.startBlock(ctx, since)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("module", "account")
	q.Set("action", "tokentx")
	q.Set("contractaddress", e.cfg.Contract)
	q.Set("address", address)
	q.Set("startblock", start)
	q.Set("page", "1")
	q.Set("offset", strconv.Itoa(listLimit))
	q.Set("sort", "asc")

	out, err := e.call(ctx, q)
	if err != nil {
		return nil, err
	}
	if out.Status != "1" {
		// An empty window is reported as an error-status response, not an
		// empty array — treat only that case as "no transfers yet".
		if strings.Contains(out.Message, "No transactions found") {
			return nil, nil
		}
		return nil, fmt.Errorf("etherscan: tokentx: %s: %s", out.Message, string(out.Result))
	}

	var rows []tokenTxRow
	if err := json.Unmarshal(out.Result, &rows); err != nil {
		return nil, fmt.Errorf("etherscan: decode tokentx result: %w", err)
	}
	if len(rows) == listLimit {
		slog.Warn("etherscan: shared-address transfer list hit the page cap — older transfers may be deferred",
			"address", address, "limit", listLimit)
	}

	sinceUnix := since.Unix()
	payments := make([]tron.Payment, 0, len(rows))
	for _, tx := range rows {
		// tokentx filters by address/contract server-side, but re-check
		// defensively (EVM hex is case-insensitive) so an API quirk can't
		// credit the wrong token or address.
		if !strings.EqualFold(tx.To, address) || !strings.EqualFold(tx.ContractAddress, e.cfg.Contract) {
			continue
		}
		ts, err := strconv.ParseInt(tx.TimeStamp, 10, 64)
		if err != nil || ts < sinceUnix {
			continue
		}
		conf, err := strconv.ParseInt(tx.Confirmations, 10, 64)
		if err != nil || conf < e.cfg.MinConfirmations {
			continue
		}
		micros, ok := weiToMicros(tx.Value)
		if !ok || micros <= 0 {
			continue
		}
		payments = append(payments, tron.Payment{
			TxHash:       tx.Hash,
			From:         tx.From,
			AmountMicros: micros,
			BlockTime:    time.Unix(ts, 0).UTC(),
		})
	}
	return payments, nil
}

// startBlock resolves since into a block number via getblocknobytime
// (closest=before), memoized per since value.
func (e *etherscan) startBlock(ctx context.Context, since time.Time) (string, error) {
	unix := since.Unix()
	e.mu.Lock()
	if e.cachedBlock != "" && e.cachedSince == unix {
		block := e.cachedBlock
		e.mu.Unlock()
		return block, nil
	}
	e.mu.Unlock()

	q := url.Values{}
	q.Set("module", "block")
	q.Set("action", "getblocknobytime")
	q.Set("timestamp", strconv.FormatInt(unix, 10))
	q.Set("closest", "before")

	out, err := e.call(ctx, q)
	if err != nil {
		return "", err
	}
	if out.Status != "1" {
		return "", fmt.Errorf("etherscan: block lookup: %s: %s", out.Message, string(out.Result))
	}
	var block string
	if err := json.Unmarshal(out.Result, &block); err != nil {
		return "", fmt.Errorf("etherscan: decode block lookup result: %w", err)
	}
	if _, err := strconv.ParseUint(block, 10, 64); err != nil {
		return "", fmt.Errorf("etherscan: block lookup returned %q, want a block number", block)
	}

	e.mu.Lock()
	e.cachedSince, e.cachedBlock = unix, block
	e.mu.Unlock()
	return block, nil
}

// call performs one Etherscan V2 request and decodes the envelope (status
// interpretation is per-action, left to the caller).
func (e *etherscan) call(ctx context.Context, q url.Values) (*etherscanResponse, error) {
	q.Set("chainid", strconv.Itoa(e.cfg.ChainID))
	if e.cfg.APIKey != "" {
		q.Set("apikey", e.cfg.APIKey)
	}
	endpoint := e.cfg.BaseURL + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("etherscan: new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("etherscan: http: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("etherscan: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("etherscan: http %d: %s", resp.StatusCode, string(body))
	}

	var out etherscanResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("etherscan: decode: %w", err)
	}
	return &out, nil
}
