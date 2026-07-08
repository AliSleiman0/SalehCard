package role

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
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// adminHandler serves the admin role endpoints.
type adminHandler struct {
	repo Repository
	rec  audit.Recorder
}

// RegisterAdminRoutes mounts role management onto r (the /api/admin group,
// guarded by AdminOnly). Reads (list + permission catalog) are open to every
// admin — the console needs role names to render assignments — while all
// mutations require a super admin, so a users.manage role can never mint
// itself more authority.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder) {
	_ = EnsureIndexes(context.Background(), db)
	a := &adminHandler{repo: NewMongoRepository(db), rec: rec}

	r.Get("/roles", a.list)
	r.Get("/roles/permissions", a.catalog)
	r.Group(func(g chi.Router) {
		g.Use(auth.RequireSuperAdmin())
		g.Post("/roles", a.create)
		g.Put("/roles/{id}", a.update)
		g.Delete("/roles/{id}", a.delete)
	})
}

// list handles GET /api/admin/roles — all custom roles.
func (a *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	roles, err := a.repo.List(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, roles)
}

// catalog handles GET /api/admin/roles/permissions — the grantable permission
// domains, for the role editor's checkbox grid.
func (a *adminHandler) catalog(w http.ResponseWriter, _ *http.Request) {
	response.OK(w, map[string]any{"domains": Domains})
}

// roleBody is the create/update request payload.
type roleBody struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// validate normalizes and checks the payload, returning the clean permission
// set. It writes the error response itself when invalid.
func (b *roleBody) validate(w http.ResponseWriter) ([]string, bool) {
	b.Name = strings.TrimSpace(b.Name)
	b.Description = strings.TrimSpace(b.Description)
	if b.Name == "" || len(b.Name) > 60 {
		response.BadRequest(w, "name is required (max 60 characters)")
		return nil, false
	}
	perms, ok := NormalizePermissions(b.Permissions)
	if !ok {
		response.BadRequest(w, "permissions contain an unknown entry")
		return nil, false
	}
	if len(perms) == 0 {
		response.BadRequest(w, "at least one permission is required")
		return nil, false
	}
	return perms, true
}

// create handles POST /api/admin/roles (super admin only).
func (a *adminHandler) create(w http.ResponseWriter, r *http.Request) {
	var b roleBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	perms, ok := b.validate(w)
	if !ok {
		return
	}
	role := &Role{Name: b.Name, Description: b.Description, Permissions: perms}
	if err := a.repo.Create(r.Context(), role); err != nil {
		writeRoleError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionRoleCreate,
		TargetType: "role",
		TargetID:   role.ID.Hex(),
		Summary:    map[string]any{"name": role.Name, "permissions": role.Permissions},
	})
	response.OK(w, role)
}

// update handles PUT /api/admin/roles/{id} (super admin only). Permission
// changes take effect for holders on their next token refresh.
func (a *adminHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var b roleBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	perms, ok := b.validate(w)
	if !ok {
		return
	}
	role, err := a.repo.Update(r.Context(), id, b.Name, b.Description, perms)
	if err != nil {
		writeRoleError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionRoleUpdate,
		TargetType: "role",
		TargetID:   role.ID.Hex(),
		Summary:    map[string]any{"name": role.Name, "permissions": role.Permissions},
	})
	response.OK(w, role)
}

// delete handles DELETE /api/admin/roles/{id} (super admin only). A role still
// assigned to any user cannot be deleted — reassign those admins first.
func (a *adminHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	role, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
		writeRoleError(w, err)
		return
	}
	// Count-then-delete is not atomic: a concurrent role assignment could slip a
	// user onto this role between the count and the delete, leaving a dangling
	// adminRoleId (resolvePerms then fails closed — the admin logs in with no
	// permissions until reassigned). Accepted, like the last-super-admin guard:
	// role management is a low-concurrency, super-admin-only surface, and the
	// no-multi-document-transactions model makes a fully-atomic guard costly.
	assigned, err := a.repo.CountAssigned(r.Context(), id)
	if err != nil {
		response.InternalError(w)
		return
	}
	if assigned > 0 {
		response.Error(w, http.StatusConflict, "ROLE_IN_USE",
			"role is assigned to one or more admins — reassign them first")
		return
	}
	if err := a.repo.Delete(r.Context(), id); err != nil {
		writeRoleError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionRoleDelete,
		TargetType: "role",
		TargetID:   id.Hex(),
		Summary:    map[string]any{"name": role.Name},
	})
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

// writeRoleError maps repository errors to HTTP responses. A duplicate-key
// insert/update (unique name index) surfaces as 409 ROLE_NAME_TAKEN.
func writeRoleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	case mongo.IsDuplicateKeyError(err):
		response.Error(w, http.StatusConflict, "ROLE_NAME_TAKEN", "a role with this name already exists")
	default:
		response.InternalError(w)
	}
}
