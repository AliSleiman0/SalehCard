package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
)

type nopRecorder struct{}

func (nopRecorder) Record(_ context.Context, _ audit.Entry) {}

// fakeSettingsRepo is an in-memory Repository for handler tests.
type fakeSettingsRepo struct {
	cur     *Settings
	updated bool
}

func (f *fakeSettingsRepo) Get(_ context.Context) (*Settings, error) {
	if f.cur == nil {
		return defaults(), nil
	}
	return f.cur, nil
}

func (f *fakeSettingsRepo) Update(_ context.Context, in UpdateInput, _ string) (*Settings, error) {
	f.updated = true
	s := defaults()
	if in.AdminSmsTwoFactorEnabled != nil {
		s.AdminSmsTwoFactorEnabled = *in.AdminSmsTwoFactorEnabled
	}
	f.cur = s
	return s, nil
}

func doUpdate(t *testing.T, h *adminHandler, phone string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", bytes.NewReader(buf))
	req = req.WithContext(auth.ContextWithClaims(req.Context(),
		&auth.Claims{UserID: "abc", Email: "admin@x.com", Phone: phone, Role: "admin"}))
	rec := httptest.NewRecorder()
	h.update(rec, req)
	return rec
}

// Enabling admin SMS 2FA is refused when the acting admin has no phone (would
// lock them out); with a phone it succeeds and persists.
func TestUpdate_Enable2FA_RequiresActorPhone(t *testing.T) {
	repo := &fakeSettingsRepo{}
	h := &adminHandler{repo: repo, rec: nopRecorder{}, smsProvider: "monty", pushProvider: "log"}

	rec := doUpdate(t, h, "", map[string]any{"adminSmsTwoFactorEnabled": true})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("no-phone enable: want 400, got %d", rec.Code)
	}
	if repo.updated {
		t.Fatalf("settings must not be written when the guardrail trips")
	}

	rec = doUpdate(t, h, "+96170123456", map[string]any{"adminSmsTwoFactorEnabled": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("with-phone enable: want 200, got %d", rec.Code)
	}
	if repo.cur == nil || !repo.cur.AdminSmsTwoFactorEnabled {
		t.Fatalf("flag was not persisted")
	}
}

// Turning 2FA off never requires a phone.
func TestUpdate_Disable2FA_NoPhoneNeeded(t *testing.T) {
	repo := &fakeSettingsRepo{}
	h := &adminHandler{repo: repo, rec: nopRecorder{}, smsProvider: "monty", pushProvider: "log"}

	rec := doUpdate(t, h, "", map[string]any{"adminSmsTwoFactorEnabled": false})
	if rec.Code != http.StatusOK {
		t.Fatalf("disable: want 200, got %d", rec.Code)
	}
}
