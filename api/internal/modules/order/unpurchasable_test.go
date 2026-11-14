package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// A hidden (draft) or trashed product is refused at checkout with
// PRODUCT_UNAVAILABLE even when the caller still holds its id — and nothing is
// charged, claimed or persisted.
func TestPlaceOrder_RefusesUnpurchasableProduct(t *testing.T) {
	now := time.Now()
	cases := map[string]func(p *product.Product){
		"hidden (draft)":   func(p *product.Product) { p.Available = false },
		"trashed (live)":   func(p *product.Product) { p.DeletedAt = &now },
		"trashed (hidden)": func(p *product.Product) { p.Available = false; p.DeletedAt = &now },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p := codeProduct(10, nil)
			mutate(p)
			codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
			walletSvc := &fakeWalletSvc{balance: 100}
			svc, repo := newSUT(p, codeSvc, walletSvc)

			_, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k-"+name, PlaceOrderInput{
				Items:         []PlaceOrderItemInput{itemFor(p, 1)},
				PaymentMethod: PaymentMethodWallet,
			})
			require.Error(t, err)
			assert.ErrorIs(t, err, apperrors.ErrBadRequest)
			var ae *apperrors.AppError
			require.True(t, errors.As(err, &ae))
			assert.Equal(t, "PRODUCT_UNAVAILABLE", ae.Code)

			assert.Equal(t, 100.0, walletSvc.balance, "wallet must not be charged")
			assert.Equal(t, 5, codeSvc.available[p.ID.Hex()], "no code may be claimed")
			assert.Empty(t, repo.byID, "no order may be persisted")
		})
	}
}
