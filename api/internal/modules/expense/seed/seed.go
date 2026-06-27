// Package seed inserts development expenses so the admin Expenses list, filters,
// and totals are demoable. Idempotent: each sample is keyed on its note, so
// re-running inserts nothing new. Safe to run repeatedly.
package seed

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/internal/modules/expense"
)

// expenseSeed is one demo expense to ensure exists (keyed on Note).
type expenseSeed struct {
	Amount        float64
	Currency      string
	Category      expense.ExpenseCategory
	CategoryOther string
	Note          string
	DaysAgo       int // IncurredAt = now - this many days
}

// Seed ensures a handful of demo expenses across categories and dates exist.
func Seed(ctx context.Context, db *mongo.Database) error {
	if err := expense.EnsureIndexes(ctx, db); err != nil {
		return err
	}

	defs := []expenseSeed{
		{Amount: 1200, Currency: "USD", Category: expense.CategorySalary, Note: "seed: June salary - Ahmad", DaysAgo: 2},
		{Amount: 450, Currency: "USD", Category: expense.CategoryRent, Note: "seed: Office rent - June", DaysAgo: 5},
		{Amount: 60, Currency: "USD", Category: expense.CategoryUtilities, Note: "seed: Fiber internet - June", DaysAgo: 8},
		{Amount: 150, Currency: "USD", Category: expense.CategoryMarketing, Note: "seed: Instagram ads", DaysAgo: 12},
		{Amount: 35, Currency: "USD", Category: expense.CategoryOther, CategoryOther: "Bank charges", Note: "seed: Monthly account fee", DaysAgo: 15},
	}

	now := time.Now().UTC()
	col := db.Collection("expenses")
	for _, d := range defs {
		incurredAt := now.AddDate(0, 0, -d.DaysAgo)
		_, err := col.UpdateOne(ctx,
			bson.D{{Key: "note", Value: d.Note}},
			bson.D{{Key: "$setOnInsert", Value: bson.D{
				{Key: "amount", Value: d.Amount},
				{Key: "currency", Value: d.Currency},
				{Key: "category", Value: d.Category},
				{Key: "categoryOther", Value: d.CategoryOther},
				{Key: "note", Value: d.Note},
				{Key: "incurredAt", Value: incurredAt},
				{Key: "createdAt", Value: now},
				{Key: "updatedAt", Value: now},
			}}},
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return err
		}
	}
	return nil
}
