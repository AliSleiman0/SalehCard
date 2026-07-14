package category

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// adminHandler serves the admin category CRUD. It owns the category service
// (wired with the product repository as its counter so counts + the delete
// product-guard work) and records mutations to the audit log.
type adminHandler struct {
	svc *CategoryService
	rec audit.Recorder
}

// RegisterAdminRoutes mounts the admin category management onto r (the
// /api/admin group, guarded by AdminOnly + the "categories" RBAC domain).
// Mutations are recorded via rec.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder) {
	_ = EnsureIndexes(context.Background(), db)
	svc := NewCategoryService(NewMongoRepository(db), product.NewMongoRepository(db))
	a := &adminHandler{svc: svc, rec: rec}

	r.Get("/categories", a.list)
	r.Post("/categories", a.create)
	r.Get("/categories/{id}", a.detail)
	r.Put("/categories/{id}", a.update)
	r.Delete("/categories/{id}", a.delete)
}

// createBody is the create payload. Visible defaults to true when omitted so a
// newly-created category is live unless the admin explicitly hides it.
type createBody struct {
	ParentID  *string    `json:"parentId"`
	Name      I18nString `json:"name"`
	Image     string     `json:"image"`
	SortOrder int        `json:"sortOrder"`
	Visible   *bool      `json:"visible"`
}

// updateBody is the partial-update payload; nil fields are left unchanged.
type updateBody struct {
	Name      *I18nString `json:"name"`
	Image     *string     `json:"image"`
	SortOrder *int        `json:"sortOrder"`
	Visible   *bool       `json:"visible"`
	ParentID  *string     `json:"parentId"` // move
}

// list handles GET /api/admin/categories — the full tree (all depths, incl.
// hidden) with has-children flags and rolled-up product counts.
func (a *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := a.svc.ListAll(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, items)
}

// detail handles GET /api/admin/categories/{id}.
func (a *adminHandler) detail(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCatID(w, r)
	if !ok {
		return
	}
	c, err := a.svc.Get(r.Context(), id)
	if err != nil {
		writeCatError(w, err)
		return
	}
	response.OK(w, c)
}

// create handles POST /api/admin/categories.
func (a *adminHandler) create(w http.ResponseWriter, r *http.Request) {
	var b createBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	visible := true
	if b.Visible != nil {
		visible = *b.Visible
	}
	c, err := a.svc.Create(r.Context(), CreateCategoryInput{
		ParentID:  b.ParentID,
		Name:      b.Name,
		Image:     b.Image,
		SortOrder: b.SortOrder,
		Visible:   visible,
	})
	if err != nil {
		writeCatError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionCategoryCreate,
		TargetType: "category",
		TargetID:   c.ID.Hex(),
		Summary:    map[string]any{"name": c.Name.En, "slug": c.Slug, "depth": c.Depth},
	})
	response.OK(w, c)
}

// update handles PUT /api/admin/categories/{id}.
func (a *adminHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCatID(w, r)
	if !ok {
		return
	}
	var b updateBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	c, err := a.svc.Update(r.Context(), id, UpdateCategoryInput(b))
	if err != nil {
		writeCatError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionCategoryUpdate,
		TargetType: "category",
		TargetID:   c.ID.Hex(),
		Summary:    map[string]any{"name": c.Name.En, "slug": c.Slug, "moved": b.ParentID != nil},
	})
	response.OK(w, c)
}

// delete handles DELETE /api/admin/categories/{id}. Blocked (400) when the node
// still has children or assigned products.
func (a *adminHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCatID(w, r)
	if !ok {
		return
	}
	if err := a.svc.Delete(r.Context(), id); err != nil {
		writeCatError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionCategoryDelete,
		TargetType: "category",
		TargetID:   id.Hex(),
	})
	response.OK(w, map[string]bool{"deleted": true})
}

// parseCatID extracts the {id} path param as an ObjectID, writing a 404 when
// malformed.
func parseCatID(w http.ResponseWriter, r *http.Request) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return bson.ObjectID{}, false
	}
	return id, true
}

// writeCatError maps service/repository errors to HTTP responses.
func writeCatError(w http.ResponseWriter, err error) {
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
