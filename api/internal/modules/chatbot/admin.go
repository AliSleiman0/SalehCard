package chatbot

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

// RegisterAdminRoutes mounts the chatbot Q&A management on the /api/admin group.
// The caller wraps it in the "chatbot" RBAC domain (GET = chatbot.view,
// mutations = chatbot.manage). Every create / update / delete is audited.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder) {
	ensureIndexes(db)
	h := &adminHandler{repo: NewMongoRepository(db), rec: rec}
	r.Get("/chatbot", h.list)
	r.Post("/chatbot", h.create)
	r.Put("/chatbot/{id}", h.update)
	r.Delete("/chatbot/{id}", h.delete)
}

func ensureIndexes(db *mongo.Database) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := EnsureIndexes(ctx, db); err != nil {
		slog.Warn("chatbot: failed to ensure indexes", "error", err)
	}
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
	out, err := h.repo.List(r.Context(), false)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, out)
}

// checkParent enforces the one-level tree: a follow-up's parent must exist and
// be a top-level question, an entry cannot be its own parent, and an entry that
// already has follow-ups cannot itself become a follow-up.
func (h *adminHandler) checkParent(ctx context.Context, self *bson.ObjectID, parent *bson.ObjectID) (string, error) {
	if parent == nil {
		return "", nil
	}
	if self != nil && *self == *parent {
		return "a question cannot follow up on itself", errInvalid
	}
	p, err := h.repo.Get(ctx, *parent)
	if errors.Is(err, errNotFound) {
		return "the parent question does not exist", errInvalid
	}
	if err != nil {
		return "", err
	}
	if p.ParentID != nil {
		return "follow-ups can only be added under a top-level question", errInvalid
	}
	if self != nil {
		has, err := h.repo.HasChildren(ctx, *self)
		if err != nil {
			return "", err
		}
		if has {
			return "this question has its own follow-ups, so it must stay top-level", errInvalid
		}
	}
	return "", nil
}

func (h *adminHandler) decode(w http.ResponseWriter, r *http.Request, self *bson.ObjectID) (Input, *bson.ObjectID, bool) {
	var raw Input
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		response.BadRequest(w, "invalid request body")
		return raw, nil, false
	}
	in, parent, msg, err := raw.normalize()
	if err != nil {
		response.BadRequest(w, msg)
		return in, nil, false
	}
	if msg, err := h.checkParent(r.Context(), self, parent); err != nil {
		if errors.Is(err, errInvalid) {
			response.BadRequest(w, msg)
		} else {
			response.InternalError(w)
		}
		return in, nil, false
	}
	return in, parent, true
}

func (h *adminHandler) create(w http.ResponseWriter, r *http.Request) {
	in, parent, ok := h.decode(w, r, nil)
	if !ok {
		return
	}
	if n, err := h.repo.Count(r.Context()); err != nil {
		response.InternalError(w)
		return
	} else if n >= maxEntries {
		response.BadRequest(w, "the chatbot already has the maximum number of questions")
		return
	}
	now := time.Now().UTC()
	e := Entry{
		ID: bson.NewObjectID(), ParentID: parent,
		Question: in.Question, Answer: in.Answer, Keywords: in.Keywords,
		SortOrder: in.SortOrder, Enabled: in.Enabled,
		CreatedBy: actor(r), CreatedAt: now, UpdatedAt: now,
	}
	if err := h.repo.Create(r.Context(), &e); err != nil {
		response.InternalError(w)
		return
	}
	h.audit(r, audit.ActionChatbotCreate, &e, nil)
	response.JSON(w, http.StatusCreated, response.Response{Success: true, Data: e})
}

func (h *adminHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	cur, err := h.repo.Get(r.Context(), id)
	if errors.Is(err, errNotFound) {
		response.NotFound(w)
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}
	in, parent, ok := h.decode(w, r, &id)
	if !ok {
		return
	}
	cur.ParentID = parent
	cur.Question, cur.Answer, cur.Keywords = in.Question, in.Answer, in.Keywords
	cur.SortOrder, cur.Enabled = in.SortOrder, in.Enabled
	cur.UpdatedAt = time.Now().UTC()
	if err := h.repo.Replace(r.Context(), cur); err != nil {
		if errors.Is(err, errNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}
	h.audit(r, audit.ActionChatbotUpdate, cur, nil)
	response.OK(w, cur)
}

// delete removes a question together with its follow-ups (the console asks
// for confirmation first and says how many follow-ups go with it).
func (h *adminHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	gone, children, err := h.repo.DeleteWithChildren(r.Context(), id)
	if errors.Is(err, errNotFound) {
		response.NotFound(w)
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}
	h.audit(r, audit.ActionChatbotDelete, gone, map[string]any{"followUpsDeleted": children})
	response.OK(w, map[string]any{"deleted": true, "followUpsDeleted": children})
}

func (h *adminHandler) audit(r *http.Request, action string, e *Entry, extra map[string]any) {
	summary := map[string]any{"question": e.Question.En, "questionAr": e.Question.Ar, "enabled": e.Enabled}
	if e.ParentID != nil {
		summary["parentId"] = e.ParentID.Hex()
	}
	for k, v := range extra {
		summary[k] = v
	}
	h.rec.Record(r.Context(), audit.Entry{Action: action, TargetType: "chatbot", TargetID: e.ID.Hex(), Summary: summary})
}
