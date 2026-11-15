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
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
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
// names; only those with a real adapter are offered. probes (may be nil) is the
// shared supplier catalog/balance cache the pre-send checks read.
func RegisterRerouteRoutes(r chi.Router, svc *OrderService, probes *provider.Probes, rec audit.Recorder, options []RerouteProvider, interval, giveUp time.Duration) {
	h := &rerouteHandler{settler: NewSupplierSettler(svc, rec, interval, giveUp).WithProbes(probes)}
	for _, p := range options {
		if svc.providers.Has(p.ID) {
			h.providers = append(h.providers, p)
		}
	}
	r.Get("/orders/reroute-providers", h.listProviders)
	r.Get("/orders/{id}/supplier-options", h.supplierOptions)
	r.Post("/orders/{id}/reroute", h.reroute)
	r.Post("/orders/reroute-bulk", h.rerouteBulk)
}

func (h *rerouteHandler) providerName(id int) string {
	for _, p := range h.providers {
		if p.ID == id {
			return p.Name
		}
	}
	return ""
}

// supplierOptions handles GET /api/admin/orders/{id}/supplier-options — the
// order's product supplier links with this order's cost at each. Balances are
// shown only to admins who may see supplier accounts (suppliers.view).
func (h *rerouteHandler) supplierOptions(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	claims, _ := auth.ClaimsFromContext(r.Context())
	out, err := h.settler.SupplierOptions(r.Context(), id, claims.HasPerm("suppliers.view"))
	if err != nil {
		writeRerouteError(w, err)
		return
	}
	for i := range out.Options {
		out.Options[i].ProviderName = h.providerName(out.Options[i].Provider)
	}
	response.OK(w, out)
}

// rerouteBulk handles POST /api/admin/orders/reroute-bulk — re-send up to
// MaxBulkReroute orders, each to its product's first usable supplier link (or
// to its link at one chosen supplier). Per-order results, like refund-bulk.
func (h *rerouteHandler) rerouteBulk(w http.ResponseWriter, r *http.Request) {
	var in BulkRerouteInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	claims, _ := auth.ClaimsFromContext(r.Context())
	results, err := h.settler.BulkReroute(r.Context(), in, auth.ActorLabel(claims))
	if err != nil {
		writeRerouteError(w, err)
		return
	}
	sent := 0
	for _, res := range results {
		if res.Status == "sent" || res.Status == "completed" {
			sent++
		}
	}
	response.OK(w, map[string]any{"sent": sent, "total": len(results), "results": results})
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
		writeRerouteError(w, err)
		return
	}
	response.OK(w, o)
}

// writeRerouteError maps a reroute error to its HTTP status + typed code.
func writeRerouteError(w http.ResponseWriter, err error) {
	var ae *apperrors.AppError
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	case errors.As(err, &ae) && errors.Is(err, errServiceUnavailable):
		response.Error(w, http.StatusServiceUnavailable, ae.Code, ae.Message)
	case errors.As(err, &ae) && errors.Is(err, apperrors.ErrConflict):
		response.Error(w, http.StatusConflict, ae.Code, ae.Message)
	case errors.As(err, &ae) && errors.Is(err, apperrors.ErrBadRequest):
		response.Error(w, http.StatusBadRequest, ae.Code, ae.Message)
	default:
		response.InternalError(w)
	}
}
