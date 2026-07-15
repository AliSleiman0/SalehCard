package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/modules/bridge"
	"github.com/AliSleiman0/salehcard/api/internal/modules/code"
	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/modules/offer"
	"github.com/AliSleiman0/salehcard/api/internal/modules/payment"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/promo"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/internal/platform/payments"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// badRequest builds a 400-class AppError with a machine code.
func badRequest(msg string) error {
	return &apperrors.AppError{Code: "BAD_REQUEST", Message: msg, Err: apperrors.ErrBadRequest}
}

// unavailablePaymentMethod is the shared rejection for a payment method whose
// backing gateway/provider is not configured.
func unavailablePaymentMethod() error {
	return &apperrors.AppError{
		Code:    "PAYMENT_METHOD_UNAVAILABLE",
		Message: "this payment method is not available — top up your wallet to purchase",
		Err:     apperrors.ErrBadRequest,
	}
}

// ErrKYCRequired rejects checkout for users without an approved KYC
// submission; the handler maps it to 403 KYC_REQUIRED so clients can route
// the customer into the verification flow.
var ErrKYCRequired = errors.New("identity verification required")

// kycChecker is the slice of the KYC module the order service needs to gate
// checkout on identity verification (kept minimal for testability).
type kycChecker interface {
	IsApproved(ctx context.Context, userID bson.ObjectID) (bool, error)
}

// resellerPricing resolves the buyer's reseller pricing inputs: their tier
// margin percent (0 = no discount) and any per-variant custom price overrides.
// Kept as one narrow port so the order module stays decoupled from the reseller
// module's concrete repository.
type resellerPricing interface {
	MarginForUser(ctx context.Context, userID bson.ObjectID) (float64, error)
	PricesForUser(ctx context.Context, userID bson.ObjectID) (map[string]float64, error)
}

// loyaltyAwarder is the slice of the loyalty module the order service needs to
// mint points when an order completes. It is best-effort (returns nothing; logs
// and swallows its own failures) so it can never fail a completed order — kept
// narrow for testability.
type loyaltyAwarder interface {
	Award(ctx context.Context, userID bson.ObjectID, orderTotal float64)
}

// usdtIntents is the slice of the payment module the order service needs for
// on-chain USDT checkout: create a payment intent for a placed order, and let
// the handler re-read the live intent (deposit address/amount) to attach it to
// the order response. nil (or Enabled()==false) means USDT checkout is off, so
// the usdt method is rejected exactly like an unconfigured gateway. The reverse
// direction — the watcher fulfilling a paid order — is the payment.OrderSettler
// interface, which *OrderService implements (FulfillPaidOrder/FailUnpaidOrder).
type usdtIntents interface {
	Enabled() bool
	SupportsNetwork(network string) bool
	CreateOrderIntent(ctx context.Context, userID, orderID bson.ObjectID, amountUSD float64, network string) (*payment.Intent, error)
	GetActiveOrderIntent(ctx context.Context, orderID bson.ObjectID) (*payment.Intent, error)
}

// Service defines the business-logic operations for the order domain.
type Service interface {
	PlaceOrder(ctx context.Context, userID bson.ObjectID, isReseller bool, idempotencyKey string, input PlaceOrderInput) (*Order, error)
	GetOrder(ctx context.Context, userID, id bson.ObjectID) (*Order, error)
	ListOrders(ctx context.Context, userID bson.ObjectID) ([]*Order, error)
}

// OrderService is the concrete implementation of Service. It orchestrates
// pricing (from the product catalog), payment (wallet-only at launch), and
// fulfillment — dispatching each order down one of four paths keyed on the
// product's fulfillment mode (inventory / api / manual_operator / bridge_device).
type OrderService struct {
	repo      Repository
	products  product.Service
	codes     code.Service
	wallet    wallet.Service
	promo     promo.Service
	offers    offer.Service
	providers *provider.Registry
	payments  *payments.Registry
	usdt      usdtIntents
	kyc       kycChecker
	margins   resellerPricing
	ntf       notification.Notifier
	loyalty   loyaltyAwarder
	bridge    bridgeDispatcher
}

// NewOrderService constructs an OrderService wired to the catalog, code
// inventory, wallet, promo, offers, upstream-provider registry, USDT payment
// intents, KYC gate, reseller-margin lookup, and notifier it depends on. A nil
// margins port disables tier-margin pricing (resellers fall back to per-variant
// overrides); a nil usdt port disables on-chain USDT checkout.
func NewOrderService(repo Repository, products product.Service, codes code.Service, wlt wallet.Service, promos promo.Service, offers offer.Service, providers *provider.Registry, pay *payments.Registry, usdt usdtIntents, kycGate kycChecker, margins resellerPricing, ntf notification.Notifier, loyalty loyaltyAwarder, brdg bridgeDispatcher) *OrderService {
	return &OrderService{repo: repo, products: products, codes: codes, wallet: wlt, promo: promos, offers: offers, providers: providers, payments: pay, usdt: usdt, kyc: kycGate, margins: margins, ntf: ntf, loyalty: loyalty, bridge: brdg}
}

// Providers exposes the upstream-fulfillment registry so the admin supplier
// module can resolve each configured provider id to its Cataloger (balance +
// catalog probes) without rebuilding the adapters. Resolve returns a parking
// stub for unconfigured ids, which the Cataloger assertion then treats as
// "catalog unavailable".
func (s *OrderService) Providers() *provider.Registry { return s.providers }

// PlaceOrder validates and prices an order server-side, charges the chosen
// payment method, and fulfills code items by claiming inventory. It follows an
// order-first strategy: the order is persisted as pending, then charged and
// fulfilled, then flipped to completed/processing; any failure compensates
// (releasing claimed codes, refunding the wallet) and marks the order failed.
func (s *OrderService) PlaceOrder(ctx context.Context, userID bson.ObjectID, isReseller bool, idempotencyKey string, input PlaceOrderInput) (*Order, error) {
	// 0. KYC gate: every purchase requires an approved identity verification.
	//    Checked first — before pricing/inventory — so both the cart and the
	//    direct-checkout paths are covered by the one gate.
	if s.kyc != nil {
		approved, err := s.kyc.IsApproved(ctx, userID)
		if err != nil {
			return nil, err
		}
		if !approved {
			return nil, ErrKYCRequired
		}
	}

	// 1. Validate. Wallet is the only live payment method: card/usdt were
	//    mock-approved (delivering real inventory with no charge) and stay
	//    rejected until a real gateway is integrated.
	if len(input.Items) == 0 {
		return nil, badRequest("order must contain at least one item")
	}
	switch input.PaymentMethod {
	case PaymentMethodWallet:
	case PaymentMethodUSDT:
		// On-chain USDT is accepted only when the payment module is enabled.
		// It settles asynchronously — see the intent branch after the order is
		// persisted. An explicit network must be validated HERE, before the
		// order row exists, so a bad value never creates a failed order.
		if s.usdt == nil || !s.usdt.Enabled() {
			return nil, unavailablePaymentMethod()
		}
		if input.UsdtNetwork != "" && !s.usdt.SupportsNetwork(input.UsdtNetwork) {
			return nil, badRequest("unsupported USDT payment network")
		}
	case PaymentMethodCard:
		// Card is accepted only when a gateway provider is configured
		// (PAYMENT_PROVIDER); otherwise checkout stays wallet-only.
		if s.payments == nil || !s.payments.Enabled(string(input.PaymentMethod)) {
			return nil, unavailablePaymentMethod()
		}
	default:
		return nil, &apperrors.AppError{
			Code:    "PAYMENT_METHOD_UNAVAILABLE",
			Message: "unsupported payment method",
			Err:     apperrors.ErrBadRequest,
		}
	}
	currency := input.Currency
	if currency == "" {
		currency = "USD"
	}

	// 2. Idempotency pre-check: a prior order under this key is returned as-is.
	if idempotencyKey != "" {
		if existing, err := s.repo.FindByIdempotencyKey(ctx, userID, idempotencyKey); err == nil {
			return existing, nil
		} else if !errors.Is(err, apperrors.ErrNotFound) {
			return nil, err
		}
	}

	// 3. Re-price every line from the catalog (never trust client prices) and
	//    snapshot the fulfillment classification (type + execution mode).
	items := make([]OrderItem, 0, len(input.Items))
	var subtotal float64
	codeNeed := map[string]int{} // productID(hex) -> qty for inventory items

	// Preload the reseller's pricing inputs once (tier margin + per-variant custom
	// overrides) rather than per line — PlaceOrder re-prices every item. A lookup
	// error degrades to the remaining pricing layers (logged, never fails the order).
	var resellerMargin float64
	var resellerCustom map[string]float64
	if isReseller && s.margins != nil {
		if pct, err := s.margins.MarginForUser(ctx, userID); err != nil {
			slog.Warn("order: reseller margin lookup failed", "user", userID.Hex(), "error", err)
		} else {
			resellerMargin = pct
		}
		if m, err := s.margins.PricesForUser(ctx, userID); err != nil {
			slog.Warn("order: reseller custom-price lookup failed", "user", userID.Hex(), "error", err)
		} else {
			resellerCustom = m
		}
	}

	for _, in := range input.Items {
		if in.Qty <= 0 {
			return nil, badRequest("item quantity must be positive")
		}
		p, err := s.products.FindByID(ctx, in.ProductID)
		if err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				return nil, badRequest("unknown product: " + in.ProductID)
			}
			return nil, err
		}
		variant, ok := findVariant(p, in.VariantID)
		if !ok {
			return nil, badRequest("unknown variant for product " + in.ProductID)
		}
		price := variant.Price
		if isReseller {
			// Resellers pay the lowest of retail, the tier-margin price, the
			// per-variant global override, and any per-reseller custom price.
			// Offers don't stack on top of reseller pricing. The formula is
			// shared with the catalog's reseller enrichment so displayed price
			// always equals charged price.
			price = product.ResellerUnitPrice(variant, resellerMargin, resellerCustom)
		} else if s.offers != nil {
			// Honor a live sale-price offer on the retail price. Absence of an
			// offer (ErrNotFound) leaves the price unchanged.
			if off, oerr := s.offers.FindLiveByProduct(ctx, p.ID, time.Now().UTC()); oerr == nil {
				price = offer.OfferPriceFor(off, price)
			}
		}
		ft := string(p.FulfillmentType)
		// Resolve the execution mode, deriving a behavior-preserving default for
		// legacy products that predate the fulfillmentMode field.
		mode := p.FulfillmentMode
		if mode == "" {
			mode = product.DeriveMode(p.FulfillmentType)
		}
		if mode == product.FulfillmentModeInventory {
			codeNeed[p.ID.Hex()] += in.Qty
		}
		// Bridge (Lebanese mobile recharge) lines carry hard constraints the
		// device engine relies on: exactly one line, quantity one (a partial
		// multi-unit recharge has no clean meaning), a dialable mobile number, and
		// — for transfer_credit — a positive face value to transfer.
		// API-supplier lines are dispatched one order = one upstream call;
		// multi-item dispatch doesn't exist yet, so refuse mixed carts rather
		// than silently fulfilling only the first api line (the app's buy-now
		// flow is single-item anyway). Quantity stays free — panels take qty.
		if mode == product.FulfillmentModeAPI && len(input.Items) != 1 {
			return nil, badRequest("this product must be purchased on its own")
		}
		if mode == product.FulfillmentModeBridgeDevice {
			if len(input.Items) != 1 {
				return nil, badRequest("a mobile recharge must be purchased on its own")
			}
			if in.Qty != 1 {
				return nil, badRequest("a mobile recharge must have quantity 1")
			}
			if p.Bridge == nil || !p.Bridge.Valid() {
				return nil, badRequest("this product is not configured for mobile recharge")
			}
			if !validLebaneseMobile(normalizeLebanesePhone(bridgePhone(in))) {
				return nil, badRequest("a valid Lebanese mobile number is required")
			}
			if p.Bridge.Method == product.BridgeMethodTransferCredit && (variant.FaceValue == nil || *variant.FaceValue <= 0) {
				return nil, badRequest("this recharge denomination is missing its amount")
			}
		}
		if err := validateQuantityField(p, in.Qty); err != nil {
			return nil, err
		}
		if err := validateFieldInputs(p, in.Fields); err != nil {
			return nil, err
		}
		items = append(items, OrderItem{
			ProductID:           p.ID,
			VariantID:           variant.ID,
			Title:               p.Title,
			Denomination:        variant.Denomination,
			Category:            p.Category,
			Qty:                 in.Qty,
			Price:               price,
			FulfillmentType:     ft,
			FulfillmentMode:     string(mode),
			FulfillmentProvider: p.FulfillmentProvider,
			UpstreamProductID:   p.UpstreamProductID,
			PlayerID:            in.PlayerID,
			Recipient:           in.Recipient,
			Fields:              resolveOrderFields(p, in.Fields, in.Qty),
		})
		subtotal += price * float64(in.Qty)
	}
	// Apply a promo code when supplied: re-validated server-side against the
	// re-priced subtotal (never trust the client). An invalid/expired/below-min
	// code fails the order so the customer learns why; the use is recorded after
	// the charge succeeds (step 6).
	total := subtotal
	var discount float64
	promoCode := strings.ToUpper(strings.TrimSpace(input.PromoCode))
	if promoCode != "" {
		p, err := s.promo.Validate(ctx, promo.ValidateInput{Code: promoCode, OrderTotal: subtotal})
		if err != nil {
			return nil, err
		}
		discount = promo.DiscountFor(p, subtotal)
		total = subtotal - discount
		if total < 0 {
			total = 0
		}
	}

	// 4. Fast-fail stock pre-check for code items (best effort; the atomic claim
	//    below is the real guard against concurrent double-sell).
	for pid, need := range codeNeed {
		if avail, err := s.codes.CountAvailable(ctx, pid); err == nil && avail < need {
			return nil, code.ErrOutOfStock
		}
	}

	// 5. Persist the pending order.
	order := &Order{
		UserID:         userID,
		Items:          items,
		Subtotal:       subtotal,
		Discount:       discount,
		PromoCode:      promoCode,
		Total:          total,
		Currency:       currency,
		PaymentMethod:  input.PaymentMethod,
		Status:         OrderStatusPending,
		Fulfillment:    Fulfillment{StatusTimeline: []TimelineEvent{{Status: "created", At: time.Now().UTC()}}},
		IdempotencyKey: idempotencyKey,
	}
	if err := s.repo.Create(ctx, order); err != nil {
		if errors.Is(err, apperrors.ErrConflict) {
			// A concurrent request won the idempotency race; return the winner.
			if existing, ferr := s.repo.FindByIdempotencyKey(ctx, userID, idempotencyKey); ferr == nil {
				return existing, nil
			}
		}
		return nil, err
	}

	// 5b. On-chain USDT: no synchronous charge and no immediate fulfillment.
	//     Create the deposit intent and return the still-pending order; the
	//     payment module's watcher confirms the transfer and calls back into
	//     FulfillPaidOrder, which runs the same dispatch below with charged=true.
	//     Promo redemption is likewise deferred to fulfillment. (A zero-total
	//     order needs no payment, so it falls through to instant fulfillment.)
	if order.PaymentMethod == PaymentMethodUSDT && total > 0 {
		if _, err := s.usdt.CreateOrderIntent(ctx, userID, order.ID, total, input.UsdtNetwork); err != nil {
			_ = s.repo.UpdateStatus(ctx, order.ID, OrderStatusFailed)
			return nil, err
		}
		return order, nil
	}

	// 6. Charge. Only wallet reaches this point (card via the gateway, and
	//    wallet); the guard stays so a zero-total order simply skips the debit.
	charged := false
	if total > 0 {
		switch order.PaymentMethod {
		case PaymentMethodWallet:
			if _, err := s.wallet.Debit(ctx, userID, total, order.ID.Hex()); err != nil {
				_ = s.repo.UpdateStatus(ctx, order.ID, OrderStatusFailed)
				return nil, err
			}
			charged = true
		default:
			// Card via the configured gateway (validated enabled in step 1). The
			// returned transaction id is persisted so refund/compensation can
			// reverse the charge.
			prov, _ := s.payments.For(string(order.PaymentMethod))
			txn, err := prov.ProcessPayment(ctx, total, currency, order.ID.Hex())
			if err != nil {
				_ = s.repo.UpdateStatus(ctx, order.ID, OrderStatusFailed)
				return nil, &apperrors.AppError{Code: "PAYMENT_FAILED", Message: "the payment was declined", Err: apperrors.ErrBadRequest}
			}
			order.PaymentRef = txn
			_ = s.repo.SetPaymentRef(ctx, order.ID, txn)
			charged = true
		}
	}

	// Record the promo redemption (best-effort): the discount is already applied
	// and the order is persisted/charged, so a rare depleted-race is logged, not
	// failed — consistent with the no-multi-document-transactions model.
	if promoCode != "" {
		if err := s.promo.Redeem(ctx, promoCode); err != nil {
			slog.Warn("promo: redeem failed after order placement", "code", promoCode, "order", order.ID.Hex(), "error", err)
		}
	}

	return s.dispatchFulfillment(ctx, userID, order, charged)
}

// dispatchFulfillment routes an order down one of the four fulfillment paths
// keyed on its resolved mode (spec §2.2). Shared by PlaceOrder (wallet/card,
// synchronously) and FulfillPaidOrder (usdt, after the on-chain payment
// confirms). `charged` tells the compensation path whether a wallet/gateway
// charge must be reversed on failure.
func (s *OrderService) dispatchFulfillment(ctx context.Context, userID bson.ObjectID, order *Order, charged bool) (*Order, error) {
	switch resolveOrderMode(order.Items) {
	case product.FulfillmentModeInventory:
		return s.fulfillInventory(ctx, userID, order, charged)
	case product.FulfillmentModeAPI:
		return s.fulfillAPI(ctx, userID, order, charged)
	case product.FulfillmentModeBridgeDevice:
		return s.fulfillBridge(ctx, userID, order, charged)
	default: // manual_operator (and the safe fallback for mixed carts)
		return s.fulfillProcessing(ctx, order)
	}
}

// resolveOrderMode collapses a cart's per-item modes into the single path the
// order takes. A homogeneous cart routes down its shared mode; a mixed cart
// falls back to manual_operator (preserving today's "park the whole order"
// behavior — no auto-delivery of part of a mixed order). Item modes are derived
// from the type when an item's snapshot is empty (legacy-order safety).
func resolveOrderMode(items []OrderItem) product.FulfillmentMode {
	var mode product.FulfillmentMode
	for i, it := range items {
		m := product.FulfillmentMode(it.FulfillmentMode)
		if m == "" {
			m = product.DeriveMode(product.FulfillmentType(it.FulfillmentType))
		}
		if i == 0 {
			mode = m
			continue
		}
		if m != mode {
			return product.FulfillmentModeManualOperator
		}
	}
	return mode
}

// fulfillInventory claims a code per unit for every item, compensating on
// failure (releasing claimed codes and refunding any wallet charge). It loops
// all order items unconditionally — safe because resolveOrderMode only routes
// here when every item is inventory-mode (so every product has a code pool).
func (s *OrderService) fulfillInventory(ctx context.Context, userID bson.ObjectID, order *Order, charged bool) (*Order, error) {
	claimedProducts := make([]string, 0, len(order.Items))
	var firstCode string
	for _, it := range order.Items {
		pid := it.ProductID.Hex()
		codes, err := s.codes.ClaimForOrder(ctx, pid, order.ID.Hex(), userID.Hex(), it.Qty)
		if err != nil {
			s.compensate(ctx, userID, order, claimedProducts, charged)
			return nil, err
		}
		claimedProducts = append(claimedProducts, pid)
		if firstCode == "" && len(codes) > 0 {
			firstCode = codes[0].Code
		}
	}

	fulfillment := order.Fulfillment
	fulfillment.DeliveredCode = firstCode
	fulfillment.StatusTimeline = append(fulfillment.StatusTimeline, TimelineEvent{Status: "delivered", At: time.Now().UTC()})
	if err := s.repo.UpdateFulfillment(ctx, order.ID, OrderStatusCompleted, fulfillment); err != nil {
		return nil, err
	}
	s.ntf.Notify(ctx, userID, orderCompletedNote(order.ID.Hex(), order.Total, order.Currency))
	if s.loyalty != nil {
		s.loyalty.Award(ctx, userID, order.Total)
	}
	return s.repo.FindByID(ctx, order.ID)
}

// orderCompletedNote builds the customer notification for a completed order —
// shared by the instant code-claim path, a successful api-mode provider, and
// the admin manual-completion endpoint.
func orderCompletedNote(orderID string, total float64, currency string) notification.Note {
	return notification.Note{
		Kind:  notification.KindOrderCompleted,
		Title: "Order completed",
		Body:  fmt.Sprintf("Your order was delivered — $%.2f.", total),
		Data: map[string]string{
			"orderId":  orderID,
			"amount":   fmt.Sprintf("%.2f", total),
			"currency": currency,
		},
	}
}

// fulfillProcessing records a processing order for manual (admin) fulfillment of
// account_credit / transfer items. No codes are claimed.
func (s *OrderService) fulfillProcessing(ctx context.Context, order *Order) (*Order, error) {
	fulfillment := order.Fulfillment
	for _, it := range order.Items {
		if it.FulfillmentType == string(product.FulfillmentCredit) && it.PlayerID != "" {
			fulfillment.CreditedToID = it.PlayerID
			break
		}
	}
	now := time.Now().UTC()
	fulfillment.StatusTimeline = append(fulfillment.StatusTimeline,
		TimelineEvent{Status: "submitted", At: now},
		TimelineEvent{Status: "processing", At: now},
	)
	if err := s.repo.UpdateFulfillment(ctx, order.ID, OrderStatusProcessing, fulfillment); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, order.ID)
}

// park records an order as processing with an explanatory timeline note. It is
// the shared shape for modes whose backend fulfilment isn't wired yet (api with
// no real provider, bridge_device): the order is queued, not failed.
func (s *OrderService) park(ctx context.Context, order *Order, note string) (*Order, error) {
	fulfillment := order.Fulfillment
	now := time.Now().UTC()
	fulfillment.StatusTimeline = append(fulfillment.StatusTimeline,
		TimelineEvent{Status: "submitted", At: now},
		TimelineEvent{Status: "processing", Note: note, At: now},
	)
	if err := s.repo.UpdateFulfillment(ctx, order.ID, OrderStatusProcessing, fulfillment); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, order.ID)
}

// fulfillAPI dispatches an api-mode order to its upstream provider. This is the
// §5 seam: an unconfigured provider id resolves to a stub returning
// ErrNotImplemented, so the order parks. A real adapter (platform/provider
// Panel) drives four outcomes: success completes the order; ErrPending
// (upstream accepted, still working) parks it with the upstream reference for
// the supplier settler; ErrUnavailable (environmental failure — supplier
// balance/throttle/auth/transport ambiguity, no acceptance) parks it WITHOUT a
// reference so a re-dispatch with the same order_uuid stays idempotent; any
// other error is definitively about this order → compensate (refund + fail).
func (s *OrderService) fulfillAPI(ctx context.Context, userID bson.ObjectID, order *Order, charged bool) (*Order, error) {
	provID, in, _ := supplierFulfillInput(order)

	res, err := s.providers.Resolve(provID).Fulfill(ctx, in)
	switch {
	case errors.Is(err, provider.ErrNotImplemented):
		// No real upstream wired yet → queue for future/manual completion.
		return s.park(ctx, order, "awaiting provider integration")
	case errors.Is(err, provider.ErrPending):
		// Upstream accepted but hasn't finished. Persist its reference (park
		// copies order.Fulfillment, so mutating first rides along) — the
		// supplier settler polls CheckStatus on it.
		order.Fulfillment.ProviderRef = res.Reference
		return s.park(ctx, order, "awaiting upstream provider")
	case errors.Is(err, provider.ErrUnavailable):
		// Environmental failure with no upstream acceptance: park, never
		// compensate — the customer paid and the failure isn't theirs. The
		// error detail is operational (never contains customer inputs).
		// Arming the retry clock marks the park re-dispatchable: the supplier
		// settler retries it on the backoff schedule (safe — the same
		// OrderUUID dedupes upstream).
		slog.Warn("order: upstream unavailable, parking", "order", order.ID.Hex(), "error", err)
		next := time.Now().UTC().Add(supplierBackoff[0])
		order.Fulfillment.SupplierNextRetryAt = &next
		return s.park(ctx, order, "upstream temporarily unavailable")
	case err != nil:
		// A wired provider hard-failed → reverse the charge and fail the order.
		s.compensate(ctx, userID, order, nil, charged)
		return nil, err
	default:
		fulfillment := order.Fulfillment
		fulfillment.TransferRef = res.Reference
		fulfillment.ProviderRef = res.Reference
		if len(res.Codes) > 0 {
			fulfillment.DeliveredCode = strings.Join(res.Codes, "\n")
		}
		fulfillment.StatusTimeline = append(fulfillment.StatusTimeline, TimelineEvent{Status: "completed", At: time.Now().UTC()})
		if uerr := s.repo.UpdateFulfillment(ctx, order.ID, OrderStatusCompleted, fulfillment); uerr != nil {
			return nil, uerr
		}
		s.ntf.Notify(ctx, userID, orderCompletedNote(order.ID.Hex(), order.Total, order.Currency))
		if s.loyalty != nil {
			s.loyalty.Award(ctx, userID, order.Total)
		}
		return s.repo.FindByID(ctx, order.ID)
	}
}

// fulfillBridge dispatches a bridge_device-mode order (Lebanese mobile recharge)
// to the bridge module: for a recharge_line it first claims a scratch-card code
// from inventory; it parks the order (processing) BEFORE enqueuing the command so
// a device result can never race a still-pending order; then it enqueues the
// command. A claim/dispatch failure compensates (release the code + refund if
// charged) and fails the order. When the bridge is disabled the order simply
// parks for manual completion — the pre-bridge behavior.
func (s *OrderService) fulfillBridge(ctx context.Context, userID bson.ObjectID, order *Order, charged bool) (*Order, error) {
	if s.bridge == nil || !s.bridge.Enabled() {
		return s.park(ctx, order, "queued for bridge device")
	}
	// PlaceOrder guarantees a single, qty-1 bridge line.
	it := order.Items[0]
	p, perr := s.products.FindByID(ctx, it.ProductID.Hex())
	if perr != nil || p.Bridge == nil || !p.Bridge.Valid() {
		// Misconfigured after purchase — don't fail a paid order; queue it for a
		// human to sort out rather than dispatching an ambiguous command.
		return s.park(ctx, order, "queued for manual recharge")
	}
	phone := normalizeLebanesePhone(it.PlayerID)
	spec := p.Bridge

	// recharge_line: claim one card code up front (while pending) so a shortage
	// fails cleanly with no processing flicker.
	var claimedProducts []string
	var cardCode string
	if spec.Method == product.BridgeMethodRechargeLine {
		codes, err := s.codes.ClaimForOrder(ctx, p.ID.Hex(), order.ID.Hex(), "bridge:"+phone, 1)
		if err != nil {
			s.compensate(ctx, userID, order, nil, charged)
			return nil, err
		}
		claimedProducts = []string{p.ID.Hex()}
		cardCode = codes[0].Code
	}

	// Park BEFORE the command becomes visible to devices.
	if _, err := s.park(ctx, order, "dispatched to bridge device"); err != nil {
		s.compensate(ctx, userID, order, claimedProducts, charged)
		return nil, err
	}

	var amount *float64
	if spec.Method == product.BridgeMethodTransferCredit {
		if v, ok := findVariant(p, it.VariantID.Hex()); ok {
			amount = v.FaceValue
		}
	}
	if err := s.bridge.DispatchOrder(ctx, bridge.DispatchInput{
		OrderID:  order.ID,
		Provider: string(spec.Provider),
		Method:   string(spec.Method),
		Phone:    phone,
		Amount:   amount,
		CardCode: cardCode,
	}); err != nil {
		s.compensate(ctx, userID, order, claimedProducts, charged)
		return nil, err
	}
	return s.repo.FindByID(ctx, order.ID)
}

// compensate reverses a partially-fulfilled order: releases any claimed codes,
// refunds a wallet charge, and marks the order failed (all best effort).
func (s *OrderService) compensate(ctx context.Context, userID bson.ObjectID, order *Order, claimedProducts []string, charged bool) {
	_ = s.codes.ReleaseForOrder(ctx, order.ID.Hex(), claimedProducts)
	if charged {
		if order.PaymentMethod == PaymentMethodWallet {
			_, _ = s.wallet.Refund(ctx, userID, order.Total, order.ID.Hex())
		} else if order.PaymentRef != "" && s.payments != nil {
			if prov, ok := s.payments.For(string(order.PaymentMethod)); ok {
				_ = prov.RefundPayment(ctx, order.PaymentRef)
			}
		}
	}
	_ = s.repo.UpdateStatus(ctx, order.ID, OrderStatusFailed)
}

// FulfillPaidOrder fulfills a USDT order whose on-chain payment has confirmed.
// It implements payment.OrderSettler and is called by the payment watcher, so
// it is idempotent and safe to retry:
//   - It first claims pending → processing atomically (TransitionStatus). That
//     claim is the race guard against the intent-expiry sweep: whichever of the
//     two transitions the order wins, the other sees ErrConflict.
//   - ErrConflict is resolved by re-reading: an order already processing or
//     completed is treated as done (nil, a no-op retry); an order that the
//     sweep already failed returns payment.ErrOrderNotPayable, telling the
//     payment module to credit the paid amount to the wallet instead.
//   - Fulfillment runs with charged=false: there is nothing to reverse
//     on-chain, so a failure (e.g. an out-of-stock code pool) marks the order
//     failed and returns ErrOrderNotPayable, again routing the money to the
//     wallet (once, keyed on the intent ref) rather than stranding it.
func (s *OrderService) FulfillPaidOrder(ctx context.Context, orderID bson.ObjectID, paidUSD float64, txRef string) error {
	_, err := s.repo.TransitionStatus(ctx, orderID,
		[]OrderStatus{OrderStatusPending}, OrderStatusProcessing,
		TimelineEvent{Status: "payment_confirmed", Note: "USDT payment confirmed on-chain", At: time.Now().UTC()},
		bson.D{{Key: "paymentRef", Value: txRef}},
	)
	if err != nil {
		if errors.Is(err, apperrors.ErrConflict) {
			o, ferr := s.repo.FindByID(ctx, orderID)
			if ferr != nil {
				return ferr
			}
			if o.Status == OrderStatusFailed {
				return payment.ErrOrderNotPayable // expiry sweep won → credit the wallet
			}
			return nil // already processing/completed → idempotent no-op
		}
		return err
	}

	o, err := s.repo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	o.PaymentRef = txRef
	if _, ferr := s.dispatchFulfillment(ctx, o.UserID, o, false); ferr != nil {
		// dispatchFulfillment already released any claimed codes and marked the
		// order failed (compensate). The on-chain money is real, so hand it to
		// the payment module to credit to the wallet rather than retrying.
		slog.Warn("order: usdt fulfillment failed after payment — crediting wallet",
			"order", orderID.Hex(), "error", ferr)
		return payment.ErrOrderNotPayable
	}

	// Promo redemption was deferred from PlaceOrder (best-effort).
	if o.PromoCode != "" {
		if rerr := s.promo.Redeem(ctx, o.PromoCode); rerr != nil {
			slog.Warn("promo: redeem failed after usdt fulfillment", "code", o.PromoCode, "order", orderID.Hex(), "error", rerr)
		}
	}
	return nil
}

// FailUnpaidOrder marks a pending USDT order failed (expiry or underpayment).
// Implements payment.OrderSettler; idempotent — an order that already moved on
// (fulfilled, or failed by a prior call) is a no-op.
func (s *OrderService) FailUnpaidOrder(ctx context.Context, orderID bson.ObjectID, reason string) error {
	_, err := s.repo.TransitionStatus(ctx, orderID,
		[]OrderStatus{OrderStatusPending}, OrderStatusFailed,
		TimelineEvent{Status: "failed", Note: reason, At: time.Now().UTC()},
		nil,
	)
	if errors.Is(err, apperrors.ErrConflict) || errors.Is(err, apperrors.ErrNotFound) {
		return nil
	}
	return err
}

// GetOrder returns an order, enforcing that it belongs to userID (a foreign
// order is reported as not found).
func (s *OrderService) GetOrder(ctx context.Context, userID, id bson.ObjectID) (*Order, error) {
	o, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if o.UserID != userID {
		return nil, apperrors.ErrNotFound
	}
	return o, nil
}

// ListOrders returns the caller's order history, newest first.
func (s *OrderService) ListOrders(ctx context.Context, userID bson.ObjectID) ([]*Order, error) {
	return s.repo.FindByUserID(ctx, userID)
}

// findVariant locates a variant by hex id within a product.
func findVariant(p *product.Product, variantID string) (product.Variant, bool) {
	for _, v := range p.Variants {
		if v.ID.Hex() == variantID {
			return v, true
		}
	}
	return product.Variant{}, false
}

// resolveOrderFields snapshots the customer's per-line inputs with their
// server-resolved labels (from the product's InputFields spec — client labels
// are never trusted). Fields with an empty value, an unknown key, or a spec
// marked Sensitive (spec §3.2) are dropped, so sensitive values are never
// persisted on the order. A quantity-type field's value is never trusted from
// the client either — it's always overridden with the line's real qty (and
// always emitted, even if the client sent nothing for it), so the operator
// fulfilling the order sees the quantity that was actually paid for.
func resolveOrderFields(p *product.Product, in []OrderFieldInput, qty int) []OrderField {
	if len(p.InputFields) == 0 {
		return nil
	}
	values := make(map[string]string, len(in))
	for _, f := range in {
		values[f.Key] = f.Value
	}
	var out []OrderField
	for _, def := range p.InputFields {
		if def.Sensitive {
			continue
		}
		if def.Type == product.InputFieldQuantity {
			out = append(out, OrderField{Key: def.Key, Label: def.Label, Value: strconv.Itoa(qty)})
			continue
		}
		v := strings.TrimSpace(values[def.Key])
		if v == "" {
			continue
		}
		out = append(out, OrderField{Key: def.Key, Label: def.Label, Value: v})
	}
	return out
}

// validateQuantityField rejects an order line whose qty falls outside the
// bounds of its product's quantity-type input field (if any) — a legacy
// minimum/maximum purchase quantity that must actually be honored now that
// the field's value is derived from qty rather than freely typed. A
// {min:0,max:0} constraint is a known legacy-import corruption shape
// (migration/internal/transform/inputfields.go's corrupt() case), not a real
// bound, and is treated as unconstrained.
func validateQuantityField(p *product.Product, qty int) error {
	for _, def := range p.InputFields {
		if def.Type != product.InputFieldQuantity || def.Constraints == nil {
			continue
		}
		min, max := def.Constraints.Min, def.Constraints.Max
		if min == nil || max == nil {
			continue
		}
		if *min == 0 && *max == 0 {
			continue // known corrupt shape — not a real constraint
		}
		if float64(qty) < *min || float64(qty) > *max {
			return badRequest(fmt.Sprintf("quantity must be between %g and %g for this product", *min, *max))
		}
	}
	return nil
}

// validateFieldInputs enforces the product's per-field constraints on the
// customer's submitted values: a select value must be one of the field's
// options, an amount value must be numeric and inside its min/max. It mirrors
// validateQuantityField's legacy tolerances — a {min:0,max:0} or min>max
// constraint is import corruption, not a real bound, and is skipped so a
// mis-migrated product never bricks checkout. Empty values stay legal (fields
// are optional at order time; "required" is a client concern), and submitted
// values are never echoed into error messages — a field may be sensitive.
func validateFieldInputs(p *product.Product, in []OrderFieldInput) error {
	if len(p.InputFields) == 0 {
		return nil
	}
	values := make(map[string]string, len(in))
	for _, f := range in {
		values[f.Key] = f.Value
	}
	for _, def := range p.InputFields {
		// Quantity is enforced against the line's real qty by
		// validateQuantityField — never against the client-typed value.
		if def.Type == product.InputFieldQuantity || def.Constraints == nil {
			continue
		}
		v := strings.TrimSpace(values[def.Key])
		if v == "" {
			continue
		}
		switch def.Type {
		case product.InputFieldSelect:
			// An empty options list is legacy free-entry — nothing to enforce.
			if len(def.Constraints.Options) == 0 {
				continue
			}
			ok := false
			for _, opt := range def.Constraints.Options {
				if v == strings.TrimSpace(opt) {
					ok = true
					break
				}
			}
			if !ok {
				return badRequest(fmt.Sprintf("%s must be one of the listed options", fieldName(def)))
			}
		case product.InputFieldAmount:
			lo, hi := def.Constraints.Min, def.Constraints.Max
			if lo != nil && hi != nil && (*lo == 0 && *hi == 0 || *lo > *hi) {
				continue // known corrupt shapes — not real bounds
			}
			if lo == nil && hi == nil {
				continue
			}
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return badRequest(fmt.Sprintf("%s must be a number", fieldName(def)))
			}
			switch {
			case lo != nil && hi != nil && (f < *lo || f > *hi):
				return badRequest(fmt.Sprintf("%s must be between %s and %s", fieldName(def), fmtBound(*lo), fmtBound(*hi)))
			case lo != nil && f < *lo:
				return badRequest(fmt.Sprintf("%s must be at least %s", fieldName(def), fmtBound(*lo)))
			case hi != nil && f > *hi:
				return badRequest(fmt.Sprintf("%s must be at most %s", fieldName(def), fmtBound(*hi)))
			}
		}
	}
	return nil
}

// fieldName names a field in a customer-facing error: the English label when
// set, else the key — never the submitted value (the field may be sensitive).
func fieldName(def product.InputField) string {
	if def.Label.En != "" {
		return def.Label.En
	}
	return def.Key
}

// fmtBound renders a bound in plain decimal for customer-facing messages —
// %g would show a large legacy bound like 50000000000 as "5e+10".
func fmtBound(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
