package category

import (
	"context"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// maxDepth caps the tree at 3 tiers (Collection=0, Category=1, Subcategory=2),
// matching the client's described model. Enforced server-side as a safety net
// behind the admin UI's own cap.
const maxDepth = 2

// ProductCounter is the narrow slice of the product repository (implemented by
// product.MongoRepository) the category service uses to annotate tiles with live
// product counts and to guard category deletes — without importing the whole
// product service. nil is allowed (counts skipped, delete product-guard skipped).
type ProductCounter interface {
	CountByRootDomain(ctx context.Context) (map[string]int64, error)
	CountByCategory(ctx context.Context) (map[bson.ObjectID]int64, error)
	CountAssignedTo(ctx context.Context, categoryID bson.ObjectID) (int64, error)
}

// ListItem is a category plus derived fields for the storefront: its product
// count (rolled up over the node's whole subtree; roots use the denormalized
// rootDomain count so it works pre- and post-migration) and whether it has any
// child categories (so the app decides drill-vs-list without a round-trip).
type ListItem struct {
	Category
	ProductCount *int64 `json:"productCount,omitempty"`
	HasChildren  bool   `json:"hasChildren"`
}

// Service is the business-logic contract for the category taxonomy.
type Service interface {
	List(ctx context.Context, f CategoryFilter, withCounts bool) ([]ListItem, error)
}

// CategoryService is the concrete Service.
type CategoryService struct {
	repo    Repository
	counter ProductCounter
}

// NewCategoryService constructs a CategoryService. counter may be nil.
func NewCategoryService(repo Repository, counter ProductCounter) *CategoryService {
	return &CategoryService{repo: repo, counter: counter}
}

// List returns the filtered categories, each annotated with HasChildren and
// (when withCounts) its live product count. Counts roll up the whole subtree:
// a depth-0 root uses its denormalized rootDomain count (migration-independent),
// deeper nodes sum the available products assigned to the node and all its
// descendants. The full tree is loaded once (tiny collection) to derive both the
// has-children flags and the ancestor roll-up.
func (s *CategoryService) List(ctx context.Context, f CategoryFilter, withCounts bool) ([]ListItem, error) {
	cats, err := s.repo.FindAll(ctx, f)
	if err != nil {
		return nil, err
	}
	items := make([]ListItem, len(cats))
	for i, c := range cats {
		items[i] = ListItem{Category: c}
	}

	// All nodes (incl. hidden) back the has-children flags and count roll-up.
	all, err := s.repo.FindAll(ctx, CategoryFilter{IncludeHidden: true})
	if err != nil {
		return nil, err
	}
	hasChild := make(map[bson.ObjectID]bool, len(all))
	for _, c := range all {
		if c.ParentID != nil {
			hasChild[*c.ParentID] = true
		}
	}
	for i := range items {
		items[i].HasChildren = hasChild[items[i].ID]
	}

	if !withCounts || s.counter == nil {
		return items, nil
	}

	rootCounts, err := s.counter.CountByRootDomain(ctx)
	if err != nil {
		return nil, err
	}
	byCat, err := s.counter.CountByCategory(ctx)
	if err != nil {
		return nil, err
	}
	// Roll each node's direct count up onto itself and every ancestor.
	rolled := make(map[bson.ObjectID]int64, len(all))
	for _, c := range all {
		n := byCat[c.ID]
		if n == 0 {
			continue
		}
		rolled[c.ID] += n
		for _, a := range c.Ancestors {
			rolled[a] += n
		}
	}
	for i := range items {
		var n int64
		if items[i].Depth == 0 {
			n = rootCounts[items[i].RootDomain] // authoritative for roots
		} else {
			n = rolled[items[i].ID]
		}
		items[i].ProductCount = &n
	}
	return items, nil
}

// ListAll returns the full tree (all depths, incl. hidden) with has-children +
// rolled-up counts, for the admin console category manager.
func (s *CategoryService) ListAll(ctx context.Context) ([]ListItem, error) {
	return s.List(ctx, CategoryFilter{IncludeHidden: true}, true)
}

// Get returns a single category by id.
func (s *CategoryService) Get(ctx context.Context, id bson.ObjectID) (*Category, error) {
	return s.repo.FindByID(ctx, id)
}

// Create adds a new category. A nil ParentID creates a root tier (a
// "Collection"); otherwise depth/rootDomain/ancestors are derived from the
// parent and the 3-tier cap is enforced. The slug is generated from the English
// name and uniquified among its siblings.
func (s *CategoryService) Create(ctx context.Context, in CreateCategoryInput) (*Category, error) {
	if strings.TrimSpace(in.Name.En) == "" {
		return nil, badRequest("name (English) is required")
	}
	now := time.Now().UTC()
	c := &Category{
		ID:        bson.NewObjectID(),
		Name:      in.Name,
		Image:     strings.TrimSpace(in.Image),
		SortOrder: in.SortOrder,
		Visible:   in.Visible,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if in.ParentID == nil {
		c.Depth = 0
		c.Slug = slugify(in.Name.En)
		c.RootDomain = c.Slug // a root's own slug is its domain
	} else {
		parent, err := s.findParent(ctx, *in.ParentID)
		if err != nil {
			return nil, err
		}
		if parent.Depth >= maxDepth {
			return nil, badRequest("maximum category depth reached (3 tiers)")
		}
		c.ParentID = &parent.ID
		c.Depth = parent.Depth + 1
		c.RootDomain = parent.RootDomain
		c.Ancestors = append(append([]bson.ObjectID{}, parent.Ancestors...), parent.ID)
		c.Slug = slugify(in.Name.En)
	}

	slug, err := s.uniqueSiblingSlug(ctx, c.ParentID, c.Slug, bson.ObjectID{})
	if err != nil {
		return nil, err
	}
	c.Slug = slug

	if err := s.repo.Insert(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// Update applies a partial update. A non-nil ParentID re-parents the node (a
// move), recomputing depth/rootDomain/ancestors for the node and its whole
// subtree, guarding against cycles and the 3-tier cap.
func (s *CategoryService) Update(ctx context.Context, id bson.ObjectID, in UpdateCategoryInput) (*Category, error) {
	node, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.ParentID != nil {
		if err := s.move(ctx, node, *in.ParentID); err != nil {
			return nil, err
		}
	}

	set := bson.D{}
	if in.Name != nil {
		if strings.TrimSpace(in.Name.En) == "" {
			return nil, badRequest("name (English) is required")
		}
		set = append(set, bson.E{Key: "name", Value: *in.Name})
	}
	if in.Image != nil {
		set = append(set, bson.E{Key: "image", Value: strings.TrimSpace(*in.Image)})
	}
	if in.SortOrder != nil {
		set = append(set, bson.E{Key: "sortOrder", Value: *in.SortOrder})
	}
	if in.Visible != nil {
		set = append(set, bson.E{Key: "visible", Value: *in.Visible})
	}
	if len(set) == 0 {
		// Only a move happened (or nothing) — return the current node.
		return s.repo.FindByID(ctx, id)
	}
	return s.repo.UpdateFields(ctx, id, set)
}

// move re-parents node under the category identified by newParentID, rewriting
// the whole subtree's ancestors/depth/rootDomain. Validates cycle + depth cap
// against every affected node before writing anything.
func (s *CategoryService) move(ctx context.Context, node *Category, newParentID string) error {
	np, err := s.findParent(ctx, newParentID)
	if err != nil {
		return err
	}
	// Cycle: the new parent must not be the node itself or one of its descendants.
	if np.ID == node.ID {
		return badRequest("a category cannot be its own parent")
	}
	if slices.Contains(np.Ancestors, node.ID) {
		return badRequest("cannot move a category under one of its own descendants")
	}

	subtree, err := s.repo.FindSubtree(ctx, node.ID) // sorted by depth asc, includes node
	if err != nil {
		return err
	}
	newAnc := map[bson.ObjectID][]bson.ObjectID{
		node.ID: append(append([]bson.ObjectID{}, np.Ancestors...), np.ID),
	}
	// First pass: compute + validate. subtree is depth-sorted, so a node's parent
	// is always processed before it.
	for _, c := range subtree {
		var anc []bson.ObjectID
		if c.ID == node.ID {
			anc = newAnc[node.ID]
		} else {
			panc := newAnc[*c.ParentID]
			anc = append(append([]bson.ObjectID{}, panc...), *c.ParentID)
			newAnc[c.ID] = anc
		}
		if len(anc) > maxDepth {
			return badRequest("move would exceed the maximum category depth (3 tiers)")
		}
	}
	// Second pass: write. Only the moved node changes parentId.
	for _, c := range subtree {
		anc := newAnc[c.ID]
		set := bson.D{
			{Key: "ancestors", Value: anc},
			{Key: "depth", Value: len(anc)},
			{Key: "rootDomain", Value: np.RootDomain},
		}
		if c.ID == node.ID {
			set = append(set, bson.E{Key: "parentId", Value: np.ID})
		}
		if _, err := s.repo.UpdateFields(ctx, c.ID, set); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes a category, refusing (BAD_REQUEST) when it still has child
// categories or assigned products — the admin must move/reassign them first.
func (s *CategoryService) Delete(ctx context.Context, id bson.ObjectID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return err
	}
	has, err := s.repo.HasChildren(ctx, id)
	if err != nil {
		return err
	}
	if has {
		return badRequest("this category has subcategories — move or delete them first")
	}
	if s.counter != nil {
		n, err := s.counter.CountAssignedTo(ctx, id)
		if err != nil {
			return err
		}
		if n > 0 {
			return badRequest("products are assigned to this category — reassign them first")
		}
	}
	return s.repo.Delete(ctx, id)
}

// findParent resolves a parent-id string to a category, or a BAD_REQUEST.
func (s *CategoryService) findParent(ctx context.Context, id string) (*Category, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, badRequest("invalid parent id")
	}
	parent, err := s.repo.FindByID(ctx, oid)
	if err != nil {
		if err == apperrors.ErrNotFound {
			return nil, badRequest("parent category not found")
		}
		return nil, err
	}
	return parent, nil
}

// uniqueSiblingSlug returns base, or base-2/base-3/… if a sibling (same parent /
// same root tier) already uses it. excludeID is skipped (self, on rename).
func (s *CategoryService) uniqueSiblingSlug(ctx context.Context, parentID *bson.ObjectID, base string, excludeID bson.ObjectID) (string, error) {
	f := CategoryFilter{IncludeHidden: true}
	if parentID == nil {
		zero := 0
		f.Depth = &zero
	} else {
		f.ParentID = parentID
	}
	siblings, err := s.repo.FindAll(ctx, f)
	if err != nil {
		return "", err
	}
	taken := make(map[string]bool, len(siblings))
	for _, sib := range siblings {
		if sib.ID == excludeID {
			continue
		}
		taken[sib.Slug] = true
	}
	if !taken[base] {
		return base, nil
	}
	for i := 2; ; i++ {
		cand := base + "-" + strconv.Itoa(i)
		if !taken[cand] {
			return cand, nil
		}
	}
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slugify lowercases s and collapses non-alphanumeric runs to single dashes.
func slugify(s string) string {
	out := nonSlug.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), "-")
	out = strings.Trim(out, "-")
	if out == "" {
		out = "category"
	}
	return out
}

// badRequest builds a 400-class validation error with a machine code.
func badRequest(msg string) error {
	return &apperrors.AppError{Code: "BAD_REQUEST", Message: msg, Err: apperrors.ErrBadRequest}
}
