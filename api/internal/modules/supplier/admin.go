package supplier

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// AdminHandler serves the /api/admin/suppliers routes.
type AdminHandler struct {
	svc *Service
	rec audit.Recorder
}

// RegisterAdminRoutes wires the supplier admin routes onto r (the caller supplies
// the AdminOnly + domain("suppliers") group). The provider registry is the one
// built in order.RegisterRoutes (orderSvc.Providers()); products is a product
// service (wired WithCategoryResolver so imported products can be category-assigned).
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder, registry *provider.Registry, suppliers []config.SupplierConfig, products product.Service) {
	h := &AdminHandler{svc: NewService(db, registry, suppliers, products), rec: rec}
	r.Get("/suppliers", h.list)
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

func (h *AdminHandler) sync(w http.ResponseWriter, r *http.Request) {
	id, ok := h.supplierID(w, r)
	if !ok {
		return
	}
	res, err := h.svc.Sync(r.Context(), id)
	if err != nil {
		h.probeError(w, err)
		return
	}
	h.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionSupplierSync,
		TargetType: "supplier",
		TargetID:   strconv.Itoa(id),
		Summary:    map[string]any{"checked": res.Checked, "updated": res.Updated, "unavailable": res.Unavailable, "drift": len(res.Drift)},
	})
	response.OK(w, res)
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
	set, err := h.svc.UpdateSettings(r.Context(), id, in, actorEmail(r))
	if err != nil {
		response.InternalError(w)
		return
	}
	h.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionSupplierSettingsUpdate,
		TargetType: "supplier",
		TargetID:   strconv.Itoa(id),
		Summary:    map[string]any{"lowBalanceThreshold": set.LowBalanceThreshold, "markupPercent": set.MarkupPercent},
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
