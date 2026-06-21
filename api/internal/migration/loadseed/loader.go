package loadseed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AliSleiman0/salehcard/api/internal/modules/category"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// CategoryUpserter is the slice of the category repository the loader needs.
type CategoryUpserter interface {
	Upsert(ctx context.Context, in category.UpsertCategoryInput) (*category.Category, error)
	FindByLegacyID(ctx context.Context, legacyID int) (*category.Category, error)
}

// ProductUpserter is the slice of the product repository the loader needs.
type ProductUpserter interface {
	Upsert(ctx context.Context, in product.UpsertProductInput) (*product.Product, error)
	FindByLegacyID(ctx context.Context, legacyID int) (*product.Product, error)
}

// Loader upserts seed JSON into the category + product repositories.
type Loader struct {
	Cats  CategoryUpserter
	Prods ProductUpserter
}

// Report summarizes a run.
type Report struct {
	CategoriesInserted int
	CategoriesUpdated  int
	ProductsInserted   int
	ProductsUpdated    int
}

// Run loads categories.json then products.json from dir. With dryRun=true it
// classifies each row as insert-vs-update and writes nothing. Categories are
// loaded before products so product.categorySlug references resolve.
func (l *Loader) Run(ctx context.Context, dir string, dryRun bool) (Report, error) {
	var rep Report

	var cats []SeedCategory
	if err := readSeed(filepath.Join(dir, "categories.json"), &cats); err != nil {
		return rep, err
	}
	var prods []SeedProduct
	if err := readSeed(filepath.Join(dir, "products.json"), &prods); err != nil {
		return rep, err
	}

	// Index categories by legacyId so each product can be stamped with its
	// category's rootDomain (seed products carry only legacyCategoryId).
	rootByCatLegacyID := make(map[int]string, len(cats))
	for _, sc := range cats {
		rootByCatLegacyID[sc.LegacyID] = sc.RootDomain
	}

	for _, sc := range cats {
		exists, err := categoryExists(ctx, l.Cats, sc.LegacyID)
		if err != nil {
			return rep, err
		}
		if dryRun {
			countInsertUpdate(exists, &rep.CategoriesInserted, &rep.CategoriesUpdated)
			continue
		}
		if _, err := l.Cats.Upsert(ctx, MapCategory(sc)); err != nil {
			return rep, fmt.Errorf("upsert category %d: %w", sc.LegacyID, err)
		}
		countInsertUpdate(exists, &rep.CategoriesInserted, &rep.CategoriesUpdated)
	}

	for _, sp := range prods {
		exists, err := productExists(ctx, l.Prods, sp.LegacyID)
		if err != nil {
			return rep, err
		}
		if dryRun {
			countInsertUpdate(exists, &rep.ProductsInserted, &rep.ProductsUpdated)
			continue
		}
		rootDomain := rootByCatLegacyID[sp.LegacyCategoryID]
		if _, err := l.Prods.Upsert(ctx, MapProduct(sp, rootDomain)); err != nil {
			return rep, fmt.Errorf("upsert product %d: %w", sp.LegacyID, err)
		}
		countInsertUpdate(exists, &rep.ProductsInserted, &rep.ProductsUpdated)
	}

	return rep, nil
}

func categoryExists(ctx context.Context, repo CategoryUpserter, legacyID int) (bool, error) {
	_, err := repo.FindByLegacyID(ctx, legacyID)
	return existsFromErr(err)
}

func productExists(ctx context.Context, repo ProductUpserter, legacyID int) (bool, error) {
	_, err := repo.FindByLegacyID(ctx, legacyID)
	return existsFromErr(err)
}

func existsFromErr(err error) (bool, error) {
	if err == nil {
		return true, nil
	}
	if errors.Is(err, apperrors.ErrNotFound) {
		return false, nil
	}
	return false, err
}

func countInsertUpdate(exists bool, inserted, updated *int) {
	if exists {
		*updated++
	} else {
		*inserted++
	}
}

func readSeed[T any](path string, out *[]T) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read seed %s: %w", path, err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("parse seed %s: %w", path, err)
	}
	return nil
}
