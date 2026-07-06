package bridge

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// AdminHandler serves the admin bridge console: device registration/monitoring
// and command history with retry/cancel.
type AdminHandler struct {
	svc   *Service
	store *MongoStore
	rec   audit.Recorder
}

// RegisterAdminRoutes mounts the admin bridge routes onto r (the /api/admin
// group, guarded by AdminOnly). It reuses the same store/service as the device
// routes over the same collections.
func RegisterAdminRoutes(r chi.Router, svc *Service, store *MongoStore, rec audit.Recorder) {
	a := &AdminHandler{svc: svc, store: store, rec: rec}
	r.Get("/bridge/devices", a.listDevices)
	r.Post("/bridge/devices", a.createDevice)
	r.Patch("/bridge/devices/{id}", a.updateDevice)
	r.Post("/bridge/devices/{id}/rotate-token", a.rotateToken)
	r.Post("/bridge/devices/{id}/check-balance", a.checkBalance)
	r.Delete("/bridge/devices/{id}", a.deleteDevice)
	r.Get("/bridge/commands", a.listCommands)
	r.Post("/bridge/commands/{id}/retry", a.retryCommand)
	r.Post("/bridge/commands/{id}/cancel", a.cancelCommand)
}

// deviceView is the admin device row (token hash never exposed; online derived
// client-side from lastSeenAt).
type deviceView struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Providers     []string   `json:"providers"`
	Enabled       bool       `json:"enabled"`
	LastSeenAt    *time.Time `json:"lastSeenAt,omitempty"`
	TouchBalance  *float64   `json:"touchBalance,omitempty"`
	TouchValidity string     `json:"touchValidity,omitempty"`
	AlfaBalance   *float64   `json:"alfaBalance,omitempty"`
	AlfaValidity  string     `json:"alfaValidity,omitempty"`
	AppVersion    string     `json:"appVersion,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}

func toDeviceView(d *Device) deviceView {
	return deviceView{
		ID: d.ID.Hex(), Name: d.Name, Providers: d.Providers, Enabled: d.Enabled,
		LastSeenAt: d.LastSeenAt, TouchBalance: d.TouchBalance, TouchValidity: d.TouchValidity,
		AlfaBalance: d.AlfaBalance, AlfaValidity: d.AlfaValidity, AppVersion: d.AppVersion,
		CreatedAt: d.CreatedAt,
	}
}

// commandView is the admin command row. The card code is masked to its last 4.
type commandView struct {
	ID              string    `json:"id"`
	OrderID         string    `json:"orderId,omitempty"`
	Provider        string    `json:"provider"`
	Type            string    `json:"type"`
	RecipientNumber string    `json:"recipientNumber,omitempty"`
	Amount          *float64  `json:"amount,omitempty"`
	CardCodeMasked  string    `json:"cardCodeMasked,omitempty"`
	Status          string    `json:"status"`
	Attempts        int       `json:"attempts"`
	StatusCode      *int      `json:"statusCode,omitempty"`
	RawReply        string    `json:"rawReply,omitempty"`
	FailReason      string    `json:"failReason,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func toCommandView(c *Command) commandView {
	v := commandView{
		ID: c.ID.Hex(), Provider: c.Provider, Type: c.Type, RecipientNumber: c.RecipientNumber,
		Amount: c.Amount, CardCodeMasked: maskCode(c.CardCode), Status: string(c.Status),
		Attempts: c.Attempts, FailReason: c.FailReason, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	if c.OrderID != nil {
		v.OrderID = c.OrderID.Hex()
	}
	if c.Result != nil {
		sc := c.Result.StatusCode
		v.StatusCode = &sc
		v.RawReply = c.Result.RawReply
	}
	return v
}

func maskCode(code string) string {
	if code == "" {
		return ""
	}
	if len(code) <= 4 {
		return "••••"
	}
	return "•••• " + code[len(code)-4:]
}

func (a *AdminHandler) listDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := a.store.ListDevices(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	out := make([]deviceView, len(devices))
	for i, d := range devices {
		out[i] = toDeviceView(d)
	}
	response.OK(w, out)
}

func (a *AdminHandler) createDevice(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string   `json:"name"`
		Providers []string `json:"providers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		response.BadRequest(w, "device name is required")
		return
	}
	providers := normalizeProviders(body.Providers)
	if len(providers) == 0 {
		response.BadRequest(w, "at least one provider (touch/alfa) is required")
		return
	}
	plain, hash, err := GenerateToken()
	if err != nil {
		response.InternalError(w)
		return
	}
	d := &Device{Name: name, TokenHash: hash, Providers: providers, Enabled: true}
	if err := a.store.CreateDevice(r.Context(), d); err != nil {
		response.InternalError(w)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{Action: audit.ActionBridgeDeviceCreate, TargetType: "bridge_device", TargetID: d.ID.Hex(), Summary: map[string]any{"name": name, "providers": providers}})
	// The plaintext token is returned exactly once.
	response.OK(w, map[string]any{"device": toDeviceView(d), "token": plain})
}

func (a *AdminHandler) updateDevice(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	var body struct {
		Name      *string  `json:"name,omitempty"`
		Enabled   *bool    `json:"enabled,omitempty"`
		Providers []string `json:"providers,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	set := bson.D{}
	summary := map[string]any{}
	if body.Name != nil {
		set = append(set, bson.E{Key: "name", Value: strings.TrimSpace(*body.Name)})
		summary["name"] = strings.TrimSpace(*body.Name)
	}
	if body.Enabled != nil {
		set = append(set, bson.E{Key: "enabled", Value: *body.Enabled})
		summary["enabled"] = *body.Enabled
	}
	if body.Providers != nil {
		providers := normalizeProviders(body.Providers)
		if len(providers) == 0 {
			response.BadRequest(w, "at least one provider (touch/alfa) is required")
			return
		}
		set = append(set, bson.E{Key: "providers", Value: providers})
		summary["providers"] = providers
	}
	if len(set) == 0 {
		response.BadRequest(w, "nothing to update")
		return
	}
	if err := a.store.UpdateDevice(r.Context(), id, set); err != nil {
		a.writeStoreError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{Action: audit.ActionBridgeDeviceUpdate, TargetType: "bridge_device", TargetID: id.Hex(), Summary: summary})
	d, err := a.store.FindDeviceByID(r.Context(), id)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, toDeviceView(d))
}

func (a *AdminHandler) rotateToken(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	plain, hash, err := GenerateToken()
	if err != nil {
		response.InternalError(w)
		return
	}
	if err := a.store.UpdateDevice(r.Context(), id, bson.D{{Key: "tokenHash", Value: hash}}); err != nil {
		a.writeStoreError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{Action: audit.ActionBridgeTokenRotate, TargetType: "bridge_device", TargetID: id.Hex()})
	response.OK(w, map[string]any{"token": plain})
}

func (a *AdminHandler) checkBalance(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	d, err := a.store.FindDeviceByID(r.Context(), id)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}
	if err := a.svc.EnqueueBalanceCheck(r.Context(), d); err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]any{"queued": len(d.Providers)})
}

func (a *AdminHandler) deleteDevice(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	if err := a.store.DeleteDevice(r.Context(), id); err != nil {
		a.writeStoreError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{Action: audit.ActionBridgeDeviceDelete, TargetType: "bridge_device", TargetID: id.Hex()})
	response.OK(w, map[string]any{"deleted": true})
}

func (a *AdminHandler) listCommands(w http.ResponseWriter, r *http.Request) {
	p := pagination.ParseParams(r)
	q := r.URL.Query()
	f := CommandFilter{Status: q.Get("status"), Provider: q.Get("provider")}
	if oid := strings.TrimSpace(q.Get("orderId")); oid != "" {
		if id, err := bson.ObjectIDFromHex(oid); err == nil {
			f.OrderID = &id
		}
	}
	cmds, total, err := a.store.ListCommands(r.Context(), f, p)
	if err != nil {
		response.InternalError(w)
		return
	}
	out := make([]commandView, len(cmds))
	for i, c := range cmds {
		out[i] = toCommandView(c)
	}
	response.OKWithMeta(w, out, pagination.CalcMeta(p, total))
}

func (a *AdminHandler) retryCommand(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	c, err := a.store.RetryCommand(r.Context(), id)
	if err != nil {
		a.writeConflictError(w, err, "only failed or cancelled commands can be retried")
		return
	}
	a.rec.Record(r.Context(), audit.Entry{Action: audit.ActionBridgeCommandRetry, TargetType: "bridge_command", TargetID: id.Hex()})
	response.OK(w, toCommandView(c))
}

func (a *AdminHandler) cancelCommand(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	c, err := a.store.CancelCommand(r.Context(), id)
	if err != nil {
		a.writeConflictError(w, err, "only queued or leased commands can be cancelled")
		return
	}
	a.rec.Record(r.Context(), audit.Entry{Action: audit.ActionBridgeCommandCancel, TargetType: "bridge_command", TargetID: id.Hex()})
	response.OK(w, toCommandView(c))
}

func (a *AdminHandler) writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, apperrors.ErrNotFound) || err == mongo.ErrNoDocuments {
		response.NotFound(w)
		return
	}
	response.InternalError(w)
}

func (a *AdminHandler) writeConflictError(w http.ResponseWriter, err error, conflictMsg string) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	case errors.Is(err, apperrors.ErrConflict):
		response.Error(w, http.StatusConflict, "INVALID_STATUS", conflictMsg)
	default:
		response.InternalError(w)
	}
}

// normalizeProviders lowercases, dedupes, and keeps only supported operators.
func normalizeProviders(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 2)
	for _, p := range in {
		p = strings.ToLower(strings.TrimSpace(p))
		if (p == "touch" || p == "alfa") && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}
