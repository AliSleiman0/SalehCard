package order

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// customerLookup is the slice of the user repository the admin order handler
// needs to enrich orders with their customer (kept minimal for testability).
type customerLookup interface {
	FindByID(ctx context.Context, id bson.ObjectID) (*user.User, error)
	FindByIDs(ctx context.Context, ids []bson.ObjectID) ([]*user.User, error)
	FindByEmailLike(ctx context.Context, q string) ([]*user.User, error)
}

// CustomerInfo is the customer summary attached to each admin order view. The
// User model has no name field, so the frontend derives a display name from the
// email (or phone) — we expose the raw identifiers here.
type CustomerInfo struct {
	ID    string  `json:"id"`
	Email string  `json:"email"`
	Phone *string `json:"phone,omitempty"`
}

// adminOrderView is an order plus its resolved customer. Embedding *Order
// promotes all of the order's JSON fields and just adds `customer`.
type adminOrderView struct {
	*Order
	Customer *CustomerInfo `json:"customer"`
}

// RegisterAdminRoutes mounts the admin order routes onto r (the /api/admin
// group, guarded by AdminOnly). The read endpoints (list/detail) are
// implemented and enrich each order with its customer; refund and manual status
// updates remain stubbed (they touch the wallet ledger / manual fulfillment,
// handled in a later step).
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	a := &adminHandler{
		repo:  NewMongoRepository(db.Collection("orders")),
		users: user.NewMongoRepository(db),
	}

	r.Get("/orders", a.list)
	r.Get("/orders/{id}", a.detail)
	r.Post("/orders/{id}/refund", response.Stub("order refund"))
	r.Put("/orders/{id}/status", response.Stub("transfer status update"))
}

type adminHandler struct {
	repo  Repository
	users customerLookup
}

// list handles GET /api/admin/orders — paginated, newest first, with optional
// status / fulfillmentType / paymentMethod filters and a search term (q) that
// matches an order id, a delivered code, or a customer email.
func (a *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	p := pagination.ParseParams(r)
	q := r.URL.Query()

	f := OrderFilter{
		Status:          OrderStatus(strings.TrimSpace(q.Get("status"))),
		PaymentMethod:   PaymentMethod(strings.TrimSpace(q.Get("paymentMethod"))),
		FulfillmentType: strings.TrimSpace(q.Get("fulfillmentType")),
	}
	if search := strings.TrimSpace(q.Get("q")); search != "" {
		if id, err := bson.ObjectIDFromHex(search); err == nil {
			f.OrderID = &id
		} else {
			users, err := a.users.FindByEmailLike(r.Context(), search)
			if err != nil {
				response.InternalError(w)
				return
			}
			f.SearchRaw = search
			f.UserIDs = make([]bson.ObjectID, 0, len(users))
			for _, u := range users {
				f.UserIDs = append(f.UserIDs, u.ID)
			}
		}
	}

	orders, total, err := a.repo.ListAll(r.Context(), f, p)
	if err != nil {
		response.InternalError(w)
		return
	}

	views, err := a.enrich(r.Context(), orders)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OKWithMeta(w, views, pagination.CalcMeta(p, total))
}

// detail handles GET /api/admin/orders/{id}.
func (a *adminHandler) detail(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	o, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == apperrors.ErrNotFound {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}

	view := adminOrderView{Order: o}
	if u, err := a.users.FindByID(r.Context(), o.UserID); err == nil {
		view.Customer = customerOf(u)
	}
	response.OK(w, view)
}

// enrich resolves the customer for every order in a single batched query and
// returns the order+customer views in the same order.
func (a *adminHandler) enrich(ctx context.Context, orders []*Order) ([]adminOrderView, error) {
	ids := make([]bson.ObjectID, 0, len(orders))
	for _, o := range orders {
		ids = append(ids, o.UserID)
	}
	users, err := a.users.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[bson.ObjectID]*user.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}

	views := make([]adminOrderView, len(orders))
	for i, o := range orders {
		views[i] = adminOrderView{Order: o}
		if u, ok := byID[o.UserID]; ok {
			views[i].Customer = customerOf(u)
		}
	}
	return views, nil
}

func customerOf(u *user.User) *CustomerInfo {
	return &CustomerInfo{ID: u.ID.Hex(), Email: u.Email, Phone: u.Phone}
}
