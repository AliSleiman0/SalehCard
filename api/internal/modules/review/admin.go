package review

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// adminHandler serves the admin review-moderation endpoints over the repository
// directly (no service layer — the review service exists for the customer
// submission path).
type adminHandler struct {
	repo Repository
}

// RegisterAdminRoutes mounts the admin review moderation queue onto r (the
// /api/admin group, guarded by AdminOnly): a paginated/filterable list,
// approve/reject (PUT), and delete. Approving/rejecting/deleting recomputes the
// affected product's denormalized rating.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	_ = EnsureIndexes(context.Background(), db)
	a := &adminHandler{repo: NewMongoRepository(db.Collection("reviews"), db.Collection("products"))}

	r.Get("/reviews", a.list)
	r.Put("/reviews/{id}", a.update)
	r.Delete("/reviews/{id}", a.delete)
}

// list handles GET /api/admin/reviews — paginated, newest first, with an
// optional status filter (pending/approved/rejected) and a body search (q).
func (a *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	p := pagination.ParseParams(r)
	q := r.URL.Query()
	f := ReviewFilter{
		Status: strings.TrimSpace(q.Get("status")),
		Search: strings.TrimSpace(q.Get("q")),
	}
	if pid := strings.TrimSpace(q.Get("productId")); pid != "" {
		if oid, err := bson.ObjectIDFromHex(pid); err == nil {
			f.ProductID = &oid
		}
	}
	rows, total, err := a.repo.List(r.Context(), f, p)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OKWithMeta(w, rows, pagination.CalcMeta(p, total))
}

// statusBody is the moderation payload for PUT /api/admin/reviews/{id}.
type statusBody struct {
	Status string `json:"status"`
}

// update handles PUT /api/admin/reviews/{id} — set a review's moderation status
// (approve/reject, or back to pending), then recompute the product's rating.
func (a *adminHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var b statusBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	switch b.Status {
	case StatusPending, StatusApproved, StatusRejected:
	default:
		response.BadRequest(w, "status must be pending, approved, or rejected")
		return
	}

	rv, err := a.repo.UpdateStatus(r.Context(), id, b.Status)
	if err != nil {
		writeReviewError(w, err)
		return
	}
	if err := a.repo.RecomputeProductRating(r.Context(), rv.ProductID); err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, rv)
}

// delete handles DELETE /api/admin/reviews/{id} — remove a review, then
// recompute the product's rating.
func (a *adminHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	rv, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
		writeReviewError(w, err)
		return
	}
	if err := a.repo.Delete(r.Context(), id); err != nil {
		writeReviewError(w, err)
		return
	}
	if err := a.repo.RecomputeProductRating(r.Context(), rv.ProductID); err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]bool{"deleted": true})
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
