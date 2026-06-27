package code

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
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
	r.Get("/upload-history", uploadHistoryHandler(repo))
	r.Post("/products/{id}/codes", h.Upload)
	r.Get("/products/{id}/codes", h.ListCodes)
	r.Put("/products/{id}/stock-threshold", h.SetThreshold)
	r.Get("/codes/{code}", h.Lookup)
}

// uploadHistoryHandler serves GET /api/admin/upload-history?limit= — the recent
// upload batches for the inventory screen. It reads the repository directly (no
// Service method) to keep the Service interface — and its test fakes — unchanged.
func uploadHistoryHandler(repo *MongoRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		history, err := repo.ListUploadHistory(r.Context(), limit)
		if err != nil {
			response.InternalError(w)
			return
		}
		response.OK(w, history)
	}
}

// NewService exposes a constructed Service for callers (e.g. the dashboard) that
// need inventory data without re-wiring the repository.
func NewService(db *mongo.Database) Service {
	return NewCodeService(NewMongoRepository(db))
}
