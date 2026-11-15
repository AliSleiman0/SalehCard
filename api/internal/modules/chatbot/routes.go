package chatbot

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/ratelimit"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Support is the "still need help?" contact shown under the bot, taken from
// the admin Settings (Store and support).
type Support struct {
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Whatsapp string `json:"whatsapp,omitempty"`
}

// PublicView is GET /api/v1/chatbot: the enabled questions plus the support
// contact (bundled so the storefront needs no separate public settings call).
// AI reports whether free-typed questions may go to POST /chatbot/ask.
type PublicView struct {
	Entries []Entry `json:"entries"`
	Support Support `json:"support"`
	AI      bool    `json:"ai"`
}

type publicHandler struct {
	repo      Repository
	settings  settings.Repository
	assistant *Assistant
}

// RegisterRoutes mounts the public, unauthenticated GET /api/v1/chatbot and
// the authenticated AI routes POST /api/v1/chatbot/ask (+ /ask/{id}/report).
func RegisterRoutes(r chi.Router, db *mongo.Database, jwtSecret string, assistant *Assistant) {
	ensureIndexes(db)
	h := &publicHandler{repo: NewMongoRepository(db), settings: settings.NewMongoRepository(db), assistant: assistant}
	r.Get("/api/v1/chatbot", h.get)
	r.Group(func(g chi.Router) {
		g.Use(auth.AuthRequired(jwtSecret))
		g.Use(auth.RequireActive(db))
		g.Post("/api/v1/chatbot/ask", h.ask)
		g.Post("/api/v1/chatbot/ask/{id}/report", h.report)
	})
}

func (h *publicHandler) get(w http.ResponseWriter, r *http.Request) {
	all, err := h.repo.List(r.Context(), true)
	if err != nil {
		response.InternalError(w)
		return
	}
	view := PublicView{Entries: publicEntries(all)}
	if s, err := h.settings.Get(r.Context()); err == nil && s != nil {
		view.Support = Support{Email: s.SupportEmail, Phone: s.SupportPhone, Whatsapp: s.SupportWhatsapp}
	}
	if h.assistant != nil {
		view.AI, _ = h.assistant.Enabled(r.Context())
	}
	response.OK(w, view)
}

func (h *publicHandler) ask(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims.UserID == "" {
		response.Unauthorized(w, "authentication required")
		return
	}
	var in AskInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	res, msg, err := h.assistant.Ask(r.Context(), claims.UserID, ratelimit.ClientIP(r), in)
	switch {
	case err == nil:
		response.OK(w, res)
	case errors.Is(err, errInvalid):
		response.BadRequest(w, msg)
	case errors.Is(err, ErrAIDisabled):
		response.Error(w, http.StatusServiceUnavailable, "CHATBOT_AI_DISABLED", "the AI assistant is not available")
	case errors.Is(err, ErrAILimited):
		response.Error(w, http.StatusTooManyRequests, "CHATBOT_AI_LIMIT", "you've reached the AI assistant limit — please try again later")
	default:
		response.Error(w, http.StatusServiceUnavailable, "CHATBOT_AI_UNAVAILABLE", "the AI assistant is temporarily unavailable")
	}
}

func (h *publicHandler) report(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims.UserID == "" {
		response.Unauthorized(w, "authentication required")
		return
	}
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}
	if err := h.assistant.Report(r.Context(), claims.UserID, id); err != nil {
		if errors.Is(err, errNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]any{"reported": true})
}

// publicEntries keeps only what customers may see: enabled top-level questions
// and enabled follow-ups whose parent is itself enabled (all is already
// enabled-only). Internal fields (createdBy) are stripped.
func publicEntries(all []Entry) []Entry {
	top := map[bson.ObjectID]bool{}
	for _, e := range all {
		if e.ParentID == nil {
			top[e.ID] = true
		}
	}
	out := make([]Entry, 0, len(all))
	for _, e := range all {
		if e.ParentID != nil && !top[*e.ParentID] {
			continue
		}
		e.CreatedBy = ""
		out = append(out, e)
	}
	return out
}
