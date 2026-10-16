package product_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

const unitID = "6a3f04c4ea6747f81d0baf8a"

func coinProduct() product.Product {
	return product.Product{
		FulfillmentType: product.FulfillmentCredit,
		FulfillmentMode: product.FulfillmentModeManualOperator,
		Variants:        []product.Variant{{ID: bson.NewObjectID(), Price: 10}},
		InputFields: []product.InputField{
			{Key: "id", Type: product.InputFieldText},
			{Key: "amount", Type: product.InputFieldAmount, Constraints: &product.InputFieldConstraints{Min: ptr(9000), Max: ptr(900000)}},
		},
	}
}

func TestProductService_Update_EnablesUnitPricing(t *testing.T) {
	repo := &mockRepo{products: []product.Product{coinProduct()}}
	svc := product.NewProductService(repo)

	_, err := svc.Update(context.Background(), unitID, product.UpdateProductInput{
		UnitPricing: &product.UnitPricing{FieldKey: " amount ", BlockUnits: 90000},
	})
	require.NoError(t, err)
	require.NotNil(t, repo.lastUpdate.UnitPricing)
	assert.Equal(t, "amount", repo.lastUpdate.UnitPricing.FieldKey)
}

func TestProductService_Update_RejectsUnitPricingWithQuantityField(t *testing.T) {
	p := coinProduct()
	p.InputFields = append(p.InputFields, product.InputField{Key: "qty", Type: product.InputFieldQuantity})
	repo := &mockRepo{products: []product.Product{p}}
	svc := product.NewProductService(repo)

	_, err := svc.Update(context.Background(), unitID, product.UpdateProductInput{
		UnitPricing: &product.UnitPricing{FieldKey: "amount", BlockUnits: 90000},
	})
	require.Error(t, err)
}

func TestProductService_Update_RejectsRemovingDrivingField(t *testing.T) {
	p := coinProduct()
	p.UnitPricing = &product.UnitPricing{FieldKey: "amount", BlockUnits: 90000}
	repo := &mockRepo{products: []product.Product{p}}
	svc := product.NewProductService(repo)

	_, err := svc.Update(context.Background(), unitID, product.UpdateProductInput{
		InputFields: []product.InputField{{Key: "id", Type: product.InputFieldText}},
	})
	require.Error(t, err)
}

func TestProductService_Update_ClearUnitPricingSkipsValidation(t *testing.T) {
	p := coinProduct()
	p.UnitPricing = &product.UnitPricing{FieldKey: "amount", BlockUnits: 90000}
	repo := &mockRepo{products: []product.Product{p}}
	svc := product.NewProductService(repo)

	_, err := svc.Update(context.Background(), unitID, product.UpdateProductInput{
		UnitPricing: &product.UnitPricing{},
		Variants:    []product.Variant{{Price: 1}, {Price: 2}},
	})
	require.NoError(t, err)
	require.NotNil(t, repo.lastUpdate.UnitPricing)
	assert.Zero(t, repo.lastUpdate.UnitPricing.BlockUnits)
}

func TestProductService_Create_RejectsUnitPricingOnCodeProduct(t *testing.T) {
	repo := &mockRepo{}
	svc := product.NewProductService(repo)
	p := coinProduct()

	_, err := svc.Create(context.Background(), product.CreateProductInput{
		FulfillmentType: product.FulfillmentCode,
		Variants:        p.Variants,
		InputFields:     p.InputFields,
		UnitPricing:     &product.UnitPricing{FieldKey: "amount", BlockUnits: 90000},
	})
	require.Error(t, err)
}

// fakeCleaner records DeletePricesForVariants calls.
type fakeCleaner struct {
	ids   []string
	calls int
}

func (f *fakeCleaner) DeletePricesForVariants(_ context.Context, ids []string) (int64, error) {
	f.calls++
	f.ids = append(f.ids, ids...)
	return int64(len(ids)), nil
}

// Switching a fixed product to per-unit drops its variant's per-reseller
// custom prices: they were flat-package prices and would now read as a
// per-block price, undercharging resellers.
func TestProductService_Update_ConversionDropsResellerCustomPrices(t *testing.T) {
	p := coinProduct()
	repo := &mockRepo{products: []product.Product{p}}
	cl := &fakeCleaner{}
	svc := product.NewProductService(repo, product.WithResellerPriceCleaner(cl))

	_, err := svc.Update(context.Background(), unitID, product.UpdateProductInput{
		UnitPricing: &product.UnitPricing{FieldKey: "amount", BlockUnits: 90000},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, cl.calls)
	assert.Equal(t, []string{p.Variants[0].ID.Hex()}, cl.ids)
}

// An already per-unit product keeps its (deliberately per-block) custom
// prices on later edits, and clearing unit pricing never cleans.
func TestProductService_Update_NoCleanupUnlessConverting(t *testing.T) {
	p := coinProduct()
	p.UnitPricing = &product.UnitPricing{FieldKey: "amount", BlockUnits: 90000}
	repo := &mockRepo{products: []product.Product{p}}
	cl := &fakeCleaner{}
	svc := product.NewProductService(repo, product.WithResellerPriceCleaner(cl))

	_, err := svc.Update(context.Background(), unitID, product.UpdateProductInput{
		UnitPricing: &product.UnitPricing{FieldKey: "amount", BlockUnits: 1000},
	})
	require.NoError(t, err)
	_, err = svc.Update(context.Background(), unitID, product.UpdateProductInput{UnitPricing: &product.UnitPricing{}})
	require.NoError(t, err)
	assert.Zero(t, cl.calls)
}

func TestProductService_Create_UnitPricingDropsSuppliedVariantPrices(t *testing.T) {
	cl := &fakeCleaner{}
	svc := product.NewProductService(&mockRepo{}, product.WithResellerPriceCleaner(cl))
	p := coinProduct()

	_, err := svc.Create(context.Background(), product.CreateProductInput{
		FulfillmentType: product.FulfillmentCredit,
		FulfillmentMode: product.FulfillmentModeManualOperator,
		Variants:        p.Variants,
		InputFields:     p.InputFields,
		UnitPricing:     &product.UnitPricing{FieldKey: "amount", BlockUnits: 90000},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{p.Variants[0].ID.Hex()}, cl.ids)
}

// The "from" price of a per-unit product is the price of its minimum amount
// (blockPrice × min ÷ blockUnits), matching the web/app list tiles.
func TestFromPrice_UnitPricedRescalesToMinimum(t *testing.T) {
	p := coinProduct() // $10 per 90,000, min 9,000
	p.UnitPricing = &product.UnitPricing{FieldKey: "amount", BlockUnits: 90000}
	assert.Equal(t, 1.0, product.FromPrice(&p))

	p.UnitPricing.FieldKey = "gone" // misconfigured → raw block price
	assert.Equal(t, 10.0, product.FromPrice(&p))

	fixed := coinProduct()
	assert.Equal(t, 10.0, product.FromPrice(&fixed))
}

// A fixed-amount offer is not advertised on a per-unit product (checkout
// ignores it); a percent offer is, with from-prices rescaled to the minimum.
func TestProductService_FindByID_UnitPricedOffers(t *testing.T) {
	p := coinProduct()
	p.ID = bson.NewObjectID()
	p.UnitPricing = &product.UnitPricing{FieldKey: "amount", BlockUnits: 90000}

	fixed := &fakeOfferLookup{discounts: map[string]product.Discount{p.ID.Hex(): {Type: "fixed", Value: 3}}}
	got, err := product.NewProductService(&mockRepo{products: []product.Product{p}}, product.WithOffers(fixed)).
		FindByID(context.Background(), p.ID.Hex())
	require.NoError(t, err)
	assert.Nil(t, got.Offer)
	assert.Nil(t, got.Variants[0].OfferPrice)

	pct := &fakeOfferLookup{discounts: map[string]product.Discount{p.ID.Hex(): {Type: "percent", Value: 20}}}
	got, err = product.NewProductService(&mockRepo{products: []product.Product{p}}, product.WithOffers(pct)).
		FindByID(context.Background(), p.ID.Hex())
	require.NoError(t, err)
	require.NotNil(t, got.Offer)
	assert.Equal(t, 1.0, got.Offer.OriginalFromPrice) // 10 × 9,000 ÷ 90,000
	assert.Equal(t, 0.8, got.Offer.OfferFromPrice)    // 8 × 9,000 ÷ 90,000
	require.NotNil(t, got.Variants[0].OfferPrice)
	assert.Equal(t, 8.0, *got.Variants[0].OfferPrice) // per block
}
