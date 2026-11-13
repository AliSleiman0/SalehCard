package chatbot

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
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
type PublicView struct {
	Entries []Entry `json:"entries"`
	Support Support `json:"support"`
}

type publicHandler struct {
	repo     Repository
	settings settings.Repository
}

// RegisterRoutes mounts the public, unauthenticated GET /api/v1/chatbot.
func RegisterRoutes(r chi.Router, db *mongo.Database) {
	ensureIndexes(db)
	h := &publicHandler{repo: NewMongoRepository(db), settings: settings.NewMongoRepository(db)}
	r.Get("/api/v1/chatbot", h.get)
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
	response.OK(w, view)
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
