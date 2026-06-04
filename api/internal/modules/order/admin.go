package order

import (
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// RegisterAdminRoutes registers the admin order route map onto r (the /api/admin
// group, guarded by AdminOnly). Handlers are stubbed (501) pending implementation.
//
// TODO: implement filtering (status, fulfillment type, payment method, date),
// search by order id / customer email / code, code reveal on detail, refunds,
// and manual transfer status updates.
func RegisterAdminRoutes(r chi.Router, _ *mongo.Database) {
	r.Get("/orders", response.Stub("admin order list"))
	r.Get("/orders/{id}", response.Stub("admin order detail"))
	r.Post("/orders/{id}/refund", response.Stub("order refund"))
	r.Put("/orders/{id}/status", response.Stub("transfer status update"))
}
