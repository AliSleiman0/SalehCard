// Package settings will hold platform configuration (store details, payment
// gateways, notification thresholds, admin accounts). Handlers are stubbed for now.
package settings

import (
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// RegisterAdminRoutes registers the admin settings route map onto r (the
// /api/admin group, guarded by AdminOnly). Handlers are stubbed (501).
//
// TODO: implement GET/PUT settings (store name/contact, default language &
// currency, payment gateway toggles + masked keys, notification thresholds) and
// the admin-account permission model (open item flagged in the design).
func RegisterAdminRoutes(r chi.Router, _ *mongo.Database) {
	r.Get("/settings", response.Stub("admin settings"))
	r.Put("/settings", response.Stub("admin settings update"))
}
