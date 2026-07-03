package audit

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Entry is one admin action recorded in the audit log: who did what to which
// entity, with a small free-form summary of the change (e.g. {"from":"admin",
// "to":"customer"} for a role change).
type Entry struct {
	ID         bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	ActorID    string         `bson:"actorId" json:"actorId"`
	ActorEmail string         `bson:"actorEmail" json:"actorEmail"`
	Action     string         `bson:"action" json:"action"`
	TargetType string         `bson:"targetType" json:"targetType"`
	TargetID   string         `bson:"targetId" json:"targetId"`
	Summary    map[string]any `bson:"summary,omitempty" json:"summary,omitempty"`
	At         time.Time      `bson:"at" json:"at"`
}

// Actions recorded in the audit log. Naming is <targetType>.<verb>.
const (
	ActionRoleChange     = "user.role_change"
	ActionStatusChange   = "user.status_change"
	ActionUserDelete     = "user.delete"
	ActionUserBulkEmail  = "user.bulk_email"
	ActionWalletAdjust   = "wallet.adjust"
	ActionBalanceAdjust  = "reseller.balance_adjust"
	ActionOrderRefund    = "order.refund"
	ActionOrderStatus    = "order.status"
	ActionProductDelete  = "product.delete"
	ActionKYCDecision    = "kyc.decision"
	ActionReviewDecision = "review.decision"
	ActionReviewDelete   = "review.delete"
	ActionTopupApprove   = "topup.approve"
	ActionTopupReject    = "topup.reject"
	ActionSettingsUpdate = "settings.update"
	ActionCodeExpire     = "code.expire"
	ActionCodeResend     = "order.resend_code"

	ActionResellerPriceSet    = "reseller.price_set"
	ActionResellerPriceDelete = "reseller.price_delete"
)

// Recorder is the write side of the audit log, passed into each admin module.
// Record is best-effort and called AFTER the action succeeds: it fills the
// actor from the request context's JWT claims and the timestamp, and logs (but
// does not propagate) persistence failures — an audit outage must not block
// admin operations. Money movements have their own mandatory ledgers.
type Recorder interface {
	Record(ctx context.Context, e Entry)
}
