package category

import "context"

// ProductCounter yields rootDomain -> number of available products. It is a
// narrow slice of the product repository (implemented by product.MongoRepository)
// so the category service can annotate root tiles without importing the whole
// product service. nil is allowed (counts are then never populated).
type ProductCounter interface {
	CountByRootDomain(ctx context.Context) (map[string]int64, error)
}

// ListItem is a category plus its optional product count. ProductCount is only
// populated for root domains (depth 0) when counts are requested.
type ListItem struct {
	Category
	ProductCount *int64 `json:"productCount,omitempty"`
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

// List returns the filtered categories. When withCounts is true and a counter is
// configured, each root domain (depth 0) is annotated with its live count of
// available products (0 when the domain has none). Non-root categories are never
// annotated, so a count always means "products in this root domain".
func (s *CategoryService) List(ctx context.Context, f CategoryFilter, withCounts bool) ([]ListItem, error) {
	cats, err := s.repo.FindAll(ctx, f)
	if err != nil {
		return nil, err
	}

	items := make([]ListItem, len(cats))
	for i, c := range cats {
		items[i] = ListItem{Category: c}
	}

	if !withCounts || s.counter == nil {
		return items, nil
	}

	counts, err := s.counter.CountByRootDomain(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Depth != 0 {
			continue
		}
		n := counts[items[i].RootDomain] // zero value when the domain has none
		items[i].ProductCount = &n
	}
	return items, nil
}
