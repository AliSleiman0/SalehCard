package notification

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/platform/push"
)

// pushTimeout bounds the background fan-out so goroutines never leak.
const pushTimeout = 15 * time.Second

// Note is one customer-facing event to deliver: an inbox row plus a best-effort
// push to every device the user has registered.
type Note struct {
	Kind  string
	Title string // English fallback; also the push-tray copy
	Body  string
	Data  map[string]string
}

// Notifier is the write side of the inbox, passed into the business modules.
// Notify is best-effort and called AFTER the action succeeds: failures are
// logged, never propagated — a notification outage must not block orders,
// top-ups, or KYC decisions (same contract as audit.Recorder).
type Notifier interface {
	Notify(ctx context.Context, userID bson.ObjectID, n Note)
}

// Nop is a no-op Notifier for tests.
type Nop struct{}

// Notify does nothing.
func (Nop) Notify(context.Context, bson.ObjectID, Note) {}

// notifier is the Notifier implementation backed by MongoRepository + a push
// Sender.
type notifier struct {
	repo   Repository
	sender push.Sender
}

// NewNotifier returns the Notifier the server wires into order, wallet, and kyc.
func NewNotifier(db *mongo.Database, sender push.Sender) Notifier {
	return &notifier{
		repo:   NewMongoRepository(db.Collection("notifications"), db.Collection("device_tokens")),
		sender: sender,
	}
}

// Notify inserts the inbox row synchronously, then fans the push out to the
// user's devices in the background (detached from the request context so the
// HTTP response never waits on the provider). Stale tokens reported by the
// provider are pruned.
func (nt *notifier) Notify(ctx context.Context, userID bson.ObjectID, n Note) {
	row := &Notification{
		UserID:    userID,
		Kind:      n.Kind,
		Title:     n.Title,
		Body:      n.Body,
		Data:      n.Data,
		CreatedAt: time.Now().UTC(),
	}
	if err := nt.repo.Insert(ctx, row); err != nil {
		slog.Error("notification: failed to record", "kind", n.Kind, "userId", userID.Hex(), "error", err)
	}

	bg := context.WithoutCancel(ctx)
	go func() {
		pushCtx, cancel := context.WithTimeout(bg, pushTimeout)
		defer cancel()
		nt.fanOut(pushCtx, userID, n)
	}()
}

// fanOut pushes n to every device token of userID, pruning tokens the provider
// reports as unregistered.
func (nt *notifier) fanOut(ctx context.Context, userID bson.ObjectID, n Note) {
	tokens, err := nt.repo.TokensForUser(ctx, userID)
	if err != nil {
		slog.Warn("notification: failed to load device tokens", "userId", userID.Hex(), "error", err)
		return
	}
	for _, token := range tokens {
		err := nt.sender.Send(ctx, push.Message{Token: token, Title: n.Title, Body: n.Body, Data: n.Data})
		switch {
		case errors.Is(err, push.ErrUnregistered):
			if delErr := nt.repo.DeleteTokenValue(ctx, token); delErr != nil {
				slog.Warn("notification: failed to prune stale token", "error", delErr)
			}
		case err != nil:
			slog.Warn("notification: push send failed", "kind", n.Kind, "userId", userID.Hex(), "error", err)
		}
	}
}
