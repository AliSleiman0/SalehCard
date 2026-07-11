package product_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

func TestResellerUnitPrice(t *testing.T) {
	id := bson.NewObjectID()
	f := func(v float64) *float64 { return &v }

	cases := []struct {
		name   string
		v      product.Variant
		margin float64
		custom map[string]float64
		want   float64
	}{
		{"no inputs → retail", product.Variant{ID: id, Price: 100}, 0, nil, 100},
		{"tier margin applies", product.Variant{ID: id, Price: 100}, 12, nil, 88},
		{"margin 100 ignored", product.Variant{ID: id, Price: 100}, 100, nil, 100},
		{"negative margin ignored", product.Variant{ID: id, Price: 100}, -5, nil, 100},
		{"global override wins when lowest", product.Variant{ID: id, Price: 100, ResellerPrice: f(80)}, 12, nil, 80},
		{"global override above margin loses", product.Variant{ID: id, Price: 100, ResellerPrice: f(95)}, 12, nil, 88},
		{"custom price wins when lowest", product.Variant{ID: id, Price: 100, ResellerPrice: f(80)}, 12, map[string]float64{id.Hex(): 75}, 75},
		{"custom above others loses", product.Variant{ID: id, Price: 100}, 12, map[string]float64{id.Hex(): 90}, 88},
		{"custom for other variant ignored", product.Variant{ID: id, Price: 100}, 0, map[string]float64{bson.NewObjectID().Hex(): 1}, 100},
		{"never above retail", product.Variant{ID: id, Price: 100, ResellerPrice: f(150)}, 0, nil, 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.InDelta(t, tc.want, product.ResellerUnitPrice(tc.v, tc.margin, tc.custom), 1e-9)
		})
	}
}
