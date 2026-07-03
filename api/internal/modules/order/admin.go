package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
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
// are recorded via rec; customer-visible outcomes also notify via ntf.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder, ntf notification.Notifier) {
	repo := NewMongoRepository(db.Collection("orders"))
	a := &adminHandler{
		repo:   repo,
		users:  user.NewMongoRepository(db),
		wallet: wallet.NewService(db),
		rec:    rec,
		ntf:    ntf,
	}

	r.Get("/orders", a.list)
	// Read-only sold-units read model for the admin product list. Wired off the
	// concrete repo (like product.categoryFacetsHandler) so SoldByProduct stays
	// off the Repository interface and its test fake.
	r.Get("/orders/sold-by-product", soldByProductHandler(repo))
	r.Get("/orders/{id}", a.detail)
	r.Post("/orders/{id}/refund", a.refund)
	r.Post("/orders/refund-bulk", a.refundBulk)
	r.Put("/orders/{id}/status", a.updateStatus)
	r.Post("/orders/{id}/fail", a.markFailed)
}

// errRefundCreditFailed marks a refund whose wallet credit failed: the order was
// left in its pre-refund state (compensated) so the admin can retry.
var errRefundCreditFailed = errors.New("refund credit failed")

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
	ntf    notification.Notifier
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

	if _, err := a.doRefund(r.Context(), id, strings.TrimSpace(body.Reason)); err != nil {
		if errors.Is(err, errRefundCreditFailed) {
			response.Error(w, http.StatusInternalServerError, "REFUND_CREDIT_FAILED",
				"the wallet credit failed; the order was left unrefunded — retry")
			return
		}
		a.writeTransitionError(w, err, "order is not refundable (already refunded, failed, or still pending)")
		return
	}
	a.respondFresh(w, r, id)
}

// refundBulk handles POST /api/admin/orders/refund-bulk — refunds many orders in
// one call. Because there are no cross-document transactions, each order owns its
// own atomic lock + wallet ledger + audit row, so a failure on one order never
// aborts the rest: the response is a per-id result array (refunded / conflict /
// error) plus a refunded count.
func (a *adminHandler) refundBulk(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs    []string `json:"ids"`
		Reason string   `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if len(body.IDs) == 0 {
		response.BadRequest(w, "no order ids supplied")
		return
	}
	reason := strings.TrimSpace(body.Reason)

	type outcome struct {
		ID      string `json:"id"`
		Status  string `json:"status"` // refunded | conflict | error
		Message string `json:"message,omitempty"`
	}
	results := make([]outcome, 0, len(body.IDs))
	refunded := 0
	for _, raw := range body.IDs {
		id, err := bson.ObjectIDFromHex(raw)
		if err != nil {
			results = append(results, outcome{ID: raw, Status: "error", Message: "invalid id"})
			continue
		}
		switch _, err := a.doRefund(r.Context(), id, reason); {
		case err == nil:
			results = append(results, outcome{ID: raw, Status: "refunded"})
			refunded++
		case errors.Is(err, apperrors.ErrConflict):
			results = append(results, outcome{ID: raw, Status: "conflict", Message: "not refundable (already refunded, failed, or still pending)"})
		case errors.Is(err, apperrors.ErrNotFound):
			results = append(results, outcome{ID: raw, Status: "error", Message: "not found"})
		case errors.Is(err, errRefundCreditFailed):
			results = append(results, outcome{ID: raw, Status: "error", Message: "wallet credit failed — left unrefunded"})
		default:
			results = append(results, outcome{ID: raw, Status: "error", Message: "internal error"})
		}
	}
	response.OK(w, map[string]any{"refunded": refunded, "total": len(body.IDs), "results": results})
}

// doRefund performs the guarded refund transition + wallet credit + audit +
// customer notification for one order, returning the pre-refund order on success.
// It is the shared body of the single and bulk refund endpoints. Errors:
// apperrors.ErrNotFound (missing), apperrors.ErrConflict (not refundable), or
// errRefundCreditFailed (money couldn't be returned; order left compensated).
func (a *adminHandler) doRefund(ctx context.Context, id bson.ObjectID, reason string) (*Order, error) {
	before, err := a.repo.TransitionStatus(ctx, id,
		[]OrderStatus{OrderStatusProcessing, OrderStatusCompleted},
		OrderStatusRefunded,
		TimelineEvent{Status: "refunded", Note: reason, At: time.Now().UTC()},
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Credit the money back for wallet-paid orders. Card/usdt never charged
	// anything (mock-approved historically), so there is nothing to reverse.
	if before.PaymentMethod == PaymentMethodWallet && before.Total > 0 {
		if _, err := a.wallet.Refund(ctx, before.UserID, before.Total, before.ID.Hex()); err != nil {
			// Compensation: put the order back in its pre-refund state so the admin
			// can retry. The wallet service already reversed any partial credit
			// internally (mandatory ledger).
			if _, rerr := a.repo.TransitionStatus(ctx, id,
				[]OrderStatus{OrderStatusRefunded}, before.Status,
				TimelineEvent{Status: "refund_reverted", Note: "wallet credit failed", At: time.Now().UTC()},
				nil,
			); rerr != nil {
				slog.Error("order refund: wallet credit failed AND status revert failed — order marked refunded without credit",
					"order", id.Hex(), "creditError", err, "revertError", rerr)
			}
			return nil, errRefundCreditFailed
		}
	}

	a.rec.Record(ctx, audit.Entry{
		Action:     audit.ActionOrderRefund,
		TargetType: "order",
		TargetID:   id.Hex(),
		Summary: map[string]any{
			"amount": before.Total, "paymentMethod": string(before.PaymentMethod),
			"fromStatus": string(before.Status), "reason": reason,
		},
	})
	data := map[string]string{
		"orderId":  id.Hex(),
		"amount":   fmt.Sprintf("%.2f", before.Total),
		"currency": before.Currency,
	}
	if reason != "" {
		data["reason"] = reason
	}
	a.ntf.Notify(ctx, before.UserID, notification.Note{
		Kind:  notification.KindOrderRefunded,
		Title: "Order refunded",
		Body:  fmt.Sprintf("$%.2f was returned to your wallet.", before.Total),
		Data:  data,
	})
	return before, nil
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
	a.ntf.Notify(r.Context(), before.UserID, orderCompletedNote(id.Hex(), before.Total, before.Currency))
	a.respondFresh(w, r, id)
}

// markFailed handles POST /api/admin/orders/{id}/fail — an admin cleanup for a
// stuck pending order (one that crashed mid-placement before its fulfillment
// step ran). It moves pending → failed with NO wallet movement: nothing on the
// order records whether the wallet was actually debited, so an automatic credit
// could double-refund an order that was never charged. If a charge did occur,
// the admin reverses it manually via the customer's wallet adjustment.
func (a *adminHandler) markFailed(w http.ResponseWriter, r *http.Request) {
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
		[]OrderStatus{OrderStatusPending},
		OrderStatusFailed,
		TimelineEvent{Status: "failed", Note: reason, At: time.Now().UTC()},
		nil,
	)
	if err != nil {
		a.writeTransitionError(w, err, "only pending orders can be marked failed")
		return
	}

	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionOrderStatus,
		TargetType: "order",
		TargetID:   id.Hex(),
		Summary: map[string]any{
			"fromStatus": string(before.Status), "toStatus": string(OrderStatusFailed),
			"reason": reason,
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
