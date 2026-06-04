package promo

import (
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// RegisterAdminRoutes registers the admin promo route map onto r (the /api/admin
// group, guarded by AdminOnly). Handlers are stubbed (501).
//
// TODO: implement promo CRUD — code (or auto-generate), type (percent/fixed/
// cashback), value, min order, max uses, per-user limit, date range, categories.
func RegisterAdminRoutes(r chi.Router, _ *mongo.Database) {
	r.Get("/promos", response.Stub("admin promo list"))
	r.Post("/promos", response.Stub("promo create"))
	r.Get("/promos/{id}", response.Stub("admin promo detail"))
	r.Put("/promos/{id}", response.Stub("promo update"))
	r.Delete("/promos/{id}", response.Stub("promo delete"))
}
