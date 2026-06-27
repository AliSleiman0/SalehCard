package expense

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ExpenseCategory is the kind of business spend an expense records. "other"
// carries a free-text label in Expense.CategoryOther.
type ExpenseCategory string

const (
	CategorySalary    ExpenseCategory = "salary"
	CategoryRent      ExpenseCategory = "rent"
	CategoryUtilities ExpenseCategory = "utilities"
	CategoryInventory ExpenseCategory = "inventory"
	CategoryMarketing ExpenseCategory = "marketing"
	CategoryFees      ExpenseCategory = "fees"
	CategoryOther     ExpenseCategory = "other"
)

// validCategories is the set of accepted categories, used for validation.
var validCategories = map[ExpenseCategory]bool{
	CategorySalary:    true,
	CategoryRent:      true,
	CategoryUtilities: true,
	CategoryInventory: true,
	CategoryMarketing: true,
	CategoryFees:      true,
	CategoryOther:     true,
}

// IsValid reports whether c is one of the known categories.
func (c ExpenseCategory) IsValid() bool { return validCategories[c] }

// Expense is a single manually-recorded business cost (salary, rent, internet,
// etc.). It is internal-only: there is no customer-facing endpoint. Amounts are
// stored per Currency so totals never mix currencies.
type Expense struct {
	ID            bson.ObjectID   `bson:"_id,omitempty"          json:"id"`
	Amount        float64         `bson:"amount"                 json:"amount"`
	Currency      string          `bson:"currency"               json:"currency"`
	Category      ExpenseCategory `bson:"category"               json:"category"`
	CategoryOther string          `bson:"categoryOther,omitempty" json:"categoryOther,omitempty"`
	Note          string          `bson:"note,omitempty"         json:"note,omitempty"`
	IncurredAt    time.Time       `bson:"incurredAt"             json:"incurredAt"`
	CreatedAt     time.Time       `bson:"createdAt"              json:"createdAt"`
	UpdatedAt     time.Time       `bson:"updatedAt"              json:"updatedAt"`
}
