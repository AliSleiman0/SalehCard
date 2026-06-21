// Package category persists the catalog taxonomy migrated from the legacy store
// (loaded by api/cmd/loadseed) and exposes a read-only GET /api/v1/categories so
// the storefront can browse by the imported taxonomy (top-level root domains).
// Products still carry a flat category slug + rootDomain; the admin keeps its own
// category list for now (a follow-up wires it to this endpoint).
package category

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// I18nString mirrors the product module's localized-string shape (kept local so
// category doesn't import product).
type I18nString struct {
	En string `bson:"en" json:"en"`
	Ar string `bson:"ar" json:"ar"`
	Tr string `bson:"tr" json:"tr"`
}

// Category is one node in the migrated taxonomy.
type Category struct {
	ID             bson.ObjectID `bson:"_id,omitempty"            json:"id"`
	LegacyID       int           `bson:"legacyId"                 json:"legacyId"`
	ParentLegacyID *int          `bson:"parentLegacyId,omitempty" json:"parentLegacyId,omitempty"`
	Slug           string        `bson:"slug"                     json:"slug"`
	Name           I18nString    `bson:"name"                     json:"name"`
	Image          string        `bson:"image,omitempty"          json:"image,omitempty"`
	SortOrder      int           `bson:"sortOrder"                json:"sortOrder"`
	RootDomain     string        `bson:"rootDomain,omitempty"     json:"rootDomain,omitempty"`
	Depth          int           `bson:"depth"                    json:"depth"`
	Visible        bool          `bson:"visible"                  json:"visible"`
	CreatedAt      time.Time     `bson:"createdAt"                json:"createdAt"`
	UpdatedAt      time.Time     `bson:"updatedAt"                json:"updatedAt"`
}

// CategoryFilter holds the optional filters for listing categories.
// VisibleOnly is set by the public handler; Depth==0 selects root domains.
type CategoryFilter struct {
	Depth          *int
	RootDomain     string
	ParentLegacyID *int
	VisibleOnly    bool
}

// UpsertCategoryInput is the loader input, keyed on LegacyID.
type UpsertCategoryInput struct {
	LegacyID       int
	ParentLegacyID *int
	Slug           string
	Name           I18nString
	Image          string
	SortOrder      int
	RootDomain     string
	Depth          int
	Visible        bool
}
