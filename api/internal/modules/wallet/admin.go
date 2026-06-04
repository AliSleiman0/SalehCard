package wallet

import (
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// RegisterAdminRoutes registers the admin finance route map onto r (the
// /api/admin group, guarded by AdminOnly). Handlers are stubbed (501).
//
// TODO: implement the unified transactions feed (filter by type/user/date/method),
// the revenue summary (by category/method/currency, daily/weekly/monthly, export),
// and the USDT verification queue (approve/reject).
func RegisterAdminRoutes(r chi.Router, _ *mongo.Database) {
	r.Get("/transactions", response.Stub("admin transactions feed"))
	r.Get("/revenue-summary", response.Stub("revenue summary"))
	r.Get("/usdt-verifications", response.Stub("USDT verification queue"))
	r.Put("/usdt-verifications/{id}", response.Stub("USDT verification decision"))
}
