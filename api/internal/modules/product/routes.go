package product

import (
	"context"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RegisterRoutes wires up the product module and mounts all routes onto r.
func RegisterRoutes(r chi.Router, db *mongo.Database) {
	repo := NewMongoRepository(db)

	// Best-effort index creation at startup; log or handle errors in production.
	_ = EnsureIndexes(context.Background(), db)

	svc := NewProductService(repo)
	h := NewHandler(svc)

	r.Route("/api/v1/products", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)
		r.Post("/", h.Create)     // TODO: admin auth middleware
		r.Patch("/{id}", h.Update) // TODO: admin auth middleware
		r.Delete("/{id}", h.Delete) // TODO: admin auth middleware
	})
}
