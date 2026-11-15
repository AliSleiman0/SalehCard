package chatbot

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// kbList is GET /api/admin/chatbot/kb: every doc plus the budget meter.
type kbList struct {
	Docs      []KBDoc `json:"docs"`
	UsedChars int     `json:"usedChars"`
	MaxChars  int     `json:"maxChars"`
	MaxDocs   int     `json:"maxDocs"`
}

func (h *adminHandler) kbList(w http.ResponseWriter, r *http.Request) {
	docs, err := h.kb.List(r.Context(), false)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, kbList{Docs: docs, UsedChars: kbUsedChars(docs, nil), MaxChars: maxKBChars, MaxDocs: maxKBDocs})
}

// kbDecode parses + validates the body and checks the doc count and the
// knowledge-base budget against the other docs (self = the doc being
// replaced, nil on create).
func (h *adminHandler) kbDecode(w http.ResponseWriter, r *http.Request, self *bson.ObjectID) (KBInput, bool) {
	var raw KBInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&raw); err != nil {
		response.BadRequest(w, "invalid request body")
		return raw, false
	}
	in, msg, err := raw.normalize()
	if err != nil {
		response.BadRequest(w, msg)
		return in, false
	}
	docs, err := h.kb.List(r.Context(), false)
	if err != nil {
		response.InternalError(w)
		return in, false
	}
	if self == nil && len(docs) >= maxKBDocs {
		response.BadRequest(w, "the knowledge base already has the maximum number of articles")
		return in, false
	}
	if in.Enabled {
		d := KBDoc{Title: in.Title, Body: in.Body, Enabled: true}
		if kbUsedChars(docs, self)+d.chars() > maxKBChars {
			response.Error(w, http.StatusBadRequest, "KB_BUDGET",
				"the enabled articles would exceed the knowledge-base size limit — shorten or disable another article")
			return in, false
		}
	}
	return in, true
}

func (h *adminHandler) kbCreate(w http.ResponseWriter, r *http.Request) {
	in, ok := h.kbDecode(w, r, nil)
	if !ok {
		return
	}
	now := time.Now().UTC()
	who := actor(r)
	d := KBDoc{
		ID: bson.NewObjectID(), Slug: in.Slug, Title: in.Title, Body: in.Body, Tags: in.Tags,
		SortOrder: in.SortOrder, Enabled: in.Enabled,
		CreatedBy: who, UpdatedBy: who, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.kb.Create(r.Context(), &d); err != nil {
		h.kbWriteErr(w, err)
		return
	}
	h.kbAudit(r, audit.ActionChatbotKBCreate, &d)
	response.JSON(w, http.StatusCreated, response.Response{Success: true, Data: d})
}

func (h *adminHandler) kbUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	cur, err := h.kb.Get(r.Context(), id)
	if errors.Is(err, errNotFound) {
		response.NotFound(w)
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}
	in, ok := h.kbDecode(w, r, &id)
	if !ok {
		return
	}
	cur.Slug, cur.Title, cur.Body, cur.Tags = in.Slug, in.Title, in.Body, in.Tags
	cur.SortOrder, cur.Enabled = in.SortOrder, in.Enabled
	cur.UpdatedBy, cur.UpdatedAt = actor(r), time.Now().UTC()
	if err := h.kb.Replace(r.Context(), cur); err != nil {
		h.kbWriteErr(w, err)
		return
	}
	h.kbAudit(r, audit.ActionChatbotKBUpdate, cur)
	response.OK(w, cur)
}

func (h *adminHandler) kbDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	gone, err := h.kb.Delete(r.Context(), id)
	if errors.Is(err, errNotFound) {
		response.NotFound(w)
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}
	h.kbAudit(r, audit.ActionChatbotKBDelete, gone)
	response.OK(w, map[string]any{"deleted": true})
}

func (h *adminHandler) kbWriteErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errDuplicateSlug):
		response.BadRequest(w, "another article already uses this key")
	case errors.Is(err, errNotFound):
		response.NotFound(w)
	default:
		response.InternalError(w)
	}
}

func (h *adminHandler) kbAudit(r *http.Request, action string, d *KBDoc) {
	h.rec.Record(r.Context(), audit.Entry{Action: action, TargetType: "kb_doc", TargetID: d.ID.Hex(),
		Summary: map[string]any{"title": d.Title.En, "titleAr": d.Title.Ar, "slug": d.Slug, "enabled": d.Enabled}})
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
