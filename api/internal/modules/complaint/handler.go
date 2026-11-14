package complaint

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// customerHandler serves /api/v1/complaints (AuthRequired + RequireActive).
// Every response goes through Complaint.ForCustomer, which strips internal
// notes and staff identities.
type customerHandler struct{ svc *Service }

// maxBodyBytes bounds JSON request bodies (a 2000-char message is well under).
const maxBodyBytes = 64 << 10

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		response.BadRequest(w, "invalid request body")
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return id, false
	}
	return id, true
}

func (h *customerHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	var in CreateInput
	if !decodeJSON(w, r, &in) {
		return
	}
	c, err := h.svc.Create(r.Context(), userID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, response.Response{Success: true, Data: c.ForCustomer()})
}

func (h *customerHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	p := pagination.ParseParams(r)
	rows, total, err := h.svc.ListMine(r.Context(), userID, p)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]CustomerView, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ForCustomer())
	}
	response.OKWithMeta(w, out, pagination.CalcMeta(p, total))
}

func (h *customerHandler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	c, err := h.svc.GetMine(r.Context(), userID, id)
	if err != nil {
		writeError(w, err)
		return
	}
	response.OK(w, c.ForCustomer())
}

func (h *customerHandler) reply(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in ReplyInput
	if !decodeJSON(w, r, &in) {
		return
	}
	c, err := h.svc.CustomerReply(r.Context(), userID, id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	response.OK(w, c.ForCustomer())
}
