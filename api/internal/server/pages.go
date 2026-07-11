package server

import (
	"context"
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
)

// Public legal pages (Play Store requirements): GET /privacy and
// GET /delete-account serve embedded bilingual (EN+AR) HTML documents. The
// support contact email is injected at request time from the app_settings
// singleton so admins manage it from the console without a redeploy.

//go:embed static/*.html
var pagesFS embed.FS

// fallbackSupportEmail is rendered when app_settings.supportEmail is unset or
// unreadable. PLACEHOLDER — ops must set the real address in the admin console
// (Settings → Store & support); a legal page must never 500 over a DB hiccup.
const fallbackSupportEmail = "support@salehcard.com"

// legalLastUpdated is the "Last updated" date rendered on both pages. Bump it
// whenever the page content changes materially.
const legalLastUpdated = "2026-07-11"

// legalPages renders the embedded legal documents with live settings data.
type legalPages struct {
	tmpl     *template.Template
	settings settings.Repository
}

// newLegalPages parses the embedded page templates. It panics on a parse
// error — a boot-time authoring bug, caught before any deploy finishes
// (fail-fast, like the admin domain() wiring below).
func newLegalPages(repo settings.Repository) *legalPages {
	return &legalPages{
		tmpl:     template.Must(template.ParseFS(pagesFS, "static/*.html")),
		settings: repo,
	}
}

func (p *legalPages) privacy(w http.ResponseWriter, r *http.Request) {
	p.render(w, r, "privacy.html")
}

func (p *legalPages) deleteAccount(w http.ResponseWriter, r *http.Request) {
	p.render(w, r, "delete-account.html")
}

// render executes the named page template with the current support email. A
// settings read failure falls back to the placeholder — never an error page.
func (p *legalPages) render(w http.ResponseWriter, r *http.Request, name string) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	email := fallbackSupportEmail
	if s, err := p.settings.Get(ctx); err != nil {
		slog.Warn("server: legal page settings read failed — using fallback support email", "page", name, "error", err)
	} else if v := strings.TrimSpace(s.SupportEmail); v != "" {
		email = v
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := p.tmpl.ExecuteTemplate(w, name, struct {
		SupportEmail string
		LastUpdated  string
	}{SupportEmail: email, LastUpdated: legalLastUpdated}); err != nil {
		// Headers are already out; log — the partial page is still readable.
		slog.Error("server: legal page render failed", "page", name, "error", err)
	}
}
