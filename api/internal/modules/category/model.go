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

// Category is one node in the catalog taxonomy tree. Nodes come from two sources:
// the legacy-store migration (keyed on LegacyID/ParentLegacyID) and the admin
// console (no legacy ids). ParentID + Ancestors are the uniform, source-agnostic
// tree key used by every query going forward — the migration backfills them onto
// the legacy nodes (see cmd/catmigrate).
type Category struct {
	ID bson.ObjectID `bson:"_id,omitempty" json:"id"`
	// LegacyID is a pointer so admin-created nodes carry no id (the collection's
	// legacyId index is sparse-unique). Set only by the migration loader.
	LegacyID       *int `bson:"legacyId,omitempty"       json:"legacyId,omitempty"`
	ParentLegacyID *int `bson:"parentLegacyId,omitempty" json:"parentLegacyId,omitempty"`
	// ParentID is the Mongo-native parent pointer (nil = root/top tier = a
	// "Collection"). Ancestors is the materialized root→…→parent chain, enabling
	// one-query descendant lookups without $graphLookup (not guaranteed on Cosmos).
	ParentID   *bson.ObjectID  `bson:"parentId,omitempty"   json:"parentId,omitempty"`
	Ancestors  []bson.ObjectID `bson:"ancestors,omitempty"  json:"-"`
	Slug       string          `bson:"slug"                 json:"slug"`
	Name       I18nString      `bson:"name"                 json:"name"`
	Image      string          `bson:"image,omitempty"      json:"image,omitempty"`
	SortOrder  int             `bson:"sortOrder"            json:"sortOrder"`
	RootDomain string          `bson:"rootDomain,omitempty" json:"rootDomain,omitempty"`
	Depth      int             `bson:"depth"                json:"depth"`
	Visible    bool            `bson:"visible"              json:"visible"`
	CreatedAt  time.Time       `bson:"createdAt"            json:"createdAt"`
	UpdatedAt  time.Time       `bson:"updatedAt"            json:"updatedAt"`
}

// CategoryFilter holds the optional filters for listing categories.
// VisibleOnly is set by the public handler; Depth==0 selects root domains.
// IncludeHidden is the admin escape hatch (list hidden nodes too) — it is
// mutually exclusive with VisibleOnly and neither is set means "all visible".
type CategoryFilter struct {
	Depth          *int
	RootDomain     string
	ParentLegacyID *int
	ParentID       *bson.ObjectID
	VisibleOnly    bool
	IncludeHidden  bool
}

// CreateCategoryInput is the admin create payload. ParentID nil creates a root
// tier (a "Collection"); Depth/RootDomain/Ancestors are derived server-side.
type CreateCategoryInput struct {
	ParentID  *string
	Name      I18nString
	Image     string
	SortOrder int
	Visible   bool
}

// UpdateCategoryInput is the admin partial-update payload; a nil pointer means
// "leave unchanged". A non-nil ParentID re-parents the node (a move), which
// recomputes depth/rootDomain/ancestors for it and its whole subtree.
type UpdateCategoryInput struct {
	Name      *I18nString
	Image     *string
	SortOrder *int
	Visible   *bool
	ParentID  *string // move; "" or "null" is not accepted — reparent to root is a distinct op (unsupported v1)
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
