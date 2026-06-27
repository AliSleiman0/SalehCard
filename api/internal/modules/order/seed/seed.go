// Package seed inserts demo orders for local development so the admin dashboard
// and order pages have data to render. It is idempotent and a no-op once the
// orders collection is populated.
package seed

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/order"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

// orderSpec is a compact description of a demo order to generate.
type orderSpec struct {
	daysAgo int
	title   string
	cat     string
	ff      string // code | account_credit | transfer
	qty     int
	price   float64
	pay     order.PaymentMethod
	status  order.OrderStatus
}

// specs spread orders across recent days, statuses, fulfillment types and
// payment methods. Several are completed today/yesterday so the dashboard's
// revenue/orders deltas and the revenue chart have signal.
var specs = []orderSpec{
	{0, "Steam Wallet", "giftcards", "code", 1, 50, order.PaymentMethodCard, order.OrderStatusCompleted},
	{0, "PUBG Mobile UC", "games", "account_credit", 2, 12, order.PaymentMethodWallet, order.OrderStatusCompleted},
	{0, "Bank Transfer", "transfer", "transfer", 1, 200, order.PaymentMethodUSDT, order.OrderStatusProcessing},
	{1, "Steam Wallet", "giftcards", "code", 1, 25, order.PaymentMethodCard, order.OrderStatusCompleted},
	{1, "PUBG Mobile UC", "games", "account_credit", 1, 30, order.PaymentMethodWallet, order.OrderStatusCompleted},
	{1, "Steam Wallet", "giftcards", "code", 3, 10, order.PaymentMethodUSDT, order.OrderStatusRefunded},
	{2, "PUBG Mobile UC", "games", "account_credit", 1, 60, order.PaymentMethodCard, order.OrderStatusCompleted},
	{2, "Bank Transfer", "transfer", "transfer", 1, 150, order.PaymentMethodWallet, order.OrderStatusProcessing},
	{3, "Steam Wallet", "giftcards", "code", 1, 100, order.PaymentMethodCard, order.OrderStatusCompleted},
	{3, "PUBG Mobile UC", "games", "account_credit", 2, 20, order.PaymentMethodWallet, order.OrderStatusFailed},
	{5, "Steam Wallet", "giftcards", "code", 1, 40, order.PaymentMethodUSDT, order.OrderStatusCompleted},
	{7, "PUBG Mobile UC", "games", "account_credit", 1, 75, order.PaymentMethodCard, order.OrderStatusCompleted},
	{9, "Bank Transfer", "transfer", "transfer", 1, 300, order.PaymentMethodWallet, order.OrderStatusCompleted},
	{11, "Steam Wallet", "giftcards", "code", 2, 35, order.PaymentMethodCard, order.OrderStatusCompleted},
	{13, "PUBG Mobile UC", "games", "account_credit", 1, 90, order.PaymentMethodUSDT, order.OrderStatusCompleted},
}

// Seed inserts the demo orders for the seeded customer when the orders
// collection is empty. If the customer account is missing it is a quiet no-op.
func Seed(ctx context.Context, db *mongo.Database) error {
	if err := order.EnsureIndexes(ctx, db); err != nil {
		return err
	}

	orders := db.Collection("orders")
	count, err := orders.CountDocuments(ctx, bson.D{})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var customer struct {
		ID bson.ObjectID `bson:"_id"`
	}
	err = db.Collection("users").
		FindOne(ctx, bson.D{{Key: "email", Value: "customer@salehcard.local"}}).
		Decode(&customer)
	if err != nil {
		// No customer to attach orders to; nothing to seed.
		return nil
	}

	now := time.Now().UTC()
	docs := make([]any, 0, len(specs))
	for i, s := range specs {
		total := s.price * float64(s.qty)
		// Stagger within the day so createdAt values are distinct and sortable.
		created := now.AddDate(0, 0, -s.daysAgo).Add(-time.Duration(i) * time.Hour)

		o := order.Order{
			ID:     bson.NewObjectID(),
			UserID: customer.ID,
			Items: []order.OrderItem{{
				ProductID:       bson.NewObjectID(),
				VariantID:       bson.NewObjectID(),
				Title:           product.I18nString{En: s.title, Ar: s.title, Tr: s.title},
				Category:        s.cat,
				Qty:             s.qty,
				Price:           s.price,
				FulfillmentType: s.ff,
			}},
			Subtotal:      total,
			Total:         total,
			Currency:      "USD",
			PaymentMethod: s.pay,
			Status:        s.status,
			Fulfillment:   order.Fulfillment{StatusTimeline: []order.TimelineEvent{}},
			CreatedAt:     created,
			UpdatedAt:     created,
		}
		if s.ff == "code" && s.status == order.OrderStatusCompleted {
			o.Fulfillment.DeliveredCode = fmt.Sprintf("DEMO-%04d", i)
		}
		docs = append(docs, o)
	}

	_, err = orders.InsertMany(ctx, docs)
	return err
}
