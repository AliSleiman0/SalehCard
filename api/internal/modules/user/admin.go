package user

import (
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// RegisterAdminRoutes registers the admin user route map onto r (the /api/admin
// group, guarded by AdminOnly). Handlers are stubbed (501) pending implementation.
//
// TODO: implement list (filter by role/status, search), detail, role changes,
// suspend/reactivate, and manual wallet adjustment (credit/debit with reason log).
func RegisterAdminRoutes(r chi.Router, _ *mongo.Database) {
	r.Get("/users", response.Stub("admin user list"))
	r.Get("/users/{id}", response.Stub("admin user detail"))
	r.Put("/users/{id}/role", response.Stub("user role update"))
	r.Put("/users/{id}/status", response.Stub("user status update"))
	r.Post("/users/{id}/wallet-adjust", response.Stub("wallet adjustment"))
}
