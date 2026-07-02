package kyc

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// adminHandler serves the admin KYC moderation endpoints over the repository
// directly (no service layer — the KYC service exists for the customer path).
type adminHandler struct {
	repo Repository
	rec  audit.Recorder
}

// RegisterAdminRoutes mounts the admin KYC moderation queue onto r (the
// /api/admin group, guarded by AdminOnly): a paginated/filterable list, detail,
// and approve/reject (PUT). Decisions are recorded via rec.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder) {
	_ = EnsureIndexes(context.Background(), db)
	a := &adminHandler{
		repo: NewMongoRepository(db.Collection("kyc_submissions"), db.Collection("users")),
		rec:  rec,
	}

	r.Get("/kyc", a.list)
	r.Get("/kyc/{id}", a.detail)
	r.Put("/kyc/{id}", a.update)
}

// list handles GET /api/admin/kyc — paginated, newest first, with an optional
// status filter (pending/approved/rejected) and a name/document search (q).
func (a *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	p := pagination.ParseParams(r)
	q := r.URL.Query()
	f := Filter{
		Status: strings.TrimSpace(q.Get("status")),
		Search: strings.TrimSpace(q.Get("q")),
	}
	rows, total, err := a.repo.List(r.Context(), f, p)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OKWithMeta(w, rows, pagination.CalcMeta(p, total))
}

// detail handles GET /api/admin/kyc/{id}.
func (a *adminHandler) detail(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	s, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
		writeKycError(w, err)
		return
	}
	response.OK(w, s)
}

// statusBody is the moderation payload for PUT /api/admin/kyc/{id}.
type statusBody struct {
	Status          string `json:"status"`
	RejectionReason string `json:"rejectionReason"`
}

// update handles PUT /api/admin/kyc/{id} — approve or reject a submission (a
// rejection requires a reason). The reviewer is taken from the admin's JWT.
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
	reason := strings.TrimSpace(b.RejectionReason)
	if b.Status == StatusRejected && reason == "" {
		response.BadRequest(w, "a rejection reason is required")
		return
	}

	reviewedBy := ""
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		reviewedBy = claims.Email
	}

	s, err := a.repo.UpdateStatus(r.Context(), id, b.Status, reason, reviewedBy)
	if err != nil {
		writeKycError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionKYCDecision,
		TargetType: "kyc",
		TargetID:   id.Hex(),
		Summary:    map[string]any{"status": b.Status, "reason": reason, "userId": s.UserID.Hex()},
	})
	response.OK(w, s)
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
