package order

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/modules/code"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/promo"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// badRequest builds a 400-class AppError with a machine code.
func badRequest(msg string) error {
	return &apperrors.AppError{Code: "BAD_REQUEST", Message: msg, Err: apperrors.ErrBadRequest}
}

// Service defines the business-logic operations for the order domain.
type Service interface {
	PlaceOrder(ctx context.Context, userID bson.ObjectID, isReseller bool, idempotencyKey string, input PlaceOrderInput) (*Order, error)
	GetOrder(ctx context.Context, userID, id bson.ObjectID) (*Order, error)
	ListOrders(ctx context.Context, userID bson.ObjectID) ([]*Order, error)
}

// OrderService is the concrete implementation of Service. It orchestrates
// pricing (from the product catalog), payment (wallet/card/usdt), and
// fulfillment — dispatching each order down one of four paths keyed on the
// product's fulfillment mode (inventory / api / manual_operator / bridge_device).
type OrderService struct {
	repo      Repository
	products  product.Service
	codes     code.Service
	wallet    wallet.Service
	promo     promo.Service
	providers *provider.Registry
}

// NewOrderService constructs an OrderService wired to the catalog, code
// inventory, wallet, promo, and upstream-provider registry it depends on.
func NewOrderService(repo Repository, products product.Service, codes code.Service, wlt wallet.Service, promos promo.Service, providers *provider.Registry) *OrderService {
	return &OrderService{repo: repo, products: products, codes: codes, wallet: wlt, promo: promos, providers: providers}
}

// PlaceOrder validates and prices an order server-side, charges the chosen
// payment method, and fulfills code items by claiming inventory. It follows an
// order-first strategy: the order is persisted as pending, then charged and
// fulfilled, then flipped to completed/processing; any failure compensates
// (releasing claimed codes, refunding the wallet) and marks the order failed.
func (s *OrderService) PlaceOrder(ctx context.Context, userID bson.ObjectID, isReseller bool, idempotencyKey string, input PlaceOrderInput) (*Order, error) {
	// 1. Validate.
	if len(input.Items) == 0 {
		return nil, badRequest("order must contain at least one item")
	}
	switch input.PaymentMethod {
	case PaymentMethodWallet, PaymentMethodCard, PaymentMethodUSDT:
	default:
		return nil, badRequest("unsupported payment method")
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
		if isReseller && variant.ResellerPrice != nil {
			price = *variant.ResellerPrice
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
			PlayerID:            in.PlayerID,
			Recipient:           in.Recipient,
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

	// 6. Charge.
	charged := false
	if input.PaymentMethod == PaymentMethodWallet {
		if _, err := s.wallet.Debit(ctx, userID, total, order.ID.Hex()); err != nil {
			_ = s.repo.UpdateStatus(ctx, order.ID, OrderStatusFailed)
			return nil, err
		}
		charged = true
	}
	// card / usdt are mock-approved: nothing to charge.

	// Record the promo redemption (best-effort): the discount is already applied
	// and the order is persisted/charged, so a rare depleted-race is logged, not
	// failed — consistent with the no-multi-document-transactions model.
	if promoCode != "" {
		if err := s.promo.Redeem(ctx, promoCode); err != nil {
			slog.Warn("promo: redeem failed after order placement", "code", promoCode, "order", order.ID.Hex(), "error", err)
		}
	}

	// 7. Dispatch on the order's fulfillment mode (spec §2.2).
	switch resolveOrderMode(order.Items) {
	case product.FulfillmentModeInventory:
		return s.fulfillInventory(ctx, userID, order, charged)
	case product.FulfillmentModeAPI:
		return s.fulfillAPI(ctx, userID, order, charged)
	case product.FulfillmentModeBridgeDevice:
		return s.fulfillBridge(ctx, order)
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
	return s.repo.FindByID(ctx, order.ID)
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
// §5 seam: today every provider id resolves to a stub returning ErrNotImplemented,
// so the order parks. A real adapter that succeeds completes the order; one that
// hard-fails compensates (refund + mark failed).
func (s *OrderService) fulfillAPI(ctx context.Context, userID bson.ObjectID, order *Order, charged bool) (*Order, error) {
	var provID *int
	var in provider.FulfillInput
	for _, it := range order.Items {
		if it.FulfillmentMode == string(product.FulfillmentModeAPI) {
			provID = it.FulfillmentProvider
			in = provider.FulfillInput{ProductID: it.ProductID.Hex(), PlayerID: it.PlayerID, Qty: it.Qty}
			break
		}
	}

	res, err := s.providers.Resolve(provID).Fulfill(ctx, in)
	switch {
	case errors.Is(err, provider.ErrNotImplemented):
		// No real upstream wired yet → queue for future/manual completion.
		return s.park(ctx, order, "awaiting provider integration")
	case err != nil:
		// A wired provider hard-failed → reverse the charge and fail the order.
		s.compensate(ctx, userID, order, nil, charged)
		return nil, err
	default:
		fulfillment := order.Fulfillment
		fulfillment.TransferRef = res.Reference
		fulfillment.StatusTimeline = append(fulfillment.StatusTimeline, TimelineEvent{Status: "completed", At: time.Now().UTC()})
		if uerr := s.repo.UpdateFulfillment(ctx, order.ID, OrderStatusCompleted, fulfillment); uerr != nil {
			return nil, uerr
		}
		return s.repo.FindByID(ctx, order.ID)
	}
}

// fulfillBridge dispatches a bridge_device-mode order (Lebanese mobile recharge).
// The bridge module (spec §10) is not built yet, so the order parks queued for a
// device; a bridge device is just another operator type that will pick it up.
func (s *OrderService) fulfillBridge(ctx context.Context, order *Order) (*Order, error) {
	return s.park(ctx, order, "queued for bridge device")
}

// compensate reverses a partially-fulfilled order: releases any claimed codes,
// refunds a wallet charge, and marks the order failed (all best effort).
func (s *OrderService) compensate(ctx context.Context, userID bson.ObjectID, order *Order, claimedProducts []string, charged bool) {
	_ = s.codes.ReleaseForOrder(ctx, order.ID.Hex(), claimedProducts)
	if charged {
		_, _ = s.wallet.Refund(ctx, userID, order.Total, order.ID.Hex())
	}
	_ = s.repo.UpdateStatus(ctx, order.ID, OrderStatusFailed)
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
