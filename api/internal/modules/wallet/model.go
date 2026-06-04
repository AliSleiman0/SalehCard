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

// TopUpInput holds the data required to add funds to a wallet.
type TopUpInput struct {
	Amount float64 `json:"amount"`
	Method string  `json:"method"`
	Ref    string  `json:"ref"`
}
