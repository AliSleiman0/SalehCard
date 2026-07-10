// Package payment implements on-chain USDT payment intents with automatic
// confirmation: a customer gets a unique watch-only deposit address per
// intent, a background watcher polls the chain (platform/tron) for the
// transfer, and a confirmed payment settles either a wallet top-up or an
// order — no admin in the loop. Structure ports the LACPA payments module
// (domain state machine / store / service / sweeper) minus the parts a
// redirect gateway needs (callbacks, HMAC tokens): here the watcher is the
// ground truth and the double-credit guard is the unique (network, txHash)
// index.
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

// Intent statuses. `confirming` is the atomic watcher claim separating
// "transfer seen on-chain" from "money settled locally", so a crash between
// the two retries idempotently. There is no failed status — settlement errors
// stay confirming and retry every watcher tick.
const (
	StatusPending    IntentStatus = "pending"
	StatusConfirming IntentStatus = "confirming"
	StatusConfirmed  IntentStatus = "confirmed"
	StatusExpired    IntentStatus = "expired"
)

// Supported networks. TRC20 supports both address modes; BEP20 is
// shared-address only (no BSC xpub derivation exists).
const (
	NetworkTRC20 = "trc20"
	NetworkBEP20 = "bep20"
)

// Address modes. Derived = a unique HD address per intent (identity = address).
// Shared = one fixed deposit address for everyone (identity = exact salted
// amount). Stamped per-intent so flipping the configured mode mid-flight is
// safe: open intents keep settling under the rules they were created with.
const (
	AddressModeDerived = "derived"
	AddressModeShared  = "shared"
)

// Settlement outcomes recorded on a confirmed intent.
const (
	SettlementWalletTopUp    = "wallet_topup"            // top-up credited
	SettlementOrderFulfilled = "order_fulfilled"         // order paid + fulfilled
	SettlementUnderpaid      = "wallet_credit_underpaid" // order underpaid → received credited to wallet
	SettlementLate           = "wallet_credit_late"      // paid after expiry → credited to wallet
)

// Intent is one on-chain payment request: a derived deposit address, the
// expected amount, and what a confirmed transfer settles.
type Intent struct {
	ID      bson.ObjectID  `bson:"_id,omitempty"     json:"id"`
	UserID  bson.ObjectID  `bson:"userId"            json:"-"`
	Purpose Purpose        `bson:"purpose"           json:"purpose"`
	OrderID *bson.ObjectID `bson:"orderId,omitempty" json:"orderId,omitempty"`

	Network         string `bson:"network"         json:"network"`
	Address         string `bson:"address"         json:"address"`
	DerivationIndex uint32 `bson:"derivationIndex" json:"-"`
	// AddressMode is AddressModeDerived or AddressModeShared; empty means
	// legacy derived (documents that predate shared mode).
	AddressMode string `bson:"addressMode,omitempty" json:"-"`
	// AmountSaltMicros is the shared-mode disambiguation salt folded into
	// AmountExpectedMicros (bookkeeping: base = expected - salt).
	AmountSaltMicros int64 `bson:"amountSaltMicros,omitempty" json:"-"`
	// SharedOpen marks a shared-mode intent whose amount slot is still
	// reserved; a unique partial index on (amountExpectedMicros) over these
	// docs guarantees no two open intents can match the same transfer. Unset
	// on confirmation and by the watcher once the late-payment grace ends.
	SharedOpen bool `bson:"sharedOpen,omitempty" json:"-"`

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
func CanTransition(cur, next IntentStatus) bool {
	switch cur {
	case StatusPending:
		return next == StatusConfirming || next == StatusExpired
	case StatusConfirming:
		return next == StatusConfirmed
	case StatusExpired:
		return next == StatusConfirming
	}
	return false
}

// IsShared reports whether the intent was created in shared-address mode
// (empty AddressMode = legacy derived).
func (in *Intent) IsShared() bool { return in.AddressMode == AddressModeShared }

// MicrosToUSD converts micro-USDT to USD (1 USDT = 1 USD by decision).
func MicrosToUSD(m int64) float64 { return float64(m) / 1e6 }

// USDToMicros converts a USD amount to micro-USDT, rounding to the nearest
// micro so float noise can't shift the expected amount.
func USDToMicros(usd float64) int64 { return int64(math.Round(usd * 1e6)) }
