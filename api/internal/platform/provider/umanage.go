package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Umanage is the adapter for the U-Manage Lebanese telecom reseller
// (DESIGN-SUPPLIERS.md Phase 4): Alfa/Touch data bundles, SMS credit/gifts, and
// telecom vouchers/direct-recharge. One provider serves all three families,
// discriminated by an UpstreamProductID prefix — "bundle:", "sms:", "telecom:"
// — so a product points at umanage via FulfillmentProvider and carries the
// prefixed upstream id. Auth is X-API-Key + X-API-Secret; every path is scoped
// to a store id (resolved once at boot via GET /stores when configured as 0).
// Prices are LBP; SalehCard does not FX-convert.
//
// Idempotency — the critical difference from the panels. umanage has NO
// upstream idempotency key (the panels dedupe on order_uuid; umanage's
// app_user_reference is documented as "for tracking" only). A blind re-dispatch
// after a transport timeout could therefore double-charge. So on an AMBIGUOUS
// failure (no HTTP response / undecodable body), Fulfill returns ErrPending with
// a "search:{family}:{orderUUID}" reference instead of a hard error, and
// CheckStatus resolves it by matching app_user_reference (= our order id) in the
// order list. The settler polls that reference like any other pending order —
// it never blindly re-dispatches an ambiguous umanage order.
type Umanage struct {
	id        int
	name      string
	currency  string
	baseURL   string
	apiKey    string
	apiSecret string
	http      *http.Client

	mu       sync.Mutex
	storeID  int  // 0 until resolved
	storeSet bool // true once storeID is authoritative
}

// UmanageConfig configures the umanage adapter.
type UmanageConfig struct {
	ID        int
	Name      string // slug ("umanage")
	Currency  string // "LBP"
	BaseURL   string // https://api.umanageapp.uk/api/v1/external
	APIKey    string
	APISecret string
	StoreID   int // 0 → resolve at boot/first-use via GET /stores
}

// NewUmanage builds the umanage adapter, erroring on missing config (the caller
// skips it so its id falls back to the parking stub).
func NewUmanage(cfg UmanageConfig) (*Umanage, error) {
	if cfg.ID <= 0 || cfg.Name == "" || cfg.BaseURL == "" || cfg.APIKey == "" || cfg.APISecret == "" {
		return nil, fmt.Errorf("umanage: id, name, base url, api key, and api secret are required")
	}
	currency := cfg.Currency
	if currency == "" {
		currency = "LBP"
	}
	return &Umanage{
		id:        cfg.ID,
		name:      cfg.Name,
		currency:  currency,
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:    cfg.APIKey,
		apiSecret: cfg.APISecret,
		http:      &http.Client{Timeout: 25 * time.Second},
		storeID:   cfg.StoreID,
		storeSet:  cfg.StoreID > 0,
	}, nil
}

// ID is the numeric provider id this adapter registers under.
func (u *Umanage) ID() int { return u.id }

// Verify is not supported by umanage.
func (u *Umanage) Verify(_ context.Context, _ FulfillInput) (AccountInfo, error) {
	return AccountInfo{}, ErrNotImplemented
}

// umStore is one store row from GET /stores.
type umStore struct {
	StoreID    int     `json:"store_id"`
	StoreName  string  `json:"store_name"`
	BalanceLBP float64 `json:"balance_lbp"`
}

// umError is the standard error envelope ({success:false, error:{code,message}}).
type umError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// umOrder is the (superset) order object returned by the create/detail
// endpoints across families. Fields absent for a family stay zero.
type umOrder struct {
	OrderID          json.Number `json:"order_id"`
	Status           string      `json:"status"`
	AppUserReference string      `json:"app_user_reference"`
	Refunded         bool        `json:"refunded"`
	Error            string      `json:"error"`
	AlfaResponse     string      `json:"alfa_response"`
	Voucher          *struct {
		Code       string `json:"code"`
		Serial     string `json:"serial"`
		ExpiryDate string `json:"expiry_date"`
	} `json:"voucher"`
	SMSTransaction *struct {
		ConfirmationMessage string `json:"confirmation_message"`
		ErrorMessage        string `json:"error_message"`
	} `json:"sms_transaction"`
}

// Fulfill places one umanage order for the appropriate family.
func (u *Umanage) Fulfill(ctx context.Context, in FulfillInput) (Result, error) {
	if in.UpstreamID == "" {
		return Result{}, fmt.Errorf("umanage: product %s has no upstream product id: %w", in.ProductID, ErrUnavailable)
	}
	family, rest, ok := strings.Cut(in.UpstreamID, ":")
	if !ok || rest == "" {
		return Result{}, fmt.Errorf("umanage: malformed upstream id %q (want family:id): %w", in.UpstreamID, ErrUnavailable)
	}
	store, err := u.resolveStore(ctx)
	if err != nil {
		return Result{}, err
	}

	var path string
	var body map[string]any
	switch family {
	case "bundle":
		path = fmt.Sprintf("/stores/%d/orders", store)
		body = map[string]any{
			"bundle_id":          numOrString(rest),
			"secondary_number":   umPhone(in),
			"app_user_reference": in.OrderUUID,
		}
	case "sms":
		// rest = "{type}:{code}[:{carrier}]" e.g. "credit:5.5:alfa" / "gift:MI7:alfa".
		typ, code, carrier := parseSMSUpstream(rest)
		path = fmt.Sprintf("/stores/%d/sms-orders", store)
		body = map[string]any{
			"product_type":       typ,
			"product_code":       code,
			"recipient_phone":    umPhone(in),
			"carrier_type":       carrier,
			"app_user_reference": in.OrderUUID,
		}
	case "telecom":
		path = fmt.Sprintf("/stores/%d/telecom-orders", store)
		body = map[string]any{
			"product_id":         numOrString(rest),
			"app_user_reference": in.OrderUUID,
		}
		mode := in.Fields["purchase_mode"]
		if mode == "" {
			mode = "voucher"
		}
		body["purchase_mode"] = mode
		if ph := umPhone(in); ph != "" {
			body["recipient_phone"] = ph
		}
		if plan := in.Fields["plan_id"]; plan != "" {
			body["plan_id"] = plan
		}
	default:
		return Result{}, fmt.Errorf("umanage: unknown product family %q: %w", family, ErrUnavailable)
	}

	raw, status, err := u.post(ctx, path, body)
	if err != nil {
		// Transport/read failure: AMBIGUOUS — the order may or may not have
		// landed. Never blindly compensate or re-dispatch (no idempotency key);
		// park with a search reference the settler reconciles by app_user_reference.
		return Result{Reference: searchRef(family, in.OrderUUID)}, fmt.Errorf("umanage: %s ambiguous: %w", family, ErrPending)
	}

	var env struct {
		Success bool     `json:"success"`
		Order   umOrder  `json:"order"`
		Error   *umError `json:"error"`
	}
	if json.Unmarshal(raw, &env) != nil {
		// Undecodable body on a real HTTP response: still ambiguous — park to search.
		return Result{Reference: searchRef(family, in.OrderUUID)}, fmt.Errorf("umanage: %s undecodable (status %d): %w", family, status, ErrPending)
	}

	// success:false — either a pre-charge rejection (no order) or an
	// auto-refunded failure (order present, refunded).
	if !env.Success {
		if env.Order.Refunded || env.Order.Status == "failed" {
			// umanage already refunded its wallet — this order definitively failed;
			// compensate the customer (plain error).
			return Result{}, fmt.Errorf("umanage: %s failed upstream (refunded): %s", family, u.orderReason(env.Order, env.Error))
		}
		return Result{}, u.classifyOrderError(family, env.Error)
	}

	// success:true — inspect the order status.
	ref := familyRef(family, env.Order.OrderID.String())
	switch env.Order.Status {
	case "completed":
		return Result{Reference: ref, Codes: voucherCodes(env.Order), Meta: orderMeta(env.Order)}, nil
	case "processing", "pending":
		return Result{Reference: ref}, fmt.Errorf("umanage: %s pending: %w", family, ErrPending)
	case "failed":
		return Result{}, fmt.Errorf("umanage: %s failed upstream: %s", family, u.orderReason(env.Order, env.Error))
	default:
		// Accepted but unknown state — park to poll.
		return Result{Reference: ref}, fmt.Errorf("umanage: %s unknown status %q: %w", family, env.Order.Status, ErrPending)
	}
}

// CheckStatus reconciles a pending umanage order. Two reference shapes:
//   - "search:{family}:{orderUUID}" — the create call was ambiguous; find the
//     order by app_user_reference in the family's order list.
//   - "{family}:{orderId}"          — a known upstream order id; GET its detail.
func (u *Umanage) CheckStatus(ctx context.Context, ref string) (Status, error) {
	store, err := u.resolveStore(ctx)
	if err != nil {
		return Status{}, err
	}
	if rest, ok := strings.CutPrefix(ref, "search:"); ok {
		family, uuid, _ := strings.Cut(rest, ":")
		return u.searchStatus(ctx, store, family, uuid)
	}
	family, id, ok := strings.Cut(ref, ":")
	if !ok || id == "" {
		return Status{}, fmt.Errorf("umanage: malformed check ref %q: %w", ref, ErrUnavailable)
	}
	raw, _, err := u.get(ctx, fmt.Sprintf("/stores/%d/%s/%s", store, familyPath(family), id))
	if err != nil {
		return Status{}, err
	}
	var env struct {
		Success bool    `json:"success"`
		Order   umOrder `json:"order"`
	}
	if json.Unmarshal(raw, &env) != nil {
		return Status{}, fmt.Errorf("umanage: undecodable %s detail: %w", family, ErrUnavailable)
	}
	return statusFromOrder(env.Order), nil
}

// searchStatus resolves an ambiguous create by scanning the family's recent
// orders for our app_user_reference. Not found (yet) → "wait" so the settler
// keeps polling and eventually flags it stuck for manual review — never a blind
// refund or re-dispatch.
func (u *Umanage) searchStatus(ctx context.Context, store int, family, uuid string) (Status, error) {
	const pageSize = 100
	for offset := 0; offset <= 300; offset += pageSize {
		raw, _, err := u.get(ctx, fmt.Sprintf("/stores/%d/%s?limit=%d&offset=%d", store, familyPath(family), pageSize, offset))
		if err != nil {
			return Status{}, err
		}
		var env struct {
			Success bool      `json:"success"`
			Orders  []umOrder `json:"orders"`
		}
		if json.Unmarshal(raw, &env) != nil {
			return Status{}, fmt.Errorf("umanage: undecodable %s list: %w", family, ErrUnavailable)
		}
		for _, o := range env.Orders {
			if o.AppUserReference == uuid {
				return statusFromOrder(o), nil
			}
		}
		if len(env.Orders) < pageSize {
			break // last page reached, not found
		}
	}
	return Status{State: "wait"}, nil
}

// Profile probes the store balance (admin /suppliers page). Doubles as store-id
// resolution + credential validation.
func (u *Umanage) Profile(ctx context.Context) (Account, error) {
	stores, err := u.fetchStores(ctx)
	if err != nil {
		return Account{}, err
	}
	st := u.pickStore(stores)
	if st == nil {
		return Account{}, fmt.Errorf("umanage: no linked store: %w", ErrProbeAuth)
	}
	return Account{Balance: st.BalanceLBP, Currency: u.currency}, nil
}

// ListProducts merges the three umanage families into one normalized catalog
// for the admin browse/import UI. A family that a store cannot fulfill (e.g. no
// SMS service) is skipped, not fatal.
func (u *Umanage) ListProducts(ctx context.Context) ([]CatalogProduct, error) {
	store, err := u.resolveStore(ctx)
	if err != nil {
		return nil, err
	}
	var out []CatalogProduct
	out = append(out, u.listBundles(ctx, store)...)
	out = append(out, u.listSMS(ctx, store)...)
	out = append(out, u.listTelecom(ctx, store)...)
	return out, nil
}

func (u *Umanage) listBundles(ctx context.Context, store int) []CatalogProduct {
	raw, _, err := u.get(ctx, fmt.Sprintf("/stores/%d/bundles", store))
	if err != nil {
		return nil
	}
	var env struct {
		Bundles []struct {
			BundleID   json.Number `json:"bundle_id"`
			BundleSize json.Number `json:"bundle_size"`
			BundleType string      `json:"bundle_type"`
			BundleDays *int        `json:"bundle_days"`
			PriceLBP   float64     `json:"price_lbp"`
			Available  json.Number `json:"is_available"`
		} `json:"bundles"`
	}
	if json.Unmarshal(raw, &env) != nil {
		return nil
	}
	out := make([]CatalogProduct, 0, len(env.Bundles))
	for _, b := range env.Bundles {
		name := fmt.Sprintf("Alfa Bundle %s GB", b.BundleSize.String())
		if b.BundleDays != nil {
			name = fmt.Sprintf("%s (%dd)", name, *b.BundleDays)
		}
		out = append(out, CatalogProduct{
			UpstreamID:  "bundle:" + b.BundleID.String(),
			Name:        name,
			Category:    "Alfa Bundles",
			Price:       b.PriceLBP,
			BasePrice:   b.PriceLBP,
			Currency:    u.currency,
			Available:   b.Available.String() == "1" || strings.EqualFold(b.Available.String(), "true"),
			ProductType: "bundle",
			Params:      []string{"secondary_number"},
		})
	}
	return out
}

func (u *Umanage) listSMS(ctx context.Context, store int) []CatalogProduct {
	raw, _, err := u.get(ctx, fmt.Sprintf("/stores/%d/sms-products", store))
	if err != nil {
		return nil
	}
	var env struct {
		HasSMS  bool `json:"has_sms_service"`
		Credits []struct {
			Carrier string `json:"carrier_type"`
			Pricing []struct {
				AmountUSD json.Number `json:"amount_usd"`
				PriceLBP  float64     `json:"price_lbp"`
			} `json:"pricing"`
		} `json:"credits"`
		Gifts []struct {
			ProductCode string  `json:"product_code"`
			Name        string  `json:"name"`
			Carrier     string  `json:"carrier_type"`
			PriceLBP    float64 `json:"price_lbp"`
		} `json:"gifts"`
	}
	if json.Unmarshal(raw, &env) != nil || !env.HasSMS {
		return nil
	}
	var out []CatalogProduct
	for _, c := range env.Credits {
		for _, p := range c.Pricing {
			out = append(out, CatalogProduct{
				UpstreamID:  fmt.Sprintf("sms:credit:%s:%s", p.AmountUSD.String(), c.Carrier),
				Name:        fmt.Sprintf("%s Credit $%s", titleWord(c.Carrier), p.AmountUSD.String()),
				Category:    titleWord(c.Carrier) + " Credit",
				Price:       p.PriceLBP,
				BasePrice:   p.PriceLBP,
				Currency:    u.currency,
				Available:   true,
				ProductType: "credit",
				Params:      []string{"recipient_phone"},
			})
		}
	}
	for _, g := range env.Gifts {
		out = append(out, CatalogProduct{
			UpstreamID:  fmt.Sprintf("sms:gift:%s:%s", g.ProductCode, g.Carrier),
			Name:        g.Name,
			Category:    titleWord(g.Carrier) + " Gifts",
			Price:       g.PriceLBP,
			BasePrice:   g.PriceLBP,
			Currency:    u.currency,
			Available:   true,
			ProductType: "gift",
			Params:      []string{"recipient_phone"},
		})
	}
	return out
}

func (u *Umanage) listTelecom(ctx context.Context, store int) []CatalogProduct {
	raw, _, err := u.get(ctx, fmt.Sprintf("/stores/%d/telecom-products", store))
	if err != nil {
		return nil
	}
	var env struct {
		Products []struct {
			ProductID   json.Number `json:"product_id"`
			Name        string      `json:"name"`
			Brand       string      `json:"brand"`
			ProductType string      `json:"product_type"`
			Category    string      `json:"category"`
			PriceLBP    float64     `json:"price_lbp"`
		} `json:"products"`
	}
	if json.Unmarshal(raw, &env) != nil {
		return nil
	}
	out := make([]CatalogProduct, 0, len(env.Products))
	for _, p := range env.Products {
		cat := strings.TrimSpace(p.Brand + " " + p.Category)
		out = append(out, CatalogProduct{
			UpstreamID:  "telecom:" + p.ProductID.String(),
			Name:        p.Name,
			Category:    cat,
			Price:       p.PriceLBP,
			BasePrice:   p.PriceLBP,
			Currency:    u.currency,
			Available:   true,
			ProductType: p.ProductType,
		})
	}
	return out
}

// --- store resolution ---

func (u *Umanage) resolveStore(ctx context.Context) (int, error) {
	u.mu.Lock()
	if u.storeSet {
		id := u.storeID
		u.mu.Unlock()
		return id, nil
	}
	u.mu.Unlock()

	stores, err := u.fetchStores(ctx)
	if err != nil {
		return 0, err
	}
	st := u.pickStore(stores)
	if st == nil {
		return 0, fmt.Errorf("umanage: no linked store: %w", ErrProbeAuth)
	}
	u.mu.Lock()
	u.storeID, u.storeSet = st.StoreID, true
	u.mu.Unlock()
	return st.StoreID, nil
}

func (u *Umanage) fetchStores(ctx context.Context) ([]umStore, error) {
	raw, _, err := u.get(ctx, "/stores")
	if err != nil {
		return nil, err
	}
	var env struct {
		Success bool      `json:"success"`
		Stores  []umStore `json:"stores"`
		Error   *umError  `json:"error"`
	}
	if json.Unmarshal(raw, &env) != nil {
		return nil, fmt.Errorf("umanage: undecodable stores: %w", ErrUnavailable)
	}
	if !env.Success && env.Error != nil {
		return nil, u.classifyProbe(env.Error)
	}
	return env.Stores, nil
}

// pickStore returns the configured store (when a StoreID was set) or the first
// linked store.
func (u *Umanage) pickStore(stores []umStore) *umStore {
	u.mu.Lock()
	want := u.storeID
	u.mu.Unlock()
	for i := range stores {
		if want > 0 && stores[i].StoreID == want {
			return &stores[i]
		}
	}
	if want == 0 && len(stores) > 0 {
		return &stores[0]
	}
	return nil
}

// --- HTTP + classification helpers ---

func (u *Umanage) get(ctx context.Context, path string) ([]byte, int, error) {
	return u.do(ctx, http.MethodGet, path, nil)
}

func (u *Umanage) post(ctx context.Context, path string, body map[string]any) ([]byte, int, error) {
	b, _ := json.Marshal(body)
	return u.do(ctx, http.MethodPost, path, b)
}

func (u *Umanage) do(ctx context.Context, method, path string, body []byte) ([]byte, int, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.baseURL+path, rdr)
	if err != nil {
		return nil, 0, fmt.Errorf("umanage: build request: %w", ErrUnavailable)
	}
	req.Header.Set("X-API-Key", u.apiKey)
	req.Header.Set("X-API-Secret", u.apiSecret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := u.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("umanage: request failed: %w", ErrUnavailable)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("umanage: read body: %w", ErrUnavailable)
	}
	return raw, resp.StatusCode, nil
}

// classifyProbe maps a store/list error to the health-probe vocabulary.
func (u *Umanage) classifyProbe(e *umError) error {
	if e == nil {
		return fmt.Errorf("umanage: probe error: %w", ErrUnavailable)
	}
	switch e.Code {
	case "API_KEY_MISSING", "INVALID_API_KEY", "AUTH_REQUIRED", "AUTH_ERROR", "STORE_ACCESS_DENIED", "INVALID_STORE_ID":
		return fmt.Errorf("umanage: %s: %w", e.Code, ErrProbeAuth)
	default:
		return fmt.Errorf("umanage: %s: %s: %w", e.Code, e.Message, ErrUnavailable)
	}
}

// classifyOrderError maps a create-order rejection (no order created) onto the
// fulfillment contract: environmental → ErrUnavailable (park + safe re-dispatch,
// since no order was created); order-specific validation → plain error
// (compensate). Auth/store/access errors park (config problem, never the
// customer's fault).
func (u *Umanage) classifyOrderError(family string, e *umError) error {
	if e == nil {
		return fmt.Errorf("umanage: %s rejected (no detail): %w", family, ErrUnavailable)
	}
	switch e.Code {
	case "INSUFFICIENT_BALANCE", "STORE_INSUFFICIENT_BALANCE", "SMS_SERVICE_UNAVAILABLE",
		"RATE_LIMIT_EXCEEDED", "INTERNAL_ERROR", "ORDER_FAILED", "SMS_ORDER_FAILED",
		"VALIDATION_ERROR", "API_KEY_MISSING", "INVALID_API_KEY", "AUTH_REQUIRED",
		"AUTH_ERROR", "STORE_ACCESS_DENIED", "INVALID_STORE_ID":
		// Environmental / config — no order created, park and retry cleanly.
		return fmt.Errorf("umanage: %s environmental (%s): %s: %w", family, e.Code, e.Message, ErrUnavailable)
	default:
		// BUNDLE_OUT_OF_STOCK, BUNDLE_UNAVAILABLE, INVALID_PHONE, INVALID_CARRIER,
		// INVALID_AMOUNT, INVALID_PRODUCT, PRODUCT_NOT_AVAILABLE, PRODUCT_NOT_FOUND,
		// MISSING_FIELDS — order-specific, no charge; compensate.
		return fmt.Errorf("umanage: %s rejected (%s): %s", family, e.Code, e.Message)
	}
}

func (u *Umanage) orderReason(o umOrder, e *umError) string {
	switch {
	case o.Error != "":
		return o.Error
	case o.SMSTransaction != nil && o.SMSTransaction.ErrorMessage != "":
		return o.SMSTransaction.ErrorMessage
	case o.AlfaResponse != "":
		return o.AlfaResponse
	case e != nil:
		return e.Message
	default:
		return "unknown"
	}
}

// --- pure helpers ---

// statusFromOrder maps a umanage order status to the settler's vocabulary.
func statusFromOrder(o umOrder) Status {
	switch o.Status {
	case "completed":
		return Status{State: "accept", Codes: voucherCodes(o)}
	case "failed":
		return Status{State: "reject"}
	default: // processing / pending
		return Status{State: "wait"}
	}
}

func voucherCodes(o umOrder) []string {
	if o.Voucher != nil && strings.TrimSpace(o.Voucher.Code) != "" {
		return []string{o.Voucher.Code}
	}
	return nil
}

func orderMeta(o umOrder) map[string]string {
	m := map[string]string{}
	if o.Voucher != nil {
		if o.Voucher.Serial != "" {
			m["serial"] = o.Voucher.Serial
		}
		if o.Voucher.ExpiryDate != "" {
			m["expiry"] = o.Voucher.ExpiryDate
		}
	}
	if o.AlfaResponse != "" {
		m["confirmation"] = o.AlfaResponse
	}
	if o.SMSTransaction != nil && o.SMSTransaction.ConfirmationMessage != "" {
		m["confirmation"] = o.SMSTransaction.ConfirmationMessage
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// familyRef / searchRef / familyPath encode the CheckStatus reference vocabulary.
func familyRef(family, id string) string   { return family + ":" + id }
func searchRef(family, uuid string) string { return "search:" + family + ":" + uuid }

// familyPath maps a family to its umanage order-endpoint segment.
func familyPath(family string) string {
	switch family {
	case "sms":
		return "sms-orders"
	case "telecom":
		return "telecom-orders"
	default: // bundle
		return "orders"
	}
}

// parseSMSUpstream splits "type:code[:carrier]" (carrier defaults to alfa).
func parseSMSUpstream(rest string) (typ, code, carrier string) {
	parts := strings.SplitN(rest, ":", 3)
	typ = parts[0]
	if len(parts) > 1 {
		code = parts[1]
	}
	carrier = "alfa"
	if len(parts) > 2 && parts[2] != "" {
		carrier = parts[2]
	}
	return typ, code, carrier
}

// umPhone extracts the recipient's 8-digit local number from the captured input
// fields (or PlayerID), stripping a country prefix. umanage wants 8 digits.
func umPhone(in FulfillInput) string {
	raw := firstNonEmpty(
		in.Fields["secondary_number"], in.Fields["recipient_phone"],
		in.Fields["phone"], in.Fields["playerId"], in.PlayerID,
	)
	var digits strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	d := digits.String()
	d = strings.TrimPrefix(d, "961")
	if len(d) > 8 {
		d = d[len(d)-8:]
	}
	return d
}

// titleWord upper-cases the first letter of a single lowercase word ("alfa" →
// "Alfa"), avoiding the deprecated strings.Title.
func titleWord(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// numOrString returns a JSON-friendly value: a number when rest parses as an
// int, else the raw string (umanage accepts either for product/bundle ids).
func numOrString(rest string) any {
	if n, err := strconv.Atoi(rest); err == nil {
		return n
	}
	return rest
}
