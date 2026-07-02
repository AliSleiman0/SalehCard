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

// TopUpChannels are the out-of-band payment channels a customer can declare
// on a top-up request; the admin verifies receipt before approving.
var TopUpChannels = map[string]bool{
	"usdt": true, "whish": true, "omt": true, "cash": true, "other": true,
}

// TopUpRequest is a customer's ask to have their wallet credited after paying
// out-of-band (Whish/OMT/cash/USDT). Money moves only on admin approval:
// approval credits the balance, writes the mandatory topup ledger row, and
// stamps TxID with the ledger id.
type TopUpRequest struct {
	ID             bson.ObjectID `bson:"_id,omitempty"            json:"id"`
	UserID         bson.ObjectID `bson:"userId"                   json:"userId"`
	Amount         float64       `bson:"amount"                   json:"amount"`
	Currency       string        `bson:"currency"                 json:"currency"`
	Channel        string        `bson:"channel"                  json:"channel"`
	Note           string        `bson:"note,omitempty"           json:"note,omitempty"`
	Status         TopUpStatus   `bson:"status"                   json:"status"`
	DecidedBy      string        `bson:"decidedBy,omitempty"      json:"decidedBy,omitempty"`
	DecisionReason string        `bson:"decisionReason,omitempty" json:"decisionReason,omitempty"`
	TxID           string        `bson:"txId,omitempty"           json:"txId,omitempty"`
	CreatedAt      time.Time     `bson:"createdAt"                json:"createdAt"`
	DecidedAt      *time.Time    `bson:"decidedAt,omitempty"      json:"decidedAt,omitempty"`
}

// TopUpInput is the customer's top-up request payload: how much they want
// credited and the channel they paid (or will pay) through.
type TopUpInput struct {
	Amount  float64 `json:"amount"`
	Channel string  `json:"channel"`
	Note    string  `json:"note"`
}
