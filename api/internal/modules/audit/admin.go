package audit

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// adminHandler serves the read side of the audit log.
type adminHandler struct {
	repo *MongoRepository
}

// RegisterAdminRoutes mounts the audit-log listing onto r (the /api/admin
// group, guarded by AdminOnly).
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	_ = EnsureIndexes(context.Background(), db)
	a := &adminHandler{repo: NewMongoRepository(db.Collection("admin_audit_log"))}

	r.Get("/audit-log", a.list)
}

// list handles GET /api/admin/audit-log — paginated, filterable by action,
// target type/id, actor, and time range.
func (a *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := Filter{
		Action:     strings.TrimSpace(q.Get("action")),
		TargetType: strings.TrimSpace(q.Get("targetType")),
		TargetID:   strings.TrimSpace(q.Get("targetId")),
		ActorID:    strings.TrimSpace(q.Get("actorId")),
	}
	if raw := strings.TrimSpace(q.Get("from")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			f.From = &t
		}
	}
	if raw := strings.TrimSpace(q.Get("to")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			f.To = &t
		}
	}

	p := pagination.ParseParams(r)
	entries, total, err := a.repo.List(r.Context(), f, p)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OKWithMeta(w, entries, pagination.CalcMeta(p, total))
}
