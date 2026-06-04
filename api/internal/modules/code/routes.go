package code

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RegisterAdminRoutes mounts the inventory/code admin routes onto r (the
// /api/admin group, already guarded by AdminOnly). It is fully implemented and
// extends the product reference slice.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	repo := NewMongoRepository(db)
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("code: failed to ensure indexes", "error", err)
	}
	svc := NewCodeService(repo)
	h := NewHandler(svc)

	r.Get("/inventory", h.Inventory)
	r.Post("/products/{id}/codes", h.Upload)
	r.Get("/products/{id}/codes", h.ListCodes)
	r.Put("/products/{id}/stock-threshold", h.SetThreshold)
	r.Get("/codes/{code}", h.Lookup)
}

// NewService exposes a constructed Service for callers (e.g. the dashboard) that
// need inventory data without re-wiring the repository.
func NewService(db *mongo.Database) Service {
	return NewCodeService(NewMongoRepository(db))
}
