package reseller

import (
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// RegisterAdminRoutes registers the admin reseller route map onto r (the
// /api/admin group, guarded by AdminOnly). Handlers are stubbed (501).
//
// TODO: implement reseller list/detail, tier assignment, sub-balance adjustment,
// per-product pricing overrides, and tier CRUD (Bronze/Silver/Gold).
func RegisterAdminRoutes(r chi.Router, _ *mongo.Database) {
	r.Get("/resellers", response.Stub("admin reseller list"))
	r.Get("/resellers/{id}", response.Stub("admin reseller detail"))
	r.Put("/resellers/{id}/tier", response.Stub("reseller tier assignment"))
	r.Post("/resellers/{id}/balance-adjust", response.Stub("reseller balance adjustment"))

	r.Get("/reseller-tiers", response.Stub("tier list"))
	r.Post("/reseller-tiers", response.Stub("tier create"))
	r.Put("/reseller-tiers/{id}", response.Stub("tier update"))
	r.Delete("/reseller-tiers/{id}", response.Stub("tier delete"))
}
