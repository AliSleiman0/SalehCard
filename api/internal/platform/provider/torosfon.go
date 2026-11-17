package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Torosfon is the adapter for torosfon-store.com, a wholesale game/app top-up
// store on ASP.NET with its own small API (different paths, params, envelope
// and status words from the white-label panels, and no numeric error codes):
//
//	GET /api/GetMyInfo                                   balance
//	GET /api/AllProducts                                 catalog
//	GET /api/PlaceOrder/{id}/data?amount=&player_Id=&referenceId=
//	GET /api/CheckOrder?order_id={torosfon int id}
//
// Every reply is {success, message, data}. Auth is an `api-token` header (a
// Bearer header works too).
//
// Two facts measured against the live API (2026-11-17) shape the error
// contract:
//
//  1. referenceId is NOT an idempotency key: the same referenceId sent three
//     times created three orders, and no endpoint can find an order by it.
//     So a send whose outcome we cannot read (transport failure, timeout, 5xx,
//     an undecodable body, a reply without an order id that isn't an error)
//     must NEVER be retried: it returns ErrPending with a
//     "search:torosfon:{orderUUID}" reference. The settler then polls it
//     (CheckStatus answers "wait"), never re-sends, refunds or falls back, and
//     flags it stuck for a human after the give-up window. The admin reroute
//     already demands an explicit acknowledgement for "search:" refs.
//  2. Classification keys on data.order_id, not on success: an upstream
//     rejection is success:false WITH a real order id and status "Cancelled"
//     (e.g. "User not found!"), while a validation refusal is success:false
//     with order_id null ("Product Not Found", "The product is not active…",
//     "The requested quantity is not in allowed range…"). The latter created
//     nothing, so it is a definite refusal (fallback-safe).
//
// Sensitive inputs (PlayerID, Fields values) are never logged here (spec §3.2).
type Torosfon struct {
	id       int
	name     string
	currency string
	baseURL  string // override point for tests
	token    string
	http     *http.Client
}

// TorosfonConfig configures the torosfon supplier.
type TorosfonConfig struct {
	ID       int    // registry id the adapter registers under
	Name     string // supplier slug ("torosfon") — logging/errors only
	Currency string // balance currency for the admin probe; defaults to USD
	BaseURL  string // e.g. https://torosfon-store.com
	Token    string // api-token header value
}

// torosfonTimeout is generous on purpose: Torosfon answers PlaceOrder
// synchronously after asking its own upstream (~1–2 s measured), and every
// timeout becomes a "search:" order a human has to resolve.
const torosfonTimeout = 30 * time.Second

// NewTorosfon builds the adapter, erroring on missing config (the caller skips
// a misconfigured supplier so its id falls back to the parking stub).
func NewTorosfon(cfg TorosfonConfig) (*Torosfon, error) {
	if cfg.ID <= 0 || cfg.Name == "" || cfg.BaseURL == "" || cfg.Token == "" {
		return nil, fmt.Errorf("torosfon: id, name, base url, and token are required")
	}
	currency := cfg.Currency
	if currency == "" {
		currency = "USD"
	}
	return &Torosfon{
		id:       cfg.ID,
		name:     cfg.Name,
		currency: currency,
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		token:    cfg.Token,
		http:     &http.Client{Timeout: torosfonTimeout},
	}, nil
}

// ID is the numeric provider id this adapter registers under.
func (t *Torosfon) ID() int { return t.id }

// torosfonEnvelope is the common reply wrapper; data is an order object, a
// list, the account object, or "" / null on failure.
type torosfonEnvelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// torosfonOrder is one order object (PlaceOrder / CheckOrder data). order_id
// is null on a validation refusal and 0 on "order is not found".
type torosfonOrder struct {
	OrderID flexString `json:"order_id"`
	Status  string     `json:"order_status"`
	Desc    string     `json:"order_desc"`
}

// id returns the upstream order id, or "" when there is none (null / 0).
func (o torosfonOrder) id() string {
	s := strings.TrimSpace(string(o.OrderID))
	if s == "" || s == "0" {
		return ""
	}
	return s
}

// torosfonState maps Torosfon's order_status onto the port's vocabulary.
// Documented values are Completed / Pending / Cancelled; a few likely synonyms
// are accepted, and anything else is "" (unknown — the caller keeps waiting).
func torosfonState(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "complete", "accepted", "accept", "success":
		return "accept"
	case "cancelled", "canceled", "rejected", "reject", "failed":
		return "reject"
	case "pending", "processing", "waiting", "wait", "inprogress", "in progress":
		return "wait"
	default:
		return ""
	}
}

// torosfonCodes turns order_desc into the delivered codes. Live (2026-11-17,
// real Google Play order 1511858) a card arrives as a JSON string
// `{"replay":["CODE"]}` — the panel's replay shape. Arrays use the panel's
// flat/nested parsing; any other text is passed through as one code. JSON
// that decodes but carries no codes yields nil, never the raw JSON.
func torosfonCodes(desc string) []string {
	d := strings.TrimSpace(desc)
	if d == "" {
		return nil
	}
	switch d[0] {
	case '{':
		var obj struct {
			Replay []string `json:"replay"`
		}
		if err := json.Unmarshal([]byte(d), &obj); err == nil {
			return compactStrings(obj.Replay)
		}
	case '[':
		if json.Valid([]byte(d)) {
			return parseReplayCodes(json.RawMessage(d))
		}
	}
	return []string{d}
}

// Fulfill places one order:
// GET {base}/api/PlaceOrder/{upstreamID}/data?amount=N&player_Id=P&referenceId=U.
func (t *Torosfon) Fulfill(ctx context.Context, in FulfillInput) (Result, error) {
	if in.UpstreamID == "" {
		// Nothing was sent: a mapping problem, a definite refusal.
		return Result{}, refusedf("torosfon %s: product %s has no upstream product id", t.name, in.ProductID)
	}
	player := in.PlayerID
	if player == "" {
		player = firstFieldValue(in.Fields)
	}
	q := url.Values{}
	q.Set("amount", strconv.Itoa(in.Qty))
	q.Set("player_Id", player)
	q.Set("referenceId", in.OrderUUID)
	endpoint := fmt.Sprintf("%s/api/PlaceOrder/%s/data?%s", t.baseURL, url.PathEscape(in.UpstreamID), q.Encode())

	// Anything we cannot read is ambiguous: Torosfon may have created the
	// order, and a retry would create a second one (no upstream dedupe).
	ambiguous := func(why string) (Result, error) {
		ref := searchRef("torosfon", in.OrderUUID)
		return Result{Reference: ref}, fmt.Errorf("torosfon %s: %s, order may exist upstream (%s): %w", t.name, why, ref, ErrPending)
	}

	status, body, err := t.do(ctx, endpoint)
	if err != nil {
		return ambiguous("request failed")
	}
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		// Rejected before any order logic ran.
		return Result{}, refusedf("torosfon %s: auth refused (status %d)", t.name, status)
	case status == http.StatusBadRequest && isValidationProblem(body):
		// ASP.NET model binding refused the request: no order was created.
		return Result{}, refusedf("torosfon %s: request refused as invalid (status 400)", t.name)
	case status == http.StatusTooManyRequests:
		return Result{}, refusedf("torosfon %s: throttled (status 429)", t.name)
	}

	var env torosfonEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return ambiguous(fmt.Sprintf("status %d, undecodable body", status))
	}
	var ord torosfonOrder
	_ = json.Unmarshal(env.Data, &ord) // data may be "" / null on failure
	msg := firstNonEmpty(env.Message, ord.Desc)

	ref := ord.id()
	if ref == "" {
		if !env.Success && status >= 200 && status < 300 {
			// An explicit failure with no order: a validation refusal.
			return Result{}, refusedf("torosfon %s: refused: %s", t.name, msg)
		}
		return ambiguous(fmt.Sprintf("status %d, reply without an order id", status))
	}

	res := Result{Reference: ref}
	switch torosfonState(ord.Status) {
	case "accept":
		res.Codes = torosfonCodes(ord.Desc)
		return res, nil
	case "reject":
		// Torosfon created and definitively cancelled THIS order (e.g. the
		// player id doesn't exist): order-specific — the engine compensates.
		return Result{}, fmt.Errorf("torosfon %s: order %s rejected upstream: %s", t.name, ref, msg)
	default:
		// Pending, or an unknown status on an order that exists: poll it by
		// its id. Never ErrUnavailable here — that would re-send it.
		return res, fmt.Errorf("torosfon %s: order %s %s: %w", t.name, ref, firstNonEmpty(ord.Status, "status unknown"), ErrPending)
	}
}

// CheckStatus reconciles one parked order: GET {base}/api/CheckOrder?order_id=ref.
// A "search:" reference cannot be looked up (Torosfon has no lookup by
// referenceId), so it waits for a human. Any error, including "order is not
// found", stays ErrUnavailable — never "reject", which would refund an order
// Torosfon may still deliver.
func (t *Torosfon) CheckStatus(ctx context.Context, ref string) (Status, error) {
	if strings.HasPrefix(ref, "search:") {
		return Status{State: "wait"}, nil
	}
	if _, err := strconv.ParseInt(ref, 10, 64); err != nil {
		return Status{}, fmt.Errorf("torosfon %s: reference %q is not a torosfon order id: %w", t.name, ref, ErrUnavailable)
	}
	status, body, err := t.do(ctx, t.baseURL+"/api/CheckOrder?order_id="+url.QueryEscape(ref))
	if err != nil {
		return Status{}, fmt.Errorf("torosfon %s: check failed: %w", t.name, ErrUnavailable)
	}
	var env torosfonEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return Status{}, fmt.Errorf("torosfon %s: check status %d, undecodable body: %w", t.name, status, ErrUnavailable)
	}
	var ord torosfonOrder
	if err := json.Unmarshal(env.Data, &ord); err != nil || ord.id() != ref {
		return Status{}, fmt.Errorf("torosfon %s: check: order %s not found: %s: %w", t.name, ref, env.Message, ErrUnavailable)
	}
	switch st := torosfonState(ord.Status); st {
	case "accept":
		return Status{State: "accept", Codes: torosfonCodes(ord.Desc)}, nil
	case "reject":
		return Status{State: "reject"}, nil
	default:
		return Status{State: "wait"}, nil
	}
}

// Verify is not supported — game-ID verification stays on platform/idcheck.
func (t *Torosfon) Verify(_ context.Context, _ FulfillInput) (AccountInfo, error) {
	return AccountInfo{}, ErrNotImplemented
}

// Profile probes the account balance for the admin /suppliers page:
// GET {base}/api/GetMyInfo. The balance can be negative (a credit account).
func (t *Torosfon) Profile(ctx context.Context) (Account, error) {
	env, err := t.probe(ctx, "/api/GetMyInfo")
	if err != nil {
		return Account{}, err
	}
	var info struct {
		Balance  float64 `json:"user_balance"`
		Currency string  `json:"user_currency"`
		Email    string  `json:"user_email"`
	}
	if err := json.Unmarshal(env.Data, &info); err != nil {
		return Account{}, fmt.Errorf("torosfon %s: undecodable account: %w", t.name, ErrUnavailable)
	}
	currency := t.currency
	if c := strings.TrimSpace(info.Currency); c != "" && c != "$" {
		currency = strings.ToUpper(c)
	}
	return Account{Balance: info.Balance, Currency: currency, Email: info.Email}, nil
}

// torosfonCatalogProduct is one row of GET /api/AllProducts. product_price is
// the cost of ONE unit of amount (per coin for open-amount products; per pack
// for fixed packs, whose min_qty = max_qty = 1).
type torosfonCatalogProduct struct {
	ID             flexString `json:"product_id"`
	Name           string     `json:"product_name"`
	NameTranslated string     `json:"product_name_translated"`
	Price          float64    `json:"product_price"`
	MinQty         *int       `json:"min_qty"`
	MaxQty         *int       `json:"max_qty"`
	Active         bool       `json:"is_active"`
}

// ListProducts returns the full catalog for the admin browse/import UI and the
// sync job. A failure reply ("no Products are found") is an error, never an
// empty catalog: sync would otherwise read every link as withdrawn.
func (t *Torosfon) ListProducts(ctx context.Context) ([]CatalogProduct, error) {
	env, err := t.probe(ctx, "/api/AllProducts")
	if err != nil {
		return nil, err
	}
	var rows []torosfonCatalogProduct
	if err := json.Unmarshal(env.Data, &rows); err != nil {
		return nil, fmt.Errorf("torosfon %s: undecodable products: %w", t.name, ErrUnavailable)
	}
	out := make([]CatalogProduct, 0, len(rows))
	for _, r := range rows {
		id := strings.TrimSpace(string(r.ID))
		if id == "" {
			continue
		}
		out = append(out, CatalogProduct{
			UpstreamID: id,
			Name:       firstNonEmpty(strings.TrimSpace(r.NameTranslated), strings.TrimSpace(r.Name)),
			// Torosfon has a single price: it is both our cost and the
			// import base (the browse UI and import read BasePrice).
			Price:     r.Price,
			BasePrice: r.Price,
			Currency:  t.currency,
			Available: r.Active,
			// One input: the player / account id (sent as player_Id).
			Params: []string{"Player ID"},
			QtyMin: r.MinQty,
			QtyMax: r.MaxQty,
		})
	}
	return out, nil
}

// probe GETs a Cataloger endpoint and returns its successful envelope, mapping
// failures onto the health-probe vocabulary.
func (t *Torosfon) probe(ctx context.Context, path string) (torosfonEnvelope, error) {
	status, body, err := t.do(ctx, t.baseURL+path)
	if err != nil {
		return torosfonEnvelope{}, fmt.Errorf("torosfon %s: request failed: %w", t.name, ErrUnavailable)
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return torosfonEnvelope{}, fmt.Errorf("torosfon %s: auth error (status %d): %w", t.name, status, ErrProbeAuth)
	}
	var env torosfonEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return torosfonEnvelope{}, fmt.Errorf("torosfon %s: status %d, undecodable body: %w", t.name, status, ErrUnavailable)
	}
	if !env.Success {
		return torosfonEnvelope{}, fmt.Errorf("torosfon %s: %s: %w", t.name, firstNonEmpty(env.Message, "request failed"), ErrUnavailable)
	}
	return env, nil
}

// do performs an authenticated GET and returns the status and raw body. Only a
// transport/read failure is an error; the caller classifies the reply.
func (t *Torosfon) do(ctx context.Context, endpoint string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("api-token", t.token)
	req.Header.Set("Accept", "application/json")
	resp, err := t.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, body, nil
}

// isValidationProblem reports an ASP.NET model-validation reply
// ({"title":"One or more validation errors occurred.","status":400,"errors":{…}}).
func isValidationProblem(body []byte) bool {
	var p struct {
		Status int             `json:"status"`
		Errors json.RawMessage `json:"errors"`
	}
	return json.Unmarshal(body, &p) == nil && p.Status == http.StatusBadRequest && len(p.Errors) > 0
}

// firstFieldValue returns a non-empty field value, preferring common player-id
// keys, for an order line whose PlayerID snapshot is empty.
func firstFieldValue(fields map[string]string) string {
	for _, k := range []string{"player_id", "playerId", "player", "id"} {
		if v := strings.TrimSpace(fields[k]); v != "" {
			return v
		}
	}
	for _, v := range fields {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
