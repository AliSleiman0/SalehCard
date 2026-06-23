package category

import (
	"context"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

// RegisterRoutes wires the read-only category endpoint onto r. It reuses the
// product repository as a ProductCounter so root tiles can show live product
// counts (one-way dependency: product does not import category).
func RegisterRoutes(r chi.Router, db *mongo.Database) {
	repo := NewMongoRepository(db)

	// Best-effort index creation at startup.
	_ = EnsureIndexes(context.Background(), db)

	svc := NewCategoryService(repo, product.NewMongoRepository(db))
	h := NewHandler(svc)

	r.Get("/api/v1/categories", h.List)
}
