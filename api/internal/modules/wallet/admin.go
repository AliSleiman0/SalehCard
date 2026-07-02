package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// adminTopUpView is a request plus its customer contact. The user package
// imports wallet, so wallet reads the users collection directly to keep the
// dependency graph acyclic (same pattern as user/admin.go reading orders).
type adminTopUpView struct {
	*TopUpRequest
	CustomerEmail string  `json:"customerEmail"`
	CustomerPhone *string `json:"customerPhone,omitempty"`
	// Role is the requester's account role (customer/reseller/admin) so the
	// admin queue can distinguish reseller funding from customer top-ups.
	Role string `json:"role,omitempty"`
}

// adminHandler serves the admin top-up request queue.
type adminHandler struct {
	svc    *WalletService
	topups *TopUpRepo
	users  *mongo.Collection
	rec    audit.Recorder
}

// RegisterAdminRoutes mounts the admin top-up queue onto r (the /api/admin
// group, guarded by AdminOnly): paginated list, approve, reject. Decisions are
// recorded via rec; approval writes the mandatory topup ledger row.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder) {
	topups := NewTopUpRepo(db)
	a := &adminHandler{
		svc:    NewWalletService(NewMongoRepository(db), topups),
		topups: topups,
		users:  db.Collection("users"),
		rec:    rec,
	}

	r.Get("/wallet/topups", a.list)
	r.Post("/wallet/topups/{id}/approve", a.approve)
	r.Post("/wallet/topups/{id}/reject", a.reject)
}

// list handles GET /api/admin/wallet/topups — paginated, filterable by status,
// each request enriched with its customer contact.
func (a *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	p := pagination.ParseParams(r)
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	reqs, total, err := a.topups.List(r.Context(), status, p)
	if err != nil {
		response.InternalError(w)
		return
	}
	views, err := a.enrich(r.Context(), reqs)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OKWithMeta(w, views, pagination.CalcMeta(p, total))
}

// approve handles POST /api/admin/wallet/topups/{id}/approve.
func (a *adminHandler) approve(w http.ResponseWriter, r *http.Request) {
	id, ok := parseTopUpID(w, r)
	if !ok {
		return
	}
	req, err := a.svc.ApproveTopUpRequest(r.Context(), id, actorLabel(r))
	if err != nil {
		writeTopUpError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionTopupApprove,
		TargetType: "topup",
		TargetID:   id.Hex(),
		Summary: map[string]any{
			"amount": req.Amount, "channel": req.Channel, "userId": req.UserID.Hex(),
		},
	})
	response.OK(w, req)
}

// reject handles POST /api/admin/wallet/topups/{id}/reject.
func (a *adminHandler) reject(w http.ResponseWriter, r *http.Request) {
	id, ok := parseTopUpID(w, r)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	req, err := a.svc.RejectTopUpRequest(r.Context(), id, actorLabel(r), body.Reason)
	if err != nil {
		writeTopUpError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionTopupReject,
		TargetType: "topup",
		TargetID:   id.Hex(),
		Summary: map[string]any{
			"amount": req.Amount, "channel": req.Channel,
			"userId": req.UserID.Hex(), "reason": req.DecisionReason,
		},
	})
	response.OK(w, req)
}

// enrich resolves the customer contact for every request in one batched query.
func (a *adminHandler) enrich(ctx context.Context, reqs []*TopUpRequest) ([]adminTopUpView, error) {
	ids := make([]bson.ObjectID, 0, len(reqs))
	for _, req := range reqs {
		ids = append(ids, req.UserID)
	}
	byID := map[bson.ObjectID]struct {
		Email string
		Phone *string
		Role  string
	}{}
	if len(ids) > 0 {
		cur, err := a.users.Find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}})
		if err != nil {
			return nil, err
		}
		defer cur.Close(ctx)
		var rows []struct {
			ID    bson.ObjectID `bson:"_id"`
			Email string        `bson:"email"`
			Phone *string       `bson:"phone"`
			Role  string        `bson:"role"`
		}
		if err := cur.All(ctx, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			byID[row.ID] = struct {
				Email string
				Phone *string
				Role  string
			}{row.Email, row.Phone, row.Role}
		}
	}

	views := make([]adminTopUpView, len(reqs))
	for i, req := range reqs {
		views[i] = adminTopUpView{TopUpRequest: req}
		if u, ok := byID[req.UserID]; ok {
			views[i].CustomerEmail = u.Email
			views[i].CustomerPhone = u.Phone
			views[i].Role = u.Role
		}
	}
	return views, nil
}

// actorLabel returns a never-blank identifier for the acting admin (email, else
// phone, else user id) so top-up decisions by phone-only admins are attributed.
func actorLabel(r *http.Request) string {
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		return auth.ActorLabel(claims)
	}
	return ""
}

// parseTopUpID extracts the {id} path param as an ObjectID (404 on malformed).
func parseTopUpID(w http.ResponseWriter, r *http.Request) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return bson.ObjectID{}, false
	}
	return id, true
}

// writeTopUpError maps decision errors: missing request → 404, already
// decided → 409, validation → 400, ledger failure → 500 with its code.
func writeTopUpError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	case errors.Is(err, apperrors.ErrConflict):
		response.Error(w, http.StatusConflict, "ALREADY_DECIDED", "this request was already decided")
	case errors.As(err, &appErr) && errors.Is(err, apperrors.ErrBadRequest):
		response.Error(w, http.StatusBadRequest, appErr.Code, appErr.Message)
	case errors.Is(err, ErrLedgerWriteFailed):
		response.Error(w, http.StatusInternalServerError, "LEDGER_WRITE_FAILED", ErrLedgerWriteFailed.Message)
	default:
		response.InternalError(w)
	}
}
