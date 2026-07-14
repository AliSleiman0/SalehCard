package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Panel is the shared adapter for the white-label game top-up panels
// (jentel-cash / speedcard / gift4card): they expose an identical
// /client/api surface authenticated by an api-token header, so one adapter
// serves all three — each supplier is a separate Panel instance registered
// under its own provider id.
//
// Error contract (design DESIGN-SUPPLIERS.md Phase 1): outcomes that are
// definitively about THIS order (rejected, invalid quantity, blocked player)
// return a plain error → the order engine compensates. Everything
// environmental — supplier balance, throttling, auth/IP/maintenance, product
// withdrawn upstream, transport or decode ambiguity — wraps ErrUnavailable →
// the order parks and is retried with the same order_uuid (upstream-idempotent).
// A "wait" acceptance wraps ErrPending with Result.Reference set.
//
// Sensitive inputs (PlayerID, Fields values) are never logged here (spec §3.2).
type Panel struct {
	id      int
	name    string
	baseURL string // override point for tests
	token   string
	http    *http.Client
}

// PanelConfig configures one panel supplier instance.
type PanelConfig struct {
	ID      int    // registry id the adapter registers under
	Name    string // supplier slug ("jentel", "speedcard", "gift4card") — logging/errors only
	BaseURL string // e.g. https://api.jentel-cash.com
	Token   string // api-token header value
}

// NewPanel builds a panel adapter, erroring on missing config (the caller
// skips a misconfigured supplier so its id falls back to the parking stub).
func NewPanel(cfg PanelConfig) (*Panel, error) {
	if cfg.ID <= 0 || cfg.Name == "" || cfg.BaseURL == "" || cfg.Token == "" {
		return nil, fmt.Errorf("panel: id, name, base url, and token are required")
	}
	return &Panel{
		id:      cfg.ID,
		name:    cfg.Name,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		token:   cfg.Token,
		http:    &http.Client{Timeout: 20 * time.Second},
	}, nil
}

// ID is the numeric provider id this adapter registers under.
func (p *Panel) ID() int { return p.id }

// panelEnvelope is the common panel reply wrapper. Error replies carry a
// numeric code; the JSON body — not the HTTP status — is authoritative
// (mirrors the rapidapi adapter's contract).
type panelEnvelope struct {
	Status  string          `json:"status"`
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Msg     string          `json:"msg"`
	Data    json.RawMessage `json:"data"`
}

// panelOrder is one order object as returned by newOrder (data = object) and
// check (data = array of these). order_id may arrive as a string or a number;
// replay_api is null, a flat string array (check), or [{replay:[...]}] (newOrder).
type panelOrder struct {
	OrderID   flexString      `json:"order_id"`
	Status    string          `json:"status"`
	ReplayAPI json.RawMessage `json:"replay_api"`
}

// flexString decodes a JSON string, number, or null into a string.
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		*f = ""
		return nil
	}
	if strings.HasPrefix(s, `"`) {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*f = flexString(v)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = flexString(n.String())
	return nil
}

// Fulfill places one order at the panel:
// GET {base}/client/api/newOrder/{upstreamID}/params?<fields...>&qty=N&order_uuid=U.
// qty and order_uuid are set last so they always win over a same-named field key.
func (p *Panel) Fulfill(ctx context.Context, in FulfillInput) (Result, error) {
	if in.UpstreamID == "" {
		// Product mapped to this supplier without an upstream id: a config
		// problem, not an order problem — park, admin fixes the mapping.
		return Result{}, fmt.Errorf("panel %s: product %s has no upstream product id: %w", p.name, in.ProductID, ErrUnavailable)
	}

	q := url.Values{}
	for k, v := range in.Fields {
		q.Set(k, v)
	}
	// The first input field doubles as playerId by convention; ensure it is
	// present even for products whose field key differs from "playerId" only
	// when no explicit field claimed that key.
	if in.PlayerID != "" && q.Get("playerId") == "" {
		q.Set("playerId", in.PlayerID)
	}
	q.Set("qty", strconv.Itoa(in.Qty))
	q.Set("order_uuid", in.OrderUUID)

	endpoint := fmt.Sprintf("%s/client/api/newOrder/%s/params?%s", p.baseURL, url.PathEscape(in.UpstreamID), q.Encode())
	env, err := p.get(ctx, endpoint)
	if err != nil {
		return Result{}, err
	}
	if !strings.EqualFold(env.Status, "OK") {
		return Result{}, p.classify(env)
	}

	var ord panelOrder
	if err := json.Unmarshal(env.Data, &ord); err != nil {
		return Result{}, fmt.Errorf("panel %s: undecodable order data: %w", p.name, ErrUnavailable)
	}
	res := Result{Reference: string(ord.OrderID), Codes: parseReplayCodes(ord.ReplayAPI)}
	switch ord.Status {
	case "accept":
		return res, nil
	case "wait":
		return res, fmt.Errorf("panel %s: order %s pending: %w", p.name, res.Reference, ErrPending)
	case "reject":
		return Result{}, fmt.Errorf("panel %s: order rejected upstream", p.name)
	default:
		// An acknowledged order in an unknown state is ambiguous — park.
		return res, fmt.Errorf("panel %s: unknown order status %q: %w", p.name, ord.Status, ErrUnavailable)
	}
}

// CheckStatus reconciles one previously-pending order:
// GET {base}/client/api/check?orders=[ref].
func (p *Panel) CheckStatus(ctx context.Context, ref string) (Status, error) {
	q := url.Values{}
	q.Set("orders", "["+ref+"]")
	env, err := p.get(ctx, p.baseURL+"/client/api/check?"+q.Encode())
	if err != nil {
		return Status{}, err
	}
	if !strings.EqualFold(env.Status, "OK") {
		return Status{}, p.classify(env)
	}
	var orders []panelOrder
	if err := json.Unmarshal(env.Data, &orders); err != nil || len(orders) == 0 {
		return Status{}, fmt.Errorf("panel %s: undecodable check data: %w", p.name, ErrUnavailable)
	}
	ord := orders[0]
	return Status{State: ord.Status, Codes: parseReplayCodes(ord.ReplayAPI)}, nil
}

// Verify is not supported by the panels — game-ID verification stays on
// platform/idcheck.
func (p *Panel) Verify(_ context.Context, _ FulfillInput) (AccountInfo, error) {
	return AccountInfo{}, ErrNotImplemented
}

// Profile (balance probe) and catalog listing are Phase 3 (admin /suppliers
// page) — they will be added as non-Provider methods reached via an interface
// assertion so the fulfillment port stays slim.

// get performs an authenticated GET and decodes the envelope. Transport
// errors, non-2xx statuses, and undecodable bodies all wrap ErrUnavailable:
// the customer has paid and we cannot prove the upstream didn't take the
// order, so the engine must park, never compensate.
func (p *Panel) get(ctx context.Context, endpoint string) (panelEnvelope, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return panelEnvelope{}, fmt.Errorf("panel %s: build request: %w", p.name, ErrUnavailable)
	}
	req.Header.Set("api-token", p.token)

	resp, err := p.http.Do(req)
	if err != nil {
		return panelEnvelope{}, fmt.Errorf("panel %s: request failed: %w", p.name, ErrUnavailable)
	}
	defer resp.Body.Close()

	var env panelEnvelope
	decodeErr := json.NewDecoder(resp.Body).Decode(&env)
	// A decodable error body beats the transport status (in-body codes arrive
	// on non-2xx too); an undecodable non-2xx is plain ambiguity.
	if decodeErr != nil {
		return panelEnvelope{}, fmt.Errorf("panel %s: status %d, undecodable body: %w", p.name, resp.StatusCode, ErrUnavailable)
	}
	if (resp.StatusCode < 200 || resp.StatusCode >= 300) && env.Status == "" && env.Code == 0 {
		return panelEnvelope{}, fmt.Errorf("panel %s: status %d: %w", p.name, resp.StatusCode, ErrUnavailable)
	}
	return env, nil
}

// classify maps a panel error reply onto the port's error contract.
// Order-specific codes hard-fail (the engine compensates); everything else —
// balance, throttle, auth/IP, maintenance, product withdrawn, unknown — parks.
func (p *Panel) classify(env panelEnvelope) error {
	msg := env.Message
	if msg == "" {
		msg = env.Msg
	}
	switch env.Code {
	case 105, 106, 112, 113:
		// Quantity invalid: a product/qty-constraint misconfiguration on our
		// side — fail the order (customer refunded) and log loudly upstream.
		return fmt.Errorf("panel %s: invalid quantity (code %d): %s", p.name, env.Code, msg)
	case 107:
		return fmt.Errorf("panel %s: player id blocked (code %d)", p.name, env.Code)
	default:
		// 100 balance, 109/110 product gone, 111 throttle, 108 2FA,
		// 120-123/130 auth/IP/maintenance, and anything unrecognized.
		return fmt.Errorf("panel %s: upstream error code %d: %s: %w", p.name, env.Code, msg, ErrUnavailable)
	}
}

// parseReplayCodes extracts delivered codes from a panel replay_api payload,
// which is null, a flat string array (check endpoint), or a nested
// [{"replay": ["..."]}] array (newOrder endpoint). Unknown shapes yield nil —
// delivery still completes; the code is recoverable via CheckStatus/support.
func parseReplayCodes(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var flat []string
	if err := json.Unmarshal(raw, &flat); err == nil {
		return compactStrings(flat)
	}
	var nested []struct {
		Replay []string `json:"replay"`
	}
	if err := json.Unmarshal(raw, &nested); err == nil {
		var out []string
		for _, n := range nested {
			out = append(out, n.Replay...)
		}
		return compactStrings(out)
	}
	return nil
}

// compactStrings drops empty entries, returning nil for an all-empty slice.
func compactStrings(in []string) []string {
	var out []string
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}
