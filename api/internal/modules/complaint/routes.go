package complaint

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/blob"
	"github.com/AliSleiman0/salehcard/api/internal/platform/photoupload"
	"github.com/AliSleiman0/salehcard/api/internal/platform/ratelimit"
)

func ensureIndexes(db *mongo.Database) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := EnsureIndexes(ctx, db); err != nil {
		slog.Warn("complaint: failed to ensure indexes", "error", err)
	}
}

// RegisterRoutes mounts the customer complaint routes under /api/v1/complaints
// (AuthRequired + RequireActive). Creation is per-IP rate limited (on top of
// the per-user open cap) and photo uploads reuse the KYC upload limiter, since
// each accepted upload writes billed blob storage.
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config, store blob.Storage) {
	ensureIndexes(db)
	svc := NewService(NewMongoRepository(db), nil, photoupload.NewChecker(store, PhotoPrefix))
	h := &customerHandler{svc: svc}

	limiter := ratelimit.New(ratelimit.Config{Provider: cfg.RateLimitProvider}, db)
	if _, ok := limiter.(ratelimit.NoopLimiter); !ok {
		if err := ratelimit.EnsureIndexes(context.Background(), db); err != nil {
			slog.Warn("complaint: failed to ensure rate-limit indexes", "error", err)
		}
	}
	createLimit := ratelimit.Middleware(limiter, cfg.RateLimitComplaintMax, cfg.RateLimitComplaintWindow, func(req *http.Request) string {
		return "complaintcreate:" + ratelimit.ClientIP(req)
	})
	uploadLimit := ratelimit.Middleware(limiter, cfg.RateLimitKycUploadMax, cfg.RateLimitKycUploadWindow, func(req *http.Request) string {
		return "complaintupload:" + ratelimit.ClientIP(req)
	})

	r.Route("/api/v1/complaints", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Use(auth.RequireActive(db))
		mountCustomer(r, h, createLimit, uploadLimit, photoupload.Handler(store, PhotoPrefix, "complaint"))
	})
}

func mountCustomer(r chi.Router, h *customerHandler, createLimit, uploadLimit func(http.Handler) http.Handler, upload http.HandlerFunc) {
	r.With(createLimit).Post("/", h.create)
	r.Get("/", h.list)
	r.With(uploadLimit).Post("/photos", upload)
	r.Get("/{id}", h.get)
	r.Post("/{id}/messages", h.reply)
}

// RegisterAdminRoutes mounts the support-desk console routes on the /api/admin
// group. The caller wraps it in the "complaints" RBAC domain. Replies, notes,
// status and assignee changes are audited; staff replies and resolutions notify
// the customer through ntf (inbox row + push, never SMS).
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder, ntf notification.Notifier, store blob.Storage) {
	ensureIndexes(db)
	repo := NewMongoRepository(db)
	h := &adminHandler{svc: NewService(repo, ntf, photoupload.NewChecker(store, PhotoPrefix)), repo: repo, rec: rec}
	mountAdmin(r, h)
}

func mountAdmin(r chi.Router, h *adminHandler) {
	r.Get("/complaints", h.list)
	r.Get("/complaints/counts", h.counts)
	r.Get("/complaints/{id}", h.detail)
	r.Post("/complaints/{id}/messages", h.reply)
	r.Put("/complaints/{id}/status", h.setStatus)
	r.Put("/complaints/{id}/assignee", h.setAssignee)
}
