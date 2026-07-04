// Package settings holds the platform's editable business configuration
// (store details, default language/currency, low-stock threshold, feature
// toggles) as a single document, plus a read-only, secret-free view of which
// external providers are configured. Provider credentials live in env/config,
// never in this collection.
package settings

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// adminHandler serves the admin settings endpoints. The provider names are
// captured at construction from runtime config so the integration status can be
// reported without touching secrets.
type adminHandler struct {
	repo         Repository
	rec          audit.Recorder
	smsProvider  string
	pushProvider string
}

// RegisterAdminRoutes wires GET/PUT /api/admin/settings onto r (the /api/admin
// group, guarded by AdminOnly). Updates are recorded to the audit log via rec.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder, smsProvider, pushProvider string) {
	a := &adminHandler{
		repo:         NewMongoRepository(db),
		rec:          rec,
		smsProvider:  smsProvider,
		pushProvider: pushProvider,
	}
	r.Get("/settings", a.get)
	r.Put("/settings", a.update)
}

// get handles GET /api/admin/settings.
func (a *adminHandler) get(w http.ResponseWriter, r *http.Request) {
	s, err := a.repo.Get(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, settingsView{Settings: s, Integrations: integrationsView(a.smsProvider, a.pushProvider)})
}

// update handles PUT /api/admin/settings — validates the editable fields, upserts
// the singleton, and records an audit entry.
func (a *adminHandler) update(w http.ResponseWriter, r *http.Request) {
	var in UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if in.DefaultLanguage != nil {
		switch *in.DefaultLanguage {
		case "en", "ar", "tr":
		default:
			response.BadRequest(w, "defaultLanguage must be one of en, ar, tr")
			return
		}
	}
	if in.DefaultCurrency != nil {
		switch *in.DefaultCurrency {
		case "USD", "TRY":
		default:
			response.BadRequest(w, "defaultCurrency must be one of USD, TRY")
			return
		}
	}
	if in.LowStockThreshold != nil && *in.LowStockThreshold < 0 {
		response.BadRequest(w, "lowStockThreshold must be zero or greater")
		return
	}
	if in.LoyaltyEarnUsdPerPoint != nil && *in.LoyaltyEarnUsdPerPoint <= 0 {
		response.BadRequest(w, "loyaltyEarnUsdPerPoint must be greater than zero")
		return
	}

	actor := ""
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		actor = auth.ActorLabel(claims)
	}

	s, err := a.repo.Update(r.Context(), in, actor)
	if err != nil {
		response.InternalError(w)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionSettingsUpdate,
		TargetType: "settings",
		TargetID:   settingsID,
		Summary:    map[string]any{"updatedBy": actor},
	})
	response.OK(w, settingsView{Settings: s, Integrations: integrationsView(a.smsProvider, a.pushProvider)})
}
