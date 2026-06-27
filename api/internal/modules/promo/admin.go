package promo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// adminHandler serves the admin promo endpoints over the repository directly
// (no service layer — the promo service exists only for the checkout path).
type adminHandler struct {
	repo Repository
}

// promoView is a promo plus its derived lifecycle status for the admin list.
type promoView struct {
	*PromoCode
	Status string `json:"status"`
}

// statusOf derives a promo's display status. Expiry and depletion take
// precedence over the active/paused flag (an expired code reads "expired" even
// if still flagged active).
func statusOf(p *PromoCode) string {
	now := time.Now().UTC()
	switch {
	case p.ExpiresAt != nil && now.After(*p.ExpiresAt):
		return "expired"
	case p.MaxUses > 0 && p.Uses >= p.MaxUses:
		return "depleted"
	case !p.Active:
		return "paused"
	default:
		return "active"
	}
}

// RegisterAdminRoutes mounts the admin promo CRUD onto r (the /api/admin group,
// guarded by AdminOnly): paginated/filterable list, create, detail, update,
// delete.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	_ = EnsureIndexes(context.Background(), db)
	a := &adminHandler{repo: NewMongoRepository(db.Collection("promos"))}

	r.Get("/promos", a.list)
	r.Post("/promos", a.create)
	r.Get("/promos/{id}", a.detail)
	r.Put("/promos/{id}", a.update)
	r.Delete("/promos/{id}", a.delete)
}

// list handles GET /api/admin/promos — paginated, newest first, with optional
// type / status filters and a code search (q).
func (a *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	p := pagination.ParseParams(r)
	q := r.URL.Query()
	f := PromoFilter{
		Type:   strings.TrimSpace(q.Get("type")),
		Status: strings.TrimSpace(q.Get("status")),
		Search: strings.TrimSpace(q.Get("q")),
	}
	promos, total, err := a.repo.List(r.Context(), f, p)
	if err != nil {
		response.InternalError(w)
		return
	}
	views := make([]promoView, len(promos))
	for i, pr := range promos {
		views[i] = promoView{PromoCode: pr, Status: statusOf(pr)}
	}
	response.OKWithMeta(w, views, pagination.CalcMeta(p, total))
}

// detail handles GET /api/admin/promos/{id}.
func (a *adminHandler) detail(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	p, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
		writePromoError(w, err)
		return
	}
	response.OK(w, promoView{PromoCode: p, Status: statusOf(p)})
}

// create handles POST /api/admin/promos.
func (a *adminHandler) create(w http.ResponseWriter, r *http.Request) {
	var b promoBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if err := b.validate(); err != nil {
		writePromoError(w, err)
		return
	}
	p := b.toPromo()
	if err := a.repo.Create(r.Context(), p); err != nil {
		writePromoError(w, err)
		return
	}
	response.OK(w, promoView{PromoCode: p, Status: statusOf(p)})
}

// update handles PUT /api/admin/promos/{id}.
func (a *adminHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var b promoBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if err := b.validate(); err != nil {
		writePromoError(w, err)
		return
	}
	p, err := a.repo.Update(r.Context(), id, PromoUpdate{
		Code:      b.Code,
		Type:      b.Type,
		Value:     b.Value,
		MinOrder:  b.MinOrder,
		MaxUses:   b.MaxUses,
		StartsAt:  b.StartsAt,
		ExpiresAt: b.ExpiresAt,
		Active:    b.Active,
	})
	if err != nil {
		writePromoError(w, err)
		return
	}
	response.OK(w, promoView{PromoCode: p, Status: statusOf(p)})
}

// delete handles DELETE /api/admin/promos/{id}.
func (a *adminHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := a.repo.Delete(r.Context(), id); err != nil {
		writePromoError(w, err)
		return
	}
	response.OK(w, map[string]bool{"deleted": true})
}

// promoBody is the create/update request payload.
type promoBody struct {
	Code      string     `json:"code"`
	Type      PromoType  `json:"type"`
	Value     float64    `json:"value"`
	MinOrder  float64    `json:"minOrder"`
	MaxUses   int        `json:"maxUses"`
	StartsAt  *time.Time `json:"startsAt"`
	ExpiresAt *time.Time `json:"expiresAt"`
	Active    bool       `json:"active"`
}

// validate enforces the field constraints shared by create and update.
func (b promoBody) validate() error {
	if strings.TrimSpace(b.Code) == "" {
		return badRequest("BAD_REQUEST", "code is required")
	}
	switch b.Type {
	case PromoTypePercent, PromoTypeFixed, PromoTypeCashback:
	default:
		return badRequest("BAD_REQUEST", "type must be percent, fixed, or cashback")
	}
	if b.Value <= 0 {
		return badRequest("BAD_REQUEST", "value must be greater than zero")
	}
	if b.Type == PromoTypePercent && b.Value > 100 {
		return badRequest("BAD_REQUEST", "percent value must be between 0 and 100")
	}
	if b.MinOrder < 0 {
		return badRequest("BAD_REQUEST", "minOrder must not be negative")
	}
	if b.MaxUses < 0 {
		return badRequest("BAD_REQUEST", "maxUses must not be negative")
	}
	if b.StartsAt != nil && b.ExpiresAt != nil && b.ExpiresAt.Before(*b.StartsAt) {
		return badRequest("BAD_REQUEST", "expiresAt must be after startsAt")
	}
	return nil
}

// toPromo builds a new PromoCode from the create payload (uses start at 0).
func (b promoBody) toPromo() *PromoCode {
	return &PromoCode{
		Code:      b.Code,
		Type:      b.Type,
		Value:     b.Value,
		MinOrder:  b.MinOrder,
		MaxUses:   b.MaxUses,
		StartsAt:  b.StartsAt,
		ExpiresAt: b.ExpiresAt,
		Active:    b.Active,
	}
}

// parseID extracts the {id} path param as an ObjectID, writing a 404 and
// returning ok=false when it is malformed.
func parseID(w http.ResponseWriter, r *http.Request) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return bson.ObjectID{}, false
	}
	return id, true
}

// writePromoError maps repository/validation errors to HTTP responses (404
// missing, 409 duplicate code, 400 bad-request, 500 otherwise). Shared by the
// admin handlers and the customer validate handler.
func writePromoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	case errors.Is(err, apperrors.ErrConflict):
		response.Error(w, http.StatusConflict, "CONFLICT", "a promo with this code already exists")
	case errors.Is(err, apperrors.ErrBadRequest):
		msg := err.Error()
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			msg = appErr.Message
		}
		response.BadRequest(w, msg)
	default:
		response.InternalError(w)
	}
}
