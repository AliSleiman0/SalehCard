package order

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/code"
	"github.com/AliSleiman0/salehcard/api/internal/modules/kyc"
	"github.com/AliSleiman0/salehcard/api/internal/modules/loyalty"
	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/modules/offer"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/promo"
	"github.com/AliSleiman0/salehcard/api/internal/modules/reseller"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/payments"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
)

// RegisterRoutes wires the customer-facing order routes onto r. /api/v1/orders
// is guarded by AuthRequired. The order service depends on the product catalog
// (pricing), code inventory (fulfillment), wallet (payment), the notifier
// (customer inbox + push on instant completion), and — for on-chain USDT
// checkout — the payment intents port (nil disables the usdt method). It
// returns the constructed *OrderService so the caller can wire it back as the
// payment module's OrderSettler (the async fulfillment callback).
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config, ntf notification.Notifier, usdt usdtIntents, brdg bridgeDispatcher) *OrderService {
	repo := NewMongoRepository(db.Collection("orders"))
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("order: failed to ensure indexes", "error", err)
	}
	// Deliberately option-less (no offer/reseller enrichment): PlaceOrder
	// re-prices from raw variant.Price and applies offer/reseller pricing
	// itself — an enriched service here could silently double-discount.
	products := product.NewProductService(product.NewMongoRepository(db))
	codes := code.NewService(db)
	wlt := wallet.NewService(db)
	promos := promo.NewPromoService(promo.NewMongoRepository(db.Collection("promos")))
	offers := offer.NewOfferService(offer.NewMongoRepository(db.Collection("offers")))
	// Upstream fulfillment adapters. The reference (mock) adapter is registered
	// when FULFILLMENT_MOCK is on, so an api-mode order routed to its id completes
	// instead of parking. Real adapters register here as credentials arrive (§5).
	var adapters []provider.Provider
	if cfg.FulfillmentMock {
		adapters = append(adapters, provider.NewReference(cfg.FulfillmentMockID))
	}
	providers := provider.NewRegistry(adapters...)
	// Payment gateway: "mock" enables the sandbox card/usdt path; default keeps
	// checkout wallet-only.
	pay := payments.New(payments.Config{Provider: cfg.PaymentProvider})
	kycGate := kyc.NewGate(db)
	margins := reseller.NewMongoRepository(db)
	points := loyalty.NewAwarder(db)

	svc := NewOrderService(repo, products, codes, wlt, promos, offers, providers, pay, usdt, kycGate, margins, ntf, points, brdg)
	h := NewHandler(svc, usdt)

	r.Route("/api/v1/orders", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Use(auth.RequireActive(db))
		r.Post("/", h.PlaceOrder)
		r.Get("/", h.ListOrders)
		r.Get("/{id}", h.GetOrder)
	})
	return svc
}
