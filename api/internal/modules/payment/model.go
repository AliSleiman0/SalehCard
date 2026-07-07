// Package payment implements async payment intents with automatic confirmation
// across two provider kinds, both settling a wallet top-up or an order with no
// admin in the loop:
//
//   - USDT (on-chain, provider "usdt"): a customer gets a unique watch-only
//     deposit address, a background watcher polls the chain (platform/tron) for
//     the transfer, and the double-credit guard is the unique (network, txHash)
//     index. Here the watcher is the ground truth.
//   - Whish (redirect, provider "whish"): the customer is redirected to a hosted
//     page (platform/whish), pays, and the result returns as an unsigned server
//     callback (HMAC-token authenticated) whose handler re-polls Whish's status
//     API as the ground truth. The double-credit guard is the unique
//     (provider, externalId) index plus the wallet (method, ref) index.
//
// Structure ports the LACPA payments module (domain state machine / store /
// service / sweeper). The shared settlement path (settle → wallet/order) serves
// both providers.
package payment

import (
	"math"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Purpose says what a confirmed intent settles.
type Purpose string

// Intent purposes.
const (
	PurposeTopUp Purpose = "topup"
	PurposeOrder Purpose = "order"
)

// IntentStatus is the payment-intent lifecycle state.
type IntentStatus string

// Intent statuses. `confirming` is the atomic claim separating "payment seen"
// from "money settled locally", so a crash between the two retries idempotently.
// A USDT intent never reaches `failed` (settlement errors stay `confirming` and
// retry every watcher tick); `failed` is a Whish-only terminal state for a
// gateway-reported failed payment.
const (
	StatusPending    IntentStatus = "pending"
	StatusConfirming IntentStatus = "confirming"
	StatusConfirmed  IntentStatus = "confirmed"
	StatusExpired    IntentStatus = "expired"
	StatusFailed     IntentStatus = "failed"
)

// Provider identifies which payment rail an intent uses. An intent stored
// before this field existed (all on-chain) decodes as "" — treat empty as usdt.
type ProviderKind = string

// Provider kinds.
const (
	ProviderUSDT  ProviderKind = "usdt"
	ProviderWhish ProviderKind = "whish"
)

// NetworkTRC20 is the only supported USDT network at launch; the field exists so
// additional chains slot in without a schema change.
const NetworkTRC20 = "trc20"

// Settlement outcomes recorded on a confirmed intent.
const (
	SettlementWalletTopUp    = "wallet_topup"            // top-up credited
	SettlementOrderFulfilled = "order_fulfilled"         // order paid + fulfilled
	SettlementUnderpaid      = "wallet_credit_underpaid" // order underpaid → received credited to wallet
	SettlementLate           = "wallet_credit_late"      // paid after expiry → credited to wallet
)

// Intent is one payment request. For USDT it carries a derived deposit address
// and the on-chain fields; for Whish it carries the gateway externalId, the
// hosted redirect URL, and the payer phone. The shared fields (purpose, amount,
// status, settlement) drive the provider-agnostic settlement path.
type Intent struct {
	ID       bson.ObjectID  `bson:"_id,omitempty"     json:"id"`
	UserID   bson.ObjectID  `bson:"userId"            json:"-"`
	Purpose  Purpose        `bson:"purpose"           json:"purpose"`
	OrderID  *bson.ObjectID `bson:"orderId,omitempty" json:"orderId,omitempty"`
	Provider ProviderKind   `bson:"provider,omitempty" json:"provider"`

	// USDT (on-chain) fields.
	Network         string `bson:"network"         json:"network"`
	Address         string `bson:"address"         json:"address"`
	DerivationIndex uint32 `bson:"derivationIndex" json:"-"`

	// Whish (redirect) fields.
	ExternalID  int64  `bson:"externalId,omitempty"  json:"-"`
	RedirectURL string `bson:"redirectUrl,omitempty" json:"redirectUrl,omitempty"`
	ProviderRef string `bson:"providerRef,omitempty" json:"-"`
	PayerPhone  string `bson:"payerPhone,omitempty"  json:"-"`

	// Amounts are integer micro-USDT (6 decimals, the TRC20 base unit) so
	// on-chain matching never touches floats; USD floats exist only at the
	// wallet/order boundary and in the JSON view.
	AmountExpectedMicros int64 `bson:"amountExpectedMicros" json:"-"`
	AmountReceivedMicros int64 `bson:"amountReceivedMicros,omitempty" json:"-"`

	Status      IntentStatus `bson:"status"                json:"status"`
	TxHash      string       `bson:"txHash,omitempty"      json:"txHash,omitempty"`
	FromAddress string       `bson:"fromAddress,omitempty" json:"-"`
	// Settlement records how the confirmed intent was applied (see the
	// Settlement* constants).
	Settlement     string `bson:"settlement,omitempty"     json:"settlement,omitempty"`
	SettleAttempts int    `bson:"settleAttempts,omitempty" json:"-"`
	IdempotencyKey string `bson:"idempotencyKey,omitempty" json:"-"`

	CreatedAt   time.Time  `bson:"createdAt"             json:"createdAt"`
	UpdatedAt   time.Time  `bson:"updatedAt"             json:"-"`
	ExpiresAt   time.Time  `bson:"expiresAt"             json:"expiresAt"`
	ConfirmedAt *time.Time `bson:"confirmedAt,omitempty" json:"confirmedAt,omitempty"`
}

// CanTransition reports whether an intent may move from cur to next.
// expired → confirming is the late-payment grace: a transfer that lands after
// expiry is still claimed and settled (as a wallet credit, never fulfillment).
// pending → failed is the Whish gateway-failure path.
func CanTransition(cur, next IntentStatus) bool {
	switch cur {
	case StatusPending:
		return next == StatusConfirming || next == StatusExpired || next == StatusFailed
	case StatusConfirming:
		return next == StatusConfirmed
	case StatusExpired:
		return next == StatusConfirming
	}
	return false
}

// IsTerminal reports whether an intent status is final (no further transitions).
func (s IntentStatus) IsTerminal() bool {
	return s == StatusConfirmed || s == StatusExpired || s == StatusFailed
}

// ProviderOf returns the intent's provider, treating a legacy empty value
// (pre-Whish on-chain intents) as usdt.
func ProviderOf(in *Intent) ProviderKind {
	if in.Provider == "" {
		return ProviderUSDT
	}
	return in.Provider
}

// MicrosToUSD converts micro-USDT to USD (1 USDT = 1 USD by decision).
func MicrosToUSD(m int64) float64 { return float64(m) / 1e6 }

// USDToMicros converts a USD amount to micro-USDT, rounding to the nearest
// micro so float noise can't shift the expected amount.
func USDToMicros(usd float64) int64 { return int64(math.Round(usd * 1e6)) }
