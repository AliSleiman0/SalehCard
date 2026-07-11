package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
)

// fakeSettingsRepo returns a fixed Settings value (or an error).
type fakeSettingsRepo struct {
	s   *settings.Settings
	err error
}

func (f *fakeSettingsRepo) Get(context.Context) (*settings.Settings, error) { return f.s, f.err }
func (f *fakeSettingsRepo) Update(context.Context, settings.UpdateInput, string) (*settings.Settings, error) {
	return f.s, f.err
}

func TestLegalPages_RenderWithConfiguredEmail(t *testing.T) {
	p := newLegalPages(&fakeSettingsRepo{s: &settings.Settings{SupportEmail: "help@salehcard.example"}})

	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		marker  string
	}{
		{"privacy", p.privacy, "Privacy Policy"},
		{"delete-account", p.deleteAccount, "Delete your SalehCard account"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tc.handler(rr, httptest.NewRequest(http.MethodGet, "/"+tc.name, nil))

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rr.Code)
			}
			if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
				t.Fatalf("content-type = %q, want text/html", ct)
			}
			body := rr.Body.String()
			if !strings.Contains(body, tc.marker) {
				t.Errorf("body missing %q", tc.marker)
			}
			if !strings.Contains(body, "help@salehcard.example") {
				t.Error("body missing the configured support email")
			}
			if strings.Contains(body, fallbackSupportEmail) {
				t.Error("fallback email leaked despite a configured one")
			}
			// Both language sections present, Arabic rendered RTL.
			if !strings.Contains(body, `id="ar" lang="ar" dir="rtl"`) {
				t.Error("missing RTL Arabic section")
			}
			if !strings.Contains(body, legalLastUpdated) {
				t.Error("missing last-updated stamp")
			}
		})
	}
}

func TestLegalPages_FallbackEmail(t *testing.T) {
	// A settings read failure must never break a legal page — placeholder shows.
	p := newLegalPages(&fakeSettingsRepo{err: errors.New("mongo down")})

	rr := httptest.NewRecorder()
	p.privacy(rr, httptest.NewRequest(http.MethodGet, "/privacy", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), fallbackSupportEmail) {
		t.Error("fallback support email not rendered on settings failure")
	}
}

func TestLegalPages_EmptyEmailUsesFallback(t *testing.T) {
	p := newLegalPages(&fakeSettingsRepo{s: &settings.Settings{SupportEmail: "  "}})

	rr := httptest.NewRecorder()
	p.deleteAccount(rr, httptest.NewRequest(http.MethodGet, "/delete-account", nil))

	if !strings.Contains(rr.Body.String(), fallbackSupportEmail) {
		t.Error("whitespace-only support email must fall back to the placeholder")
	}
}
