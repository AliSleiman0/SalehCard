package offer

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

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// adminHandler serves the admin offer endpoints over the repository directly,
// enriching each offer with its product summary + computed was/now prices so the
// admin table can render without a second round-trip.
type adminHandler struct {
	repo     Repository
	products product.Service
}

// offerView is an offer plus its derived status and (when resolvable) the
// product summary with the original/offer prices.
type offerView struct {
	*Offer
	Status            string          `json:"status"`
	Product           *productSummary `json:"product,omitempty"`
	OriginalFromPrice float64         `json:"originalFromPrice"`
	OfferFromPrice    float64         `json:"offerFromPrice"`
}

// statusOf derives an offer's display status. Expiry takes precedence over the
// active/paused flag; a not-yet-started active offer reads "scheduled".
func statusOf(o *Offer, now time.Time) string {
	switch {
	case o.EndsAt != nil && now.After(*o.EndsAt):
		return "expired"
	case !o.Active:
		return "paused"
	case o.StartsAt != nil && now.Before(*o.StartsAt):
		return "scheduled"
	default:
		return "active"
	}
}

// view builds an offerView, resolving the product (best-effort: a deleted product
// just leaves Product nil and prices zero).
func (a *adminHandler) view(ctx context.Context, o *Offer) offerView {
	v := offerView{Offer: o, Status: statusOf(o, time.Now().UTC())}
	if p, err := a.products.FindByID(ctx, o.ProductID.Hex()); err == nil {
		s := summarize(p)
		v.Product = &s
		v.OriginalFromPrice = s.FromPrice
		v.OfferFromPrice = OfferPriceFor(o, s.FromPrice)
	}
	return v
}

// RegisterAdminRoutes mounts the admin offer CRUD onto r (the /api/admin group,
// guarded by AdminOnly): paginated/filterable list, create, detail, update,
// delete.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	_ = EnsureIndexes(context.Background(), db)
	a := &adminHandler{
		repo:     NewMongoRepository(db.Collection("offers")),
		products: product.NewProductService(product.NewMongoRepository(db)),
	}

	r.Get("/offers", a.list)
	r.Post("/offers", a.create)
	r.Get("/offers/{id}", a.detail)
	r.Put("/offers/{id}", a.update)
	r.Delete("/offers/{id}", a.delete)
}

// list handles GET /api/admin/offers — paginated, with optional status and
// productId filters.
func (a *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	p := pagination.ParseParams(r)
	q := r.URL.Query()
	f := OfferFilter{
		Status:    strings.TrimSpace(q.Get("status")),
		ProductID: strings.TrimSpace(q.Get("productId")),
	}
	offers, total, err := a.repo.List(r.Context(), f, p)
	if err != nil {
		response.InternalError(w)
		return
	}
	views := make([]offerView, len(offers))
	for i, o := range offers {
		views[i] = a.view(r.Context(), o)
	}
	response.OKWithMeta(w, views, pagination.CalcMeta(p, total))
}

// detail handles GET /api/admin/offers/{id}.
func (a *adminHandler) detail(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	o, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
		writeOfferError(w, err)
		return
	}
	response.OK(w, a.view(r.Context(), o))
}

// create handles POST /api/admin/offers.
func (a *adminHandler) create(w http.ResponseWriter, r *http.Request) {
	var b offerBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if err := b.validate(); err != nil {
		writeOfferError(w, err)
		return
	}
	o := b.toOffer()
	if err := a.repo.Create(r.Context(), o); err != nil {
		writeOfferError(w, err)
		return
	}
	response.OK(w, a.view(r.Context(), o))
}

// update handles PUT /api/admin/offers/{id}.
func (a *adminHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var b offerBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if err := b.validate(); err != nil {
		writeOfferError(w, err)
		return
	}
	o, err := a.repo.Update(r.Context(), id, b.toUpdate())
	if err != nil {
		writeOfferError(w, err)
		return
	}
	response.OK(w, a.view(r.Context(), o))
}

// delete handles DELETE /api/admin/offers/{id}.
func (a *adminHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := a.repo.Delete(r.Context(), id); err != nil {
		writeOfferError(w, err)
		return
	}
	response.OK(w, map[string]bool{"deleted": true})
}

// offerBody is the create/update request payload.
type offerBody struct {
	ProductID     string       `json:"productId"`
	DiscountType  DiscountType `json:"discountType"`
	DiscountValue float64      `json:"discountValue"`
	StartsAt      *time.Time   `json:"startsAt"`
	EndsAt        *time.Time   `json:"endsAt"`
	Active        bool         `json:"active"`
	SortOrder     int          `json:"sortOrder"`
}

// validate enforces the field constraints shared by create and update.
func (b offerBody) validate() error {
	if strings.TrimSpace(b.ProductID) == "" {
		return badRequest("productId is required")
	}
	if _, err := bson.ObjectIDFromHex(b.ProductID); err != nil {
		return badRequest("productId is invalid")
	}
	switch b.DiscountType {
	case DiscountPercent, DiscountFixed:
	default:
		return badRequest("discountType must be percent or fixed")
	}
	if b.DiscountValue <= 0 {
		return badRequest("discountValue must be greater than zero")
	}
	if b.DiscountType == DiscountPercent && b.DiscountValue > 100 {
		return badRequest("percent value must be between 0 and 100")
	}
	if b.SortOrder < 0 {
		return badRequest("sortOrder must not be negative")
	}
	if b.StartsAt != nil && b.EndsAt != nil && b.EndsAt.Before(*b.StartsAt) {
		return badRequest("endsAt must be after startsAt")
	}
	return nil
}

// toOffer builds a new Offer from the create payload.
func (b offerBody) toOffer() *Offer {
	pid, _ := bson.ObjectIDFromHex(b.ProductID)
	return &Offer{
		ProductID:     pid,
		DiscountType:  b.DiscountType,
		DiscountValue: b.DiscountValue,
		StartsAt:      b.StartsAt,
		EndsAt:        b.EndsAt,
		Active:        b.Active,
		SortOrder:     b.SortOrder,
	}
}

// toUpdate builds the editable-field set from the update payload.
func (b offerBody) toUpdate() OfferUpdate {
	pid, _ := bson.ObjectIDFromHex(b.ProductID)
	return OfferUpdate{
		ProductID:     pid,
		DiscountType:  b.DiscountType,
		DiscountValue: b.DiscountValue,
		StartsAt:      b.StartsAt,
		EndsAt:        b.EndsAt,
		Active:        b.Active,
		SortOrder:     b.SortOrder,
	}
}

// badRequest builds a 400-class AppError with the BAD_REQUEST machine code.
func badRequest(msg string) error {
	return &apperrors.AppError{Code: "BAD_REQUEST", Message: msg, Err: apperrors.ErrBadRequest}
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

// writeOfferError maps repository/validation errors to HTTP responses.
func writeOfferError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
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
