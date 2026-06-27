package promo

import (
	"context"
	"errors"
	"testing"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestDiscountFor(t *testing.T) {
	cases := []struct {
		name  string
		promo *PromoCode
		total float64
		want  float64
	}{
		{"percent", &PromoCode{Type: PromoTypePercent, Value: 10}, 200, 20},
		{"fixed under total", &PromoCode{Type: PromoTypeFixed, Value: 5}, 50, 5},
		{"fixed clamped to total", &PromoCode{Type: PromoTypeFixed, Value: 80}, 50, 50},
		{"cashback is no checkout discount", &PromoCode{Type: PromoTypeCashback, Value: 3}, 100, 0},
		{"zero total", &PromoCode{Type: PromoTypePercent, Value: 10}, 0, 0},
		{"nil promo", nil, 100, 0},
		{"percent never exceeds total", &PromoCode{Type: PromoTypePercent, Value: 150}, 40, 40},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := DiscountFor(c.promo, c.total); got != c.want {
				t.Errorf("DiscountFor(%+v, %v) = %v, want %v", c.promo, c.total, got, c.want)
			}
		})
	}
}

// fakeRepo serves a single configured promo for FindByCode; other methods are
// unused by the service tests.
type fakeRepo struct {
	promo *PromoCode
	err   error
}

func (f *fakeRepo) FindByCode(context.Context, string) (*PromoCode, error) { return f.promo, f.err }
func (f *fakeRepo) FindByID(context.Context, bson.ObjectID) (*PromoCode, error) {
	return f.promo, f.err
}
func (f *fakeRepo) List(context.Context, PromoFilter, pagination.Params) ([]*PromoCode, int64, error) {
	return nil, 0, nil
}
func (f *fakeRepo) Create(context.Context, *PromoCode) error { return nil }
func (f *fakeRepo) Update(context.Context, bson.ObjectID, PromoUpdate) (*PromoCode, error) {
	return f.promo, f.err
}
func (f *fakeRepo) Delete(context.Context, bson.ObjectID) error { return nil }
func (f *fakeRepo) IncrementUses(context.Context, string) error { return nil }

func TestValidate(t *testing.T) {
	past := time.Now().UTC().Add(-time.Hour)
	future := time.Now().UTC().Add(time.Hour)
	base := func() *PromoCode {
		return &PromoCode{Code: "X", Type: PromoTypePercent, Value: 10, Active: true, MaxUses: 100, Uses: 1, MinOrder: 0}
	}

	cases := []struct {
		name     string
		promo    *PromoCode
		repoErr  error
		total    float64
		wantOK   bool
		wantCode string
	}{
		{"valid", base(), nil, 100, true, ""},
		{"not found", nil, apperrors.ErrNotFound, 100, false, "PROMO_INVALID"},
		{"inactive", func() *PromoCode { p := base(); p.Active = false; return p }(), nil, 100, false, "PROMO_INACTIVE"},
		{"not started", func() *PromoCode { p := base(); p.StartsAt = &future; return p }(), nil, 100, false, "PROMO_NOT_STARTED"},
		{"expired", func() *PromoCode { p := base(); p.ExpiresAt = &past; return p }(), nil, 100, false, "PROMO_EXPIRED"},
		{"depleted", func() *PromoCode { p := base(); p.MaxUses = 5; p.Uses = 5; return p }(), nil, 100, false, "PROMO_DEPLETED"},
		{"below min", func() *PromoCode { p := base(); p.MinOrder = 50; return p }(), nil, 20, false, "PROMO_MIN_ORDER"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := NewPromoService(&fakeRepo{promo: c.promo, err: c.repoErr})
			got, err := svc.Validate(context.Background(), ValidateInput{Code: "X", OrderTotal: c.total})
			if c.wantOK {
				if err != nil || got == nil {
					t.Fatalf("expected ok, got err=%v promo=%v", err, got)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, apperrors.ErrBadRequest) {
				t.Errorf("expected bad-request, got %v", err)
			}
			var appErr *apperrors.AppError
			if errors.As(err, &appErr) && appErr.Code != c.wantCode {
				t.Errorf("expected code %q, got %q", c.wantCode, appErr.Code)
			}
		})
	}
}
