package tron

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTronGridBaseURL = "https://api.trongrid.io"
	tronGridTimeout        = 10 * time.Second
	// findLimit is the page size for the single-address FindPayment path; a
	// fresh derived address sees at most a handful of transfers.
	findLimit = 50
	// listLimit is the page size for the shared-address ListTransfers path —
	// the endpoint's maximum. Exactly listLimit rows back means possible
	// truncation (warn-logged; older transfers are picked up as the window's
	// `since` advances).
	listLimit = 200
)

// TronGridConfig configures the TronGrid adapter.
type TronGridConfig struct {
	BaseURL  string // default https://api.trongrid.io
	APIKey   string // TRON-PRO-API-KEY header (raises the rate limit; required in prod)
	Contract string // token contract to match (default USDTContract)
}

// tronGrid reads confirmed TRC20 transfers from the TronGrid REST API. It owns
// no state beyond configuration and a reusable http.Client.
type tronGrid struct {
	cfg  TronGridConfig
	http *http.Client
}

func newTronGrid(cfg TronGridConfig) (*tronGrid, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultTronGridBaseURL
	}
	if cfg.Contract == "" {
		cfg.Contract = USDTContract
	}
	return &tronGrid{cfg: cfg, http: &http.Client{Timeout: tronGridTimeout}}, nil
}

// trc20Response mirrors GET /v1/accounts/{address}/transactions/trc20.
type trc20Response struct {
	Success bool `json:"success"`
	Data    []struct {
		TransactionID  string `json:"transaction_id"`
		From           string `json:"from"`
		To             string `json:"to"`
		Value          string `json:"value"` // integer string in token base units
		BlockTimestamp int64  `json:"block_timestamp"`
		TokenInfo      struct {
			Address string `json:"address"`
		} `json:"token_info"`
	} `json:"data"`
}

// FindPayment returns the earliest confirmed USDT transfer into w.Address at or
// after w.CreatedAt, or (nil, nil) when none exists yet.
func (t *tronGrid) FindPayment(ctx context.Context, w Watch) (*Payment, error) {
	list, err := t.fetchTransfers(ctx, w.Address, w.CreatedAt, findLimit)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return &list[0], nil
}

// ListTransfers returns every confirmed USDT transfer into address at or after
// since, oldest first (hints are a stub-only concept, ignored here).
func (t *tronGrid) ListTransfers(ctx context.Context, address string, since time.Time, _ []AmountHint) ([]Payment, error) {
	list, err := t.fetchTransfers(ctx, address, since, listLimit)
	if err == nil && len(list) == listLimit {
		slog.Warn("trongrid: shared-address transfer list hit the page cap — older transfers may be deferred",
			"address", address, "limit", listLimit)
	}
	return list, err
}

// fetchTransfers is the shared TronGrid call: confirmed inbound USDT transfers
// into address at or after since, oldest first, up to limit rows.
func (t *tronGrid) fetchTransfers(ctx context.Context, address string, since time.Time, limit int) ([]Payment, error) {
	q := url.Values{}
	q.Set("only_confirmed", "true")
	q.Set("only_to", "true")
	q.Set("limit", strconv.Itoa(limit))
	q.Set("contract_address", t.cfg.Contract)
	q.Set("min_timestamp", strconv.FormatInt(since.UnixMilli(), 10))
	q.Set("order_by", "block_timestamp,asc")

	endpoint := fmt.Sprintf("%s/v1/accounts/%s/transactions/trc20?%s",
		strings.TrimRight(t.cfg.BaseURL, "/"), url.PathEscape(address), q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("trongrid: new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if t.cfg.APIKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", t.cfg.APIKey)
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("trongrid: http: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("trongrid: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("trongrid: http %d: %s", resp.StatusCode, string(body))
	}

	var out trc20Response
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("trongrid: decode: %w", err)
	}
	if !out.Success {
		return nil, errors.New("trongrid: response success=false")
	}

	payments := make([]Payment, 0, len(out.Data))
	for _, tx := range out.Data {
		// only_to filters server-side, but re-check defensively; the contract
		// filter also re-checks so an API quirk can't credit the wrong token.
		if tx.To != address || tx.TokenInfo.Address != t.cfg.Contract {
			continue
		}
		// Value is an integer string in 6-decimal base units — never a float.
		amount, err := strconv.ParseInt(tx.Value, 10, 64)
		if err != nil || amount <= 0 {
			continue
		}
		payments = append(payments, Payment{
			TxHash:       tx.TransactionID,
			From:         tx.From,
			AmountMicros: amount,
			BlockTime:    time.UnixMilli(tx.BlockTimestamp).UTC(),
		})
	}
	return payments, nil
}
