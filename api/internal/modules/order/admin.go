package order

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// RegisterAdminRoutes mounts the admin order routes onto r (the /api/admin
// group, guarded by AdminOnly). The read endpoints (list/detail) are
// implemented; refund and manual status updates remain stubbed (they touch the
// wallet ledger / manual fulfillment, handled in a later step).
//
// TODO: filtering (status, fulfillment type, payment method, date), search by
// order id / customer email / code, refunds, and manual transfer completion.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	repo := NewMongoRepository(db.Collection("orders"))
	a := &adminHandler{repo: repo}

	r.Get("/orders", a.list)
	r.Get("/orders/{id}", a.detail)
	r.Post("/orders/{id}/refund", response.Stub("order refund"))
	r.Put("/orders/{id}/status", response.Stub("transfer status update"))
}

type adminHandler struct {
	repo Repository
}

// list handles GET /api/admin/orders (paginated, newest first).
func (a *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	p := pagination.ParseParams(r)
	orders, total, err := a.repo.ListAll(r.Context(), p)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OKWithMeta(w, orders, pagination.CalcMeta(p, total))
}

// detail handles GET /api/admin/orders/{id}.
func (a *adminHandler) detail(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	order, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == apperrors.ErrNotFound {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, order)
}
