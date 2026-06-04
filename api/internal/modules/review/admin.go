package review

import (
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// RegisterAdminRoutes registers the admin review-moderation route map onto r
// (the /api/admin group, guarded by AdminOnly). Handlers are stubbed (501).
//
// TODO: implement the moderation queue (filter by status, bulk approve/reject)
// and per-review approve/reject/delete.
func RegisterAdminRoutes(r chi.Router, _ *mongo.Database) {
	r.Get("/reviews", response.Stub("admin review queue"))
	r.Put("/reviews/{id}", response.Stub("review moderation"))
}
