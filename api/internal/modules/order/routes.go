package order

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/code"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
)

// RegisterRoutes wires the customer-facing order routes onto r. /api/v1/orders
// is guarded by AuthRequired. The order service depends on the product catalog
// (pricing), code inventory (fulfillment), and wallet (payment).
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config) {
	repo := NewMongoRepository(db.Collection("orders"))
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("order: failed to ensure indexes", "error", err)
	}
	products := product.NewProductService(product.NewMongoRepository(db))
	codes := code.NewService(db)
	wlt := wallet.NewService(db)
	// No real upstream adapters yet → every api-mode order resolves to a stub
	// and parks. Register adapters here as the owner provides credentials (§5).
	providers := provider.NewRegistry()

	svc := NewOrderService(repo, products, codes, wlt, providers)
	h := NewHandler(svc)

	r.Route("/api/v1/orders", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Post("/", h.PlaceOrder)
		r.Get("/", h.ListOrders)
		r.Get("/{id}", h.GetOrder)
	})
}
