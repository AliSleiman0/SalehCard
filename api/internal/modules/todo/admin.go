package todo

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

type adminHandler struct {
	repo Repository
	rec  audit.Recorder
}

// RegisterAdminRoutes mounts the to-do routes on the /api/admin group (behind
// AdminOnly; deliberately no RBAC domain — every admin uses the shared list).
// Deletes are recorded in the admin audit log.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := EnsureIndexes(ctx, db); err != nil {
		slog.Warn("todo: failed to ensure indexes", "error", err)
	}
	h := &adminHandler{repo: NewMongoRepository(db), rec: rec}
	r.Get("/todos", h.list)
	r.Post("/todos", h.create)
	r.Put("/todos/{id}", h.update)
	r.Delete("/todos/{id}", h.delete)
}

func actor(r *http.Request) string {
	c, _ := auth.ClaimsFromContext(r.Context())
	return auth.ActorLabel(c)
}

func parseID(w http.ResponseWriter, r *http.Request) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid id")
		return id, false
	}
	return id, true
}

func (h *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	out, err := h.repo.List(r.Context(), time.Now().UTC())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, out)
}

func (h *adminHandler) create(w http.ResponseWriter, r *http.Request) {
	var in Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if msg, err := in.validate(true); err != nil {
		response.BadRequest(w, msg)
		return
	}
	now := time.Now().UTC()
	t := Todo{ID: bson.NewObjectID(), Title: *in.Title, DueAt: in.DueAt, CreatedBy: actor(r), CreatedAt: now, UpdatedAt: now}
	if in.Note != nil {
		t.Note = *in.Note
	}
	if err := h.repo.Create(r.Context(), &t); err != nil {
		response.InternalError(w)
		return
	}
	response.JSON(w, http.StatusCreated, response.Response{Success: true, Data: t})
}

func (h *adminHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if msg, err := in.validate(false); err != nil {
		response.BadRequest(w, msg)
		return
	}
	out, err := h.repo.Update(r.Context(), id, buildUpdate(in, time.Now().UTC(), actor(r)))
	if errors.Is(err, errNotFound) {
		response.NotFound(w)
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, out)
}

// delete removes a task. The list is shared by every admin, so the deletion is
// recorded (who, and what the task said) in the admin audit log.
func (h *adminHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	gone, err := h.repo.Delete(r.Context(), id)
	if errors.Is(err, errNotFound) {
		response.NotFound(w)
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}
	h.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionTodoDelete,
		TargetType: "todo",
		TargetID:   id.Hex(),
		Summary:    map[string]any{"title": gone.Title, "done": gone.Done, "createdBy": gone.CreatedBy},
	})
	response.OK(w, map[string]bool{"deleted": true})
}
