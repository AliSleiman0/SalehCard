package order

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/bridge"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// bridgeDispatcher is the slice of the bridge module the order engine needs to
// enqueue a recharge command. nil (or Enabled()==false) makes fulfillBridge fall
// back to parking the order for manual completion — today's behavior. The reverse
// direction (the bridge completing/flagging an order on a device result) is the
// bridge.OrderSettler interface, which *OrderService implements below.
type bridgeDispatcher interface {
	Enabled() bool
	DispatchOrder(ctx context.Context, in bridge.DispatchInput) error
}

// CompleteBridgeOrder completes a processing order whose recharge the device
// confirmed. It drives the exact same effects as an admin manual completion
// (guarded processing→completed transition + transferRef + customer notify +
// loyalty award), and is idempotent: an order that already moved on returns nil.
// Implements bridge.OrderSettler.
func (s *OrderService) CompleteBridgeOrder(ctx context.Context, orderID bson.ObjectID, transferRef string) error {
	var extra bson.D
	if transferRef != "" {
		extra = bson.D{{Key: "fulfillment.transferRef", Value: transferRef}}
	}
	before, err := s.repo.TransitionStatus(ctx, orderID,
		[]OrderStatus{OrderStatusProcessing}, OrderStatusCompleted,
		TimelineEvent{Status: "completed", Note: "mobile recharge delivered by bridge device", At: time.Now().UTC()},
		extra,
	)
	if err != nil {
		// Already completed/refunded/failed, or missing → nothing to do.
		if errors.Is(err, apperrors.ErrConflict) || errors.Is(err, apperrors.ErrNotFound) {
			return nil
		}
		return err
	}
	s.ntf.Notify(ctx, before.UserID, orderCompletedNote(before.ID.Hex(), before.Total, before.Currency))
	if s.loyalty != nil {
		s.loyalty.Award(ctx, before.UserID, before.Total)
	}
	return nil
}

// FlagBridgeOrder records a failed recharge on an order WITHOUT changing its
// status: the order stays processing in the manual admin queue, preserving the
// codebase's single money-out path (an admin completes it by hand or refunds via
// the refund endpoint — a bridge failure never auto-refunds). Idempotent-ish: a
// repeated flag just appends another timeline note. Implements bridge.OrderSettler.
func (s *OrderService) FlagBridgeOrder(ctx context.Context, orderID bson.ObjectID, reason string) error {
	err := s.repo.AppendTimelineEvent(ctx, orderID, TimelineEvent{
		Status: "bridge_failed",
		Note:   reason,
		At:     time.Now().UTC(),
	})
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil
	}
	return err
}
