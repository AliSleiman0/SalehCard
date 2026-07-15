package wallet

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// TxType categorises a wallet transaction.
type TxType string

const (
	TxTypeTopUp      TxType = "topup"
	TxTypePurchase   TxType = "purchase"
	TxTypeRefund     TxType = "refund"
	TxTypeAdjustment TxType = "adjustment"
)

// WalletTransaction records a single credit or debit against a user wallet.
type WalletTransaction struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       bson.ObjectID `bson:"userId"        json:"userId"`
	Type         TxType        `bson:"type"          json:"type"`
	Amount       float64       `bson:"amount"        json:"amount"`
	BalanceAfter float64       `bson:"balanceAfter"  json:"balanceAfter"`
	Method       string        `bson:"method"        json:"method"`
	Ref          string        `bson:"ref"           json:"ref"`
	CreatedAt    time.Time     `bson:"createdAt"     json:"createdAt"`
}

// TopUpStatus is the moderation state of a top-up request.
type TopUpStatus string

const (
	TopUpPending  TopUpStatus = "pending"
	TopUpApproved TopUpStatus = "approved"
	TopUpRejected TopUpStatus = "rejected"
)

// TopUpChannels are the legacy hardcoded out-of-band payment channels. Admin-
// defined TopUpMethods (see methods.go) are the source of truth now; this map is
// kept only so a request that predates methods (or a legacy client that still
// sends a bare channel) still validates and renders.
var TopUpChannels = map[string]bool{
	"usdt": true, "whish": true, "omt": true, "cash": true, "other": true,
}

// TopUpField is one customer-submitted value for a manual top-up method,
// snapshotted with its admin-defined label (resolved server-side from the method
// spec — never trusted from the client) so the admin queue renders labeled
// values. A file-type value holds the uploaded document URL.
type TopUpField struct {
	Key   string `bson:"key"   json:"key"`
	Label string `bson:"label" json:"label"`
	Value string `bson:"value" json:"value"`
}

// TopUpFieldInput is one field the client submits for a top-up: just key + value.
// The label is looked up server-side from the method spec.
type TopUpFieldInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// TopUpRequest is a customer's ask to have their wallet credited after paying
// out-of-band (via an admin-defined manual method — bank transfer, Whish, cash,
// …). Money moves only on admin approval: approval credits the balance, writes
// the mandatory topup ledger row, and stamps TxID with the ledger id. MethodID/
// MethodName snapshot the chosen manual method and Fields the customer's
// submitted inputs (incl. uploaded document URLs); Channel is retained for
// backward compatibility with legacy requests.
type TopUpRequest struct {
	ID             bson.ObjectID  `bson:"_id,omitempty"            json:"id"`
	UserID         bson.ObjectID  `bson:"userId"                   json:"userId"`
	Amount         float64        `bson:"amount"                   json:"amount"`
	Currency       string         `bson:"currency"                 json:"currency"`
	Channel        string         `bson:"channel"                  json:"channel"`
	MethodID       *bson.ObjectID `bson:"methodId,omitempty"       json:"methodId,omitempty"`
	MethodName     string         `bson:"methodName,omitempty"     json:"methodName,omitempty"`
	Fields         []TopUpField   `bson:"fields,omitempty"         json:"fields,omitempty"`
	Note           string         `bson:"note,omitempty"           json:"note,omitempty"`
	Status         TopUpStatus    `bson:"status"                   json:"status"`
	DecidedBy      string         `bson:"decidedBy,omitempty"      json:"decidedBy,omitempty"`
	DecisionReason string         `bson:"decisionReason,omitempty" json:"decisionReason,omitempty"`
	TxID           string         `bson:"txId,omitempty"           json:"txId,omitempty"`
	CreatedAt      time.Time      `bson:"createdAt"                json:"createdAt"`
	DecidedAt      *time.Time     `bson:"decidedAt,omitempty"      json:"decidedAt,omitempty"`
}

// TopUpInput is the customer's top-up request payload: how much they want
// credited plus either an admin-defined MethodID (with its collected Fields) or,
// for legacy clients, a bare Channel.
type TopUpInput struct {
	Amount   float64           `json:"amount"`
	Channel  string            `json:"channel"`
	MethodID string            `json:"methodId"`
	Fields   []TopUpFieldInput `json:"fields"`
	Note     string            `json:"note"`
}
