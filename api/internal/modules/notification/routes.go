package notification

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
)

// RegisterRoutes wires the customer notification routes onto r (guarded by
// AuthRequired): the inbox listing, the unread count for the bell badge, the
// read-all marker, and push device-token registration.
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config) {
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("notification: failed to ensure indexes", "error", err)
	}
	h := NewHandler(NewMongoRepository(db.Collection("notifications"), db.Collection("device_tokens")))

	r.Route("/api/v1/notifications", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Get("/", h.List)
		r.Get("/unread-count", h.UnreadCount)
		r.Post("/read-all", h.ReadAll)
		r.Post("/devices", h.RegisterDevice)
		r.Delete("/devices", h.UnregisterDevice)
	})
}
