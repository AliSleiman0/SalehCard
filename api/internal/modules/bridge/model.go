// Package bridge is the server side of the Mobile Bridge: a dual-SIM Android
// device that fulfills Lebanese mobile recharges (MTC Touch / Alfa) by sending
// operator SMS/USSD. Orders in bridge_device fulfillment mode enqueue a command
// here; the device leases it, executes it on the SIM, and reports the result,
// which drives the order to completed (or flags it for the manual admin queue).
//
// The module mirrors the payment module's shape: an atomic-claim Mongo store, a
// service with an OrderSettler port back into the order module (no import cycle),
// device-facing poll/report endpoints authenticated by a per-device token, an
// admin monitoring surface, and a background reaper that requeues stale leases.
package bridge

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// CommandType is the operation a device performs on a SIM. The values match the
// Android command dispatcher's switch verbatim.
const (
	TypeTransferCredit = "TRANSFER_CREDIT" // send credit from the SIM balance, by amount
	TypeRechargeLine   = "RECHARGE_LINE"   // apply a scratch-card code to the customer's line
	TypeCheckBalance   = "CHECK_BALANCE"   // query the SIM's own balance/validity
	TypeSendSMS        = "SEND_SMS"        // send an arbitrary SMS
)

// CommandStatus is a command's position in its lifecycle:
//
//	queued --(poll: atomic lease)--> leased --(result)--> succeeded | failed  [terminal]
//	  ^__(reaper: lease expired, attempts < max)__|
//	queued/leased --(admin cancel)--> cancelled;  failed/cancelled --(admin retry)--> queued
type CommandStatus string

const (
	CommandQueued    CommandStatus = "queued"
	CommandLeased    CommandStatus = "leased"
	CommandSucceeded CommandStatus = "succeeded"
	CommandFailed    CommandStatus = "failed"
	CommandCancelled CommandStatus = "cancelled"
)

// IsTerminal reports whether a command has reached a final state.
func (s CommandStatus) IsTerminal() bool {
	return s == CommandSucceeded || s == CommandFailed || s == CommandCancelled
}

// Device is a registered bridge phone. Authentication is a per-device opaque
// bearer token, stored only as its SHA-256 hash (the plaintext is shown once at
// registration and never persisted). Balances/validity echo the device's last
// heartbeat so a reinstall can restore state from the config endpoint.
type Device struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name       string        `bson:"name"          json:"name"`
	TokenHash  string        `bson:"tokenHash"     json:"-"`
	Providers  []string      `bson:"providers"     json:"providers"` // "touch" / "alfa"
	Enabled    bool          `bson:"enabled"       json:"enabled"`
	LastSeenAt *time.Time    `bson:"lastSeenAt,omitempty" json:"lastSeenAt,omitempty"`

	TouchBalance  *float64 `bson:"touchBalance,omitempty"  json:"touchBalance,omitempty"`
	TouchValidity string   `bson:"touchValidity,omitempty" json:"touchValidity,omitempty"`
	AlfaBalance   *float64 `bson:"alfaBalance,omitempty"   json:"alfaBalance,omitempty"`
	AlfaValidity  string   `bson:"alfaValidity,omitempty"  json:"alfaValidity,omitempty"`
	AppVersion    string   `bson:"appVersion,omitempty"    json:"appVersion,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// Handles returns whether the device is assigned the given provider.
func (d *Device) Handles(provider string) bool {
	for _, p := range d.Providers {
		if p == provider {
			return true
		}
	}
	return false
}

// CommandResult is the device's report of executing a command. RawReply carries
// the operator's SMS/USSD reply text verbatim (persisted for audit — the whole
// point of matching English keywords is fragile, so the source is kept).
type CommandResult struct {
	StatusCode        int       `bson:"statusCode"                  json:"statusCode"`
	TransferredAmount *float64  `bson:"transferredAmount,omitempty" json:"transferredAmount,omitempty"`
	BillingAmount     *float64  `bson:"billingAmount,omitempty"     json:"billingAmount,omitempty"`
	Balance           *float64  `bson:"balance,omitempty"           json:"balance,omitempty"`
	ValidityDate      string    `bson:"validityDate,omitempty"      json:"validityDate,omitempty"`
	RawReply          string    `bson:"rawReply,omitempty"          json:"rawReply,omitempty"`
	ErrorMessage      string    `bson:"errorMessage,omitempty"      json:"errorMessage,omitempty"`
	ExecutedAt        time.Time `bson:"executedAt"                  json:"executedAt"`
}

// Command is one unit of work for a device. CardCode is json:"-" so it never
// leaks through a struct marshal; the device wire form and the (masked) admin
// view are built explicitly in handler.go / admin.go.
type Command struct {
	ID              bson.ObjectID  `bson:"_id,omitempty"`
	OrderID         *bson.ObjectID `bson:"orderId,omitempty"`
	Provider        string         `bson:"provider"`
	Type            string         `bson:"type"`
	RecipientNumber string         `bson:"recipientNumber,omitempty"`
	Amount          *float64       `bson:"amount,omitempty"`
	CardCode        string         `bson:"cardCode,omitempty"`
	Message         string         `bson:"message,omitempty"`
	Status          CommandStatus  `bson:"status"`
	Attempts        int            `bson:"attempts"`
	DeviceID        *bson.ObjectID `bson:"deviceId,omitempty"`
	LeaseExpiresAt  *time.Time     `bson:"leaseExpiresAt,omitempty"`
	Result          *CommandResult `bson:"result,omitempty"`
	FailReason      string         `bson:"failReason,omitempty"`
	CreatedAt       time.Time      `bson:"createdAt"`
	UpdatedAt       time.Time      `bson:"updatedAt"`
}

// DispatchInput is the order module's request to enqueue a recharge command. It
// carries only primitives so the bridge package never imports order/product.
type DispatchInput struct {
	OrderID  bson.ObjectID
	Provider string   // "touch" / "alfa"
	Method   string   // "transfer_credit" / "recharge_line"
	Phone    string   // normalized Lebanese mobile number
	Amount   *float64 // face value for transfer_credit
	CardCode string   // claimed scratch-card code for recharge_line
}

// isSuccessCode reports whether a device result code is a success. The Android
// CommandResultCodes table uses x000 for success (1000 SMS, 2000 transfer, 3000
// touch recharge, 4000 alfa recharge, 5000 balance check); everything else —
// including the 2001 partial-transfer code — is a failure for order purposes.
func isSuccessCode(code int) bool {
	switch code {
	case 1000, 2000, 3000, 4000, 5000:
		return true
	default:
		return false
	}
}

// isPartialCode reports the partial-transfer outcome (some chunks sent). Treated
// as a terminal failure (never auto-retried — that would double-send).
func isPartialCode(code int) bool { return code == 2001 }
