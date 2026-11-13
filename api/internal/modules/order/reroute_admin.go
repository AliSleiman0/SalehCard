package order

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// RerouteProvider is one supplier an admin can send a stuck order to.
type RerouteProvider struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type rerouteHandler struct {
	settler   *SupplierSettler
	providers []RerouteProvider
}

// RegisterRerouteRoutes mounts the "send to another supplier" action on the
// admin orders group (the caller wraps it in the orders domain, so the POST
// needs orders.manage). options are the configured suppliers with display
// names; only those with a real adapter are offered.
func RegisterRerouteRoutes(r chi.Router, svc *OrderService, rec audit.Recorder, options []RerouteProvider, interval, giveUp time.Duration) {
	h := &rerouteHandler{settler: NewSupplierSettler(svc, rec, interval, giveUp)}
	for _, p := range options {
		if svc.providers.Has(p.ID) {
			h.providers = append(h.providers, p)
		}
	}
	r.Get("/orders/reroute-providers", h.listProviders)
	r.Post("/orders/{id}/reroute", h.reroute)
}

func (h *rerouteHandler) listProviders(w http.ResponseWriter, _ *http.Request) {
	out := h.providers
	if out == nil {
		out = []RerouteProvider{}
	}
	response.OK(w, out)
}

func (h *rerouteHandler) reroute(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	var in RerouteInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	claims, _ := auth.ClaimsFromContext(r.Context())
	o, err := h.settler.Reroute(r.Context(), id, in, auth.ActorLabel(claims))
	if err != nil {
		var ae *apperrors.AppError
		switch {
		case errors.Is(err, apperrors.ErrNotFound):
			response.NotFound(w)
		case errors.As(err, &ae) && errors.Is(err, apperrors.ErrConflict):
			response.Error(w, http.StatusConflict, ae.Code, ae.Message)
		case errors.As(err, &ae) && errors.Is(err, apperrors.ErrBadRequest):
			response.Error(w, http.StatusBadRequest, ae.Code, ae.Message)
		default:
			response.InternalError(w)
		}
		return
	}
	response.OK(w, o)
}
