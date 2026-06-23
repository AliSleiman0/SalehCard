// Package legacy models the legacy SalehCard storefront API and fetches it.
//
// The source is inconsistent about numeric types — the same field arrives as a
// JSON string ("1") in one product and a JSON number (5000) in another. FlexNum
// normalizes both to float64 so the transform layer never has to care.
package legacy

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// FlexNum decodes a JSON value that may be a number, a numeric string, or null.
type FlexNum struct {
	Value float64
	Set   bool // true when the source carried a non-null value
}

// UnmarshalJSON accepts 5000, "10", "" or null.
func (f *FlexNum) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		if s == "" {
			return nil
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		f.Value, f.Set = v, true
		return nil
	}
	v, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return err
	}
	f.Value, f.Set = v, true
	return nil
}

// CategoryResponse is the top-level envelope of GET /api/category/products/{id}.
type CategoryResponse struct {
	Status string       `json:"status"`
	Data   CategoryData `json:"data"`
}

// CategoryData is the body for one fetched category: its own name, the products
// directly under it, and the sub-categories to recurse into.
type CategoryData struct {
	Name       string        `json:"name"`
	Banner     any           `json:"banner"`
	Products   []Product     `json:"products"`
	Categories []Subcategory `json:"categories"`
}

// Subcategory is a child-category index entry (used to drive recursion).
type Subcategory struct {
	ID              int    `json:"id"`
	ParentID        int    `json:"parent_id"`
	Name            string `json:"name"`
	Photo           string `json:"photo"`
	Sort            int    `json:"sort"`
	Visible         int    `json:"visible"`
	Available       int    `json:"available"`
	ProductsCount   int    `json:"products_count"`
	CategoriesCount int    `json:"categories_count"`
}

// Range is a {min,max} pair tolerant of the source's string/number mixing.
type Range struct {
	Min FlexNum `json:"min"`
	Max FlexNum `json:"max"`
}

// CheckName is the legacy ID-verification hook (becomes Verification + the api
// fulfillment provider).
type CheckName struct {
	Provider int    `json:"provider"`
	App      string `json:"app"`
}

// Require is one customer-input spec entry. type_value is polymorphic: null, a
// {min,max} object, or an array of options (selectQty) — captured raw and parsed
// by the transform layer.
type Require struct {
	ID        int             `json:"id"`
	Name      string          `json:"name"`
	Question  string          `json:"question"`
	Type      string          `json:"type"` // text | amount | quantity | selectQty
	TypeValue json.RawMessage `json:"type_value"`
}

// Product is a legacy product object (the subset we consume; the source carries
// more fields we intentionally ignore).
type Product struct {
	ID           int        `json:"id"`
	CategoryID   int        `json:"category_id"`
	CategoryName string     `json:"category_name"`
	ProductName  string     `json:"product_name"`
	Photo        string     `json:"photo"`
	MainPrice    FlexNum    `json:"mainPrice"`  // retail (per unit)
	BasePrice    FlexNum    `json:"base_price"` // cost from provider
	Available    bool       `json:"available"`
	Visible      int        `json:"visible"`
	Sort         int        `json:"sort"`
	Description  string     `json:"description"` // sometimes HTML, sometimes null→""
	IsCredit     bool       `json:"isCredit"`
	Offer        bool       `json:"offer"`
	CheckName    *CheckName `json:"check_name"`
	Amount       *Range     `json:"amount"`
	Qty          *Range     `json:"qty"`
	Requires     []Require  `json:"requires"`
}
