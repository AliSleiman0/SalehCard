package order

import (
	"context"
	"errors"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/modules/code"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
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
// fulfillment (claiming codes from inventory).
type OrderService struct {
	repo     Repository
	products product.Service
	codes    code.Service
	wallet   wallet.Service
}

// NewOrderService constructs an OrderService wired to the catalog, code
// inventory, and wallet services it depends on.
func NewOrderService(repo Repository, products product.Service, codes code.Service, wlt wallet.Service) *OrderService {
	return &OrderService{repo: repo, products: products, codes: codes, wallet: wlt}
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

	// 3. Re-price every line from the catalog (never trust client prices).
	items := make([]OrderItem, 0, len(input.Items))
	var subtotal float64
	autoFulfill := true
	codeNeed := map[string]int{} // productID(hex) -> qty for code items
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
		if p.FulfillmentType != product.FulfillmentCode {
			autoFulfill = false
		} else {
			codeNeed[p.ID.Hex()] += in.Qty
		}
		items = append(items, OrderItem{
			ProductID:       p.ID,
			VariantID:       variant.ID,
			Title:           p.Title,
			Denomination:    variant.Denomination,
			Category:        p.Category,
			Qty:             in.Qty,
			Price:           price,
			FulfillmentType: ft,
			PlayerID:        in.PlayerID,
			Recipient:       in.Recipient,
		})
		subtotal += price * float64(in.Qty)
	}
	total := subtotal // promo codes are a no-op until the promo module lands.

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

	// 7. Fulfill.
	if autoFulfill {
		return s.fulfillCodes(ctx, userID, order, charged)
	}
	return s.fulfillProcessing(ctx, order)
}

// fulfillCodes claims a code per unit for every (code) item, compensating on
// failure (releasing claimed codes and refunding any wallet charge).
func (s *OrderService) fulfillCodes(ctx context.Context, userID bson.ObjectID, order *Order, charged bool) (*Order, error) {
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
