package supplier

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/supplieralert"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// AdminHandler serves the /api/admin/suppliers routes.
type AdminHandler struct {
	svc    *Service
	job    *SyncJob
	alerts *supplieralert.Repo
	rec    audit.Recorder
}

// RegisterAdminRoutes wires the supplier admin routes onto r (the caller supplies
// the AdminOnly + domain("suppliers") group, so GETs need suppliers.view and the
// rest suppliers.manage). svc reads through the shared provider.Probes cache;
// job is the sync job the "Sync now" button runs one pass of; alerts is the
// supplier alerts store.
func RegisterAdminRoutes(r chi.Router, svc *Service, job *SyncJob, alerts *supplieralert.Repo, rec audit.Recorder) {
	h := &AdminHandler{svc: svc, job: job, alerts: alerts, rec: rec}
	r.Get("/suppliers", h.list)
	r.Get("/suppliers/sync-status", h.syncStatus)
	r.Get("/suppliers/alerts", h.listAlerts)
	r.Post("/suppliers/alerts/ack-all", h.ackAllAlerts)
	r.Post("/suppliers/alerts/{alertId}/ack", h.ackAlert)
	r.Get("/suppliers/{id}/catalog", h.catalog)
	r.Post("/suppliers/{id}/sync", h.sync)
	r.Post("/suppliers/{id}/import", h.importProducts)
	r.Put("/suppliers/{id}/settings", h.updateSettings)
	r.Get("/suppliers/{id}/orders", h.orders)
}

func (h *AdminHandler) list(w http.ResponseWriter, r *http.Request) {
	views, err := h.svc.List(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]any{"suppliers": views})
}

// supplierID parses and validates the {id} path param against the configured
// suppliers, writing a 404 and returning ok=false when unknown.
func (h *AdminHandler) supplierID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid supplier id")
		return 0, false
	}
	if _, ok := h.svc.supplierByID(id); !ok {
		response.NotFound(w)
		return 0, false
	}
	return id, true
}

func (h *AdminHandler) catalog(w http.ResponseWriter, r *http.Request) {
	id, ok := h.supplierID(w, r)
	if !ok {
		return
	}
	items, err := h.svc.Catalog(r.Context(), id)
	if err != nil {
		h.probeError(w, err)
		return
	}
	response.OK(w, map[string]any{"products": items})
}

// sync runs one sync pass over the products linked to this supplier (the
// same pass the scheduled job runs, honouring SUPPLIER_SYNC_DRYRUN).
func (h *AdminHandler) sync(w http.ResponseWriter, r *http.Request) {
	id, ok := h.supplierID(w, r)
	if !ok {
		return
	}
	if h.job == nil {
		response.Error(w, http.StatusServiceUnavailable, "SUPPLIER_SYNC_UNAVAILABLE", "supplier sync is not configured")
		return
	}
	rep, err := h.job.Pass(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrSupplierCatalog) {
			response.Error(w, http.StatusBadGateway, "SUPPLIER_UNAVAILABLE",
				"couldn't load this supplier's product list just now, so nothing was changed — try again in a minute")
			return
		}
		response.InternalError(w)
		return
	}
	h.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionSupplierSync,
		TargetType: "supplier",
		TargetID:   strconv.Itoa(id),
		Summary: map[string]any{"checked": rep.Checked, "hidden": rep.Hidden, "shown": rep.Shown, "paused": rep.Paused,
			"recovered": rep.Recovered, "promoted": rep.Promoted, "costChanges": rep.CostChanges, "dryRun": rep.DryRun},
	})
	response.OK(w, rep)
}

func (h *AdminHandler) syncStatus(w http.ResponseWriter, r *http.Request) {
	if h.job == nil {
		response.OK(w, SyncStatus{})
		return
	}
	response.OK(w, h.job.Status())
}

func (h *AdminHandler) listAlerts(w http.ResponseWriter, r *http.Request) {
	open := r.URL.Query().Get("status") != "all"
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	rows, err := h.alerts.List(r.Context(), open, limit)
	if err != nil {
		response.InternalError(w)
		return
	}
	n, err := h.alerts.CountOpen(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]any{"alerts": rows, "open": n})
}

func (h *AdminHandler) ackAlert(w http.ResponseWriter, r *http.Request) {
	oid, err := bson.ObjectIDFromHex(chi.URLParam(r, "alertId"))
	if err != nil {
		response.BadRequest(w, "invalid alert id")
		return
	}
	a, err := h.alerts.Ack(r.Context(), oid, actorEmail(r))
	if errors.Is(err, supplieralert.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "ALERT_NOT_OPEN", "this alert was already acknowledged")
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}
	h.rec.Record(r.Context(), audit.Entry{
		Action: audit.ActionSupplierAlertAck, TargetType: "supplier_alert", TargetID: a.ID.Hex(),
		Summary: map[string]any{"kind": a.Kind, "key": a.Key},
	})
	response.OK(w, a)
}

func (h *AdminHandler) ackAllAlerts(w http.ResponseWriter, r *http.Request) {
	n, err := h.alerts.AckAll(r.Context(), actorEmail(r))
	if err != nil {
		response.InternalError(w)
		return
	}
	h.rec.Record(r.Context(), audit.Entry{
		Action: audit.ActionSupplierAlertAck, TargetType: "supplier_alert", TargetID: "all",
		Summary: map[string]any{"count": n},
	})
	response.OK(w, map[string]any{"acknowledged": n})
}

func (h *AdminHandler) importProducts(w http.ResponseWriter, r *http.Request) {
	id, ok := h.supplierID(w, r)
	if !ok {
		return
	}
	var in ImportInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if len(in.Items) == 0 {
		response.BadRequest(w, "no items selected")
		return
	}
	res, err := h.svc.Import(r.Context(), id, in)
	if err != nil {
		h.probeError(w, err)
		return
	}
	h.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionSupplierImport,
		TargetType: "supplier",
		TargetID:   strconv.Itoa(id),
		Summary:    map[string]any{"created": res.Created, "skipped": len(res.Skipped)},
	})
	response.OK(w, res)
}

func (h *AdminHandler) updateSettings(w http.ResponseWriter, r *http.Request) {
	id, ok := h.supplierID(w, r)
	if !ok {
		return
	}
	var in SettingsInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if in.LowBalanceThreshold != nil && *in.LowBalanceThreshold < 0 {
		response.BadRequest(w, "low-balance threshold must be non-negative")
		return
	}
	if in.MarkupPercent != nil && *in.MarkupPercent < 0 {
		response.BadRequest(w, "markup must be non-negative")
		return
	}
	if in.FxToUSD != nil && (*in.FxToUSD < 0 || math.IsNaN(*in.FxToUSD) || math.IsInf(*in.FxToUSD, 0)) {
		response.BadRequest(w, "exchange rate must be a non-negative number (0 clears it)")
		return
	}
	set, err := h.svc.UpdateSettings(r.Context(), id, in, actorEmail(r))
	if err != nil {
		response.InternalError(w)
		return
	}
	h.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionSupplierSettingsUpdate,
		TargetType: "supplier",
		TargetID:   strconv.Itoa(id),
		Summary:    map[string]any{"lowBalanceThreshold": set.LowBalanceThreshold, "markupPercent": set.MarkupPercent, "fxToUsd": set.FxToUSD},
	})
	response.OK(w, set)
}

func (h *AdminHandler) orders(w http.ResponseWriter, r *http.Request) {
	id, ok := h.supplierID(w, r)
	if !ok {
		return
	}
	p := pagination.ParseParams(r)
	rows, total, err := h.svc.RecentOrders(r.Context(), id, p)
	if err != nil {
		response.InternalError(w)
		return
	}
	if rows == nil {
		rows = []RecentOrderView{}
	}
	response.OKWithMeta(w, map[string]any{"orders": rows}, pagination.CalcMeta(p, total))
}

// probeError maps a Cataloger probe failure to an HTTP response: a
// not-a-Cataloger supplier (stub) → 400; anything else (auth/IP/unreachable) →
// 502 so the UI can show "supplier unavailable".
func (h *AdminHandler) probeError(w http.ResponseWriter, err error) {
	if errors.Is(err, provider.ErrNotImplemented) {
		response.Error(w, http.StatusBadRequest, "SUPPLIER_NOT_PROBEABLE", "this supplier does not expose a catalog/balance API")
		return
	}
	response.Error(w, http.StatusBadGateway, "SUPPLIER_UNAVAILABLE", err.Error())
}

// actorEmail returns the acting admin's email from the JWT claims ("" if absent).
func actorEmail(r *http.Request) string {
	if c, ok := auth.ClaimsFromContext(r.Context()); ok {
		return c.Email
	}
	return ""
}
