package auth

import (
	"log/slog"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Account statuses that may no longer act. These mirror the user module's
// Status constants (user.StatusSuspended / user.StatusDeleted) — duplicated
// here because user imports wallet/notification, so this platform middleware
// cannot import user without a cycle. The strings are persisted contract
// values and never change.
const (
	statusSuspended = "suspended"
	statusDeleted   = "deleted"
)

// RequireActive returns middleware for AuthRequired customer route groups that
// refuses state-changing requests from suspended or soft-deleted accounts.
// Access JWTs are stateless and long-lived (ACCESS_TOKEN_TTL, 90d default), so
// a token can outlive an account deletion by months — without this check a
// deleted customer could keep filing top-ups, KYC submissions, or reviews.
//
// Scope: only mutating methods pay the (indexed _id, status-projection) read;
// GET/HEAD pass through — reads against an anonymized account leak nothing.
// Suspended keeps its explicit 403 ACCOUNT_SUSPENDED; deleted reads as a plain
// 401, indistinguishable from an expired token (deletion is never revealed to
// a lingering token holder). Fails open on a DB error (defense-in-depth layer,
// mirrors the rate limiter) and passes through when no claims are attached
// (AuthRequired upstream already rejected those).
func RequireActive(db *mongo.Database) func(http.Handler) http.Handler {
	col := db.Collection("users")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}
			id, ok := UserIDFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			var doc struct {
				Status string `bson:"status"`
			}
			err := col.FindOne(r.Context(),
				bson.D{{Key: "_id", Value: id}},
				options.FindOne().SetProjection(bson.D{{Key: "status", Value: 1}}),
			).Decode(&doc)
			switch {
			case err == mongo.ErrNoDocuments:
				response.Unauthorized(w, "invalid or expired token")
				return
			case err != nil:
				slog.Warn("auth: active-account check failed — allowing request", "userId", id.Hex(), "error", err)
			case doc.Status == statusSuspended:
				response.Error(w, http.StatusForbidden, "ACCOUNT_SUSPENDED", "this account has been suspended")
				return
			case doc.Status == statusDeleted:
				response.Unauthorized(w, "invalid or expired token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
