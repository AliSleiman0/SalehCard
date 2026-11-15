package chatbot

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

type adminHandler struct {
	knowledge KnowledgeRepository
	logs      LogRepository
	rec       audit.Recorder
}

// RegisterAdminRoutes mounts the AI help assistant's management on the
// /api/admin group: the knowledge text (GET/PUT /chatbot/knowledge) and the
// conversation log (GET /chatbot/logs). The caller wraps it in the "chatbot"
// RBAC domain (GET = chatbot.view, PUT = chatbot.manage); saves are audited.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder) {
	ensureIndexes(db)
	h := &adminHandler{knowledge: NewMongoKnowledgeRepository(db), logs: NewMongoLogRepository(db), rec: rec}
	r.Get("/chatbot/knowledge", h.getKnowledge)
	r.Put("/chatbot/knowledge", h.putKnowledge)
	r.Get("/chatbot/logs", h.logsList)
}

func ensureIndexes(db *mongo.Database) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := EnsureLogIndexes(ctx, db); err != nil {
		slog.Warn("chatbot: failed to ensure chat_logs indexes", "error", err)
	}
}

// knowledgeView is GET/PUT /api/admin/chatbot/knowledge.
type knowledgeView struct {
	Knowledge
	UsedChars int `json:"usedChars"`
	MaxChars  int `json:"maxChars"`
}

func viewOf(k *Knowledge) knowledgeView {
	if k == nil {
		k = &Knowledge{}
	}
	return knowledgeView{Knowledge: *k, MaxChars: maxKnowledgeChars,
		UsedChars: utf8.RuneCountInString(k.General) + utf8.RuneCountInString(k.AndroidOnly)}
}

func (h *adminHandler) getKnowledge(w http.ResponseWriter, r *http.Request) {
	k, err := h.knowledge.Get(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, viewOf(k))
}

func (h *adminHandler) putKnowledge(w http.ResponseWriter, r *http.Request) {
	var raw KnowledgeInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&raw); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	in, msg, err := raw.normalize()
	if err != nil {
		response.Error(w, http.StatusBadRequest, "KNOWLEDGE_TOO_LONG", msg)
		return
	}
	c, _ := auth.ClaimsFromContext(r.Context())
	k := &Knowledge{General: in.General, AndroidOnly: in.AndroidOnly,
		UpdatedBy: auth.ActorLabel(c), UpdatedAt: time.Now().UTC()}
	if err := h.knowledge.Save(r.Context(), k); err != nil {
		response.InternalError(w)
		return
	}
	v := viewOf(k)
	h.rec.Record(r.Context(), audit.Entry{Action: audit.ActionChatbotKnowledgeUpdate, TargetType: "chatbot_knowledge",
		TargetID: knowledgeID, Summary: map[string]any{
			"generalChars": utf8.RuneCountInString(k.General), "androidOnlyChars": utf8.RuneCountInString(k.AndroidOnly)}})
	response.OK(w, v)
}

const logsPageSize = 50

// logsList is GET /api/admin/chatbot/logs?filter=unanswered|flagged&page=N —
// the AI conversations, newest first (they expire after 30 days).
func (h *adminHandler) logsList(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter")
	if filter != "" && filter != logFilterFlagged && filter != logFilterUnanswered {
		response.BadRequest(w, "filter must be unanswered or flagged")
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	items, more, err := h.logs.List(r.Context(), filter, (page-1)*logsPageSize, logsPageSize)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]any{"items": items, "page": page, "hasMore": more})
}
