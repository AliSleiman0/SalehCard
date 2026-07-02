package order

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
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

// walletRefunder is the slice of the wallet service the admin order handler
// needs to credit a refund back (kept minimal for testability).
type walletRefunder interface {
	Refund(ctx context.Context, userID bson.ObjectID, amount float64, ref string) (*wallet.WalletTransaction, error)
}

// RegisterAdminRoutes mounts the admin order routes onto r (the /api/admin
// group, guarded by AdminOnly): list/detail (enriched with the customer),
// refund, and manual completion of processing orders. Money/status mutations
// are recorded via rec.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder) {
	repo := NewMongoRepository(db.Collection("orders"))
	a := &adminHandler{
		repo:   repo,
		users:  user.NewMongoRepository(db),
		wallet: wallet.NewService(db),
		rec:    rec,
	}

	r.Get("/orders", a.list)
	// Read-only sold-units read model for the admin product list. Wired off the
	// concrete repo (like product.categoryFacetsHandler) so SoldByProduct stays
	// off the Repository interface and its test fake.
	r.Get("/orders/sold-by-product", soldByProductHandler(repo))
	r.Get("/orders/{id}", a.detail)
	r.Post("/orders/{id}/refund", a.refund)
	r.Put("/orders/{id}/status", a.updateStatus)
}

// soldByProductHandler serves GET /api/admin/orders/sold-by-product — a map of
// product id -> total units sold across completed orders. It feeds the admin
// product list's "Sold" column.
func soldByProductHandler(repo *MongoRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sold, err := repo.SoldByProduct(r.Context())
		if err != nil {
			response.InternalError(w)
			return
		}
		response.OK(w, sold)
	}
}

type adminHandler struct {
	repo   Repository
	users  customerLookup
	wallet walletRefunder
	rec    audit.Recorder
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

// refund handles POST /api/admin/orders/{id}/refund — reverse a processing or
// completed order. The guarded status transition (a single FindOneAndUpdate)
// is the double-refund lock: exactly one of two concurrent refunds wins. A
// wallet charge is credited back through the mandatory-ledger wallet service;
// delivered codes are never re-pooled (the customer has seen them — a refunded
// completed order intentionally burns its codes).
func (a *adminHandler) refund(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body) // reason is optional; an empty body is fine
	}
	reason := strings.TrimSpace(body.Reason)

	before, err := a.repo.TransitionStatus(r.Context(), id,
		[]OrderStatus{OrderStatusProcessing, OrderStatusCompleted},
		OrderStatusRefunded,
		TimelineEvent{Status: "refunded", Note: reason, At: time.Now().UTC()},
		nil,
	)
	if err != nil {
		a.writeTransitionError(w, err, "order is not refundable (already refunded, failed, or still pending)")
		return
	}

	// Credit the money back for wallet-paid orders. Card/usdt never charged
	// anything (mock-approved historically), so there is nothing to reverse.
	if before.PaymentMethod == PaymentMethodWallet && before.Total > 0 {
		if _, err := a.wallet.Refund(r.Context(), before.UserID, before.Total, before.ID.Hex()); err != nil {
			// Compensation: put the order back in its pre-refund state so the
			// admin can retry. The wallet service already reversed any partial
			// credit internally (mandatory ledger).
			if _, rerr := a.repo.TransitionStatus(r.Context(), id,
				[]OrderStatus{OrderStatusRefunded}, before.Status,
				TimelineEvent{Status: "refund_reverted", Note: "wallet credit failed", At: time.Now().UTC()},
				nil,
			); rerr != nil {
				slog.Error("order refund: wallet credit failed AND status revert failed — order marked refunded without credit",
					"order", id.Hex(), "creditError", err, "revertError", rerr)
			}
			response.Error(w, http.StatusInternalServerError, "REFUND_CREDIT_FAILED",
				"the wallet credit failed; the order was left unrefunded — retry")
			return
		}
	}

	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionOrderRefund,
		TargetType: "order",
		TargetID:   id.Hex(),
		Summary: map[string]any{
			"amount": before.Total, "paymentMethod": string(before.PaymentMethod),
			"fromStatus": string(before.Status), "reason": reason,
		},
	})
	a.respondFresh(w, r, id)
}

// updateStatus handles PUT /api/admin/orders/{id}/status — manual completion
// of a processing (account_credit / transfer / parked) order. Completion is
// the ONLY transition this endpoint allows: anything money-reversing goes
// through /refund, and `failed` stays internal to PlaceOrder compensation, so
// there is exactly one money-out path.
func (a *adminHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	var body struct {
		Status      string `json:"status"`
		Note        string `json:"note"`
		TransferRef string `json:"transferRef"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if body.Status != string(OrderStatusCompleted) {
		response.BadRequest(w, "status must be completed (use the refund endpoint to reverse an order)")
		return
	}

	var extra bson.D
	if ref := strings.TrimSpace(body.TransferRef); ref != "" {
		extra = bson.D{{Key: "fulfillment.transferRef", Value: ref}}
	}

	before, err := a.repo.TransitionStatus(r.Context(), id,
		[]OrderStatus{OrderStatusProcessing},
		OrderStatusCompleted,
		TimelineEvent{Status: "completed", Note: strings.TrimSpace(body.Note), At: time.Now().UTC()},
		extra,
	)
	if err != nil {
		a.writeTransitionError(w, err, "only processing orders can be completed")
		return
	}

	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionOrderStatus,
		TargetType: "order",
		TargetID:   id.Hex(),
		Summary: map[string]any{
			"fromStatus": string(before.Status), "toStatus": string(OrderStatusCompleted),
			"transferRef": strings.TrimSpace(body.TransferRef), "note": strings.TrimSpace(body.Note),
		},
	})
	a.respondFresh(w, r, id)
}

// writeTransitionError maps TransitionStatus errors: missing order → 404,
// wrong current status → 409 with the given message.
func (a *adminHandler) writeTransitionError(w http.ResponseWriter, err error, conflictMsg string) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	case errors.Is(err, apperrors.ErrConflict):
		response.Error(w, http.StatusConflict, "INVALID_STATUS", conflictMsg)
	default:
		response.InternalError(w)
	}
}

// respondFresh returns the order's current (post-mutation) admin view.
func (a *adminHandler) respondFresh(w http.ResponseWriter, r *http.Request, id bson.ObjectID) {
	o, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
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
