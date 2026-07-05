package review

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// fakeRepo is a minimal Repository for exercising the service validation. Create
// records the review it was handed; the other methods are unused here.
type fakeRepo struct {
	created    *Review
	err        error
	listFilter ReviewFilter
	existing   *Review // when set, FindByUserAndProduct returns it (user already reviewed)
}

func (f *fakeRepo) Create(_ context.Context, rv *Review) error {
	if f.err != nil {
		return f.err
	}
	f.created = rv
	return nil
}
func (f *fakeRepo) Delete(context.Context, bson.ObjectID) error { return nil }
func (f *fakeRepo) FindByID(context.Context, bson.ObjectID) (*Review, error) {
	return nil, nil
}
func (f *fakeRepo) FindByUserAndProduct(context.Context, bson.ObjectID, bson.ObjectID) (*Review, error) {
	if f.existing != nil {
		return f.existing, nil
	}
	return nil, apperrors.ErrNotFound
}
func (f *fakeRepo) List(_ context.Context, filt ReviewFilter, _ pagination.Params) ([]ReviewRow, int64, error) {
	f.listFilter = filt
	return nil, 0, nil
}
func (f *fakeRepo) UpdateStatus(context.Context, bson.ObjectID, string) (*Review, error) {
	return nil, nil
}
func (f *fakeRepo) RecomputeProductRating(context.Context, bson.ObjectID) error { return nil }

func TestCreateValidation(t *testing.T) {
	pid := bson.NewObjectID()
	uid := bson.NewObjectID()

	tests := []struct {
		name    string
		input   CreateReviewInput
		wantBad bool
	}{
		{"valid", CreateReviewInput{ProductID: pid, Rating: 5, Body: "great"}, false},
		{"missing product", CreateReviewInput{Rating: 5, Body: "great"}, true},
		{"rating too low", CreateReviewInput{ProductID: pid, Rating: 0, Body: "great"}, true},
		{"rating too high", CreateReviewInput{ProductID: pid, Rating: 6, Body: "great"}, true},
		// The note is optional now: a blank body is valid at any rating.
		{"blank body high rating ok", CreateReviewInput{ProductID: pid, Rating: 5, Body: "   "}, false},
		{"empty body low rating ok", CreateReviewInput{ProductID: pid, Rating: 1, Body: ""}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewReviewService(&fakeRepo{})
			rv, err := svc.Create(context.Background(), uid, tc.input)
			if tc.wantBad {
				if !errors.Is(err, apperrors.ErrBadRequest) {
					t.Fatalf("want bad-request, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rv.Status != StatusPending {
				t.Errorf("new review status = %q, want pending", rv.Status)
			}
			if rv.UserID != uid {
				t.Errorf("review userID not set from caller")
			}
		})
	}
}

func TestCreateTrimsBody(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewReviewService(repo)
	_, err := svc.Create(context.Background(), bson.NewObjectID(), CreateReviewInput{
		ProductID: bson.NewObjectID(), Rating: 5, Body: "  spaced  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.created.Body != "spaced" {
		t.Errorf("body = %q, want trimmed %q", repo.created.Body, "spaced")
	}
}

func TestCreateDuplicateConflict(t *testing.T) {
	pid := bson.NewObjectID()
	uid := bson.NewObjectID()
	repo := &fakeRepo{existing: &Review{ID: bson.NewObjectID(), ProductID: pid, UserID: uid}}
	svc := NewReviewService(repo)
	_, err := svc.Create(context.Background(), uid, CreateReviewInput{ProductID: pid, Rating: 5, Body: "dup"})
	if !errors.Is(err, apperrors.ErrConflict) {
		t.Fatalf("want conflict on second review, got %v", err)
	}
	if repo.created != nil {
		t.Error("duplicate review should not be inserted")
	}
}

func TestMine(t *testing.T) {
	pid := bson.NewObjectID()
	uid := bson.NewObjectID()

	// No prior review → ErrNotFound.
	if _, err := NewReviewService(&fakeRepo{}).Mine(context.Background(), uid, pid); !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("want not-found for un-reviewed product, got %v", err)
	}

	// Existing review → returned with its status.
	svc := NewReviewService(&fakeRepo{existing: &Review{Status: StatusPending}})
	rv, err := svc.Mine(context.Background(), uid, pid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rv.Status != StatusPending {
		t.Errorf("status = %q, want %q", rv.Status, StatusPending)
	}
}

func TestListApprovedFilter(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewReviewService(repo)
	pid := bson.NewObjectID()
	if _, _, err := svc.ListApproved(context.Background(), pid, pagination.Params{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.listFilter.Status != StatusApproved {
		t.Errorf("status filter = %q, want %q", repo.listFilter.Status, StatusApproved)
	}
	if repo.listFilter.ProductID == nil || *repo.listFilter.ProductID != pid {
		t.Errorf("product filter = %v, want %v", repo.listFilter.ProductID, pid)
	}
}

func TestReviewFilterBuild(t *testing.T) {
	if got := (ReviewFilter{}).build(); len(got) != 0 {
		t.Errorf("empty filter should match all, got %v", got)
	}
	// Pending also surfaces legacy empty-status rows.
	pending := ReviewFilter{Status: StatusPending}.build()
	if len(pending) == 0 {
		t.Fatal("pending filter should not be empty")
	}
	approved := ReviewFilter{Status: StatusApproved}.build()
	if len(approved) == 0 {
		t.Fatal("approved filter should not be empty")
	}
	// An unknown status is ignored (no status condition added).
	if got := (ReviewFilter{Status: "bogus"}).build(); len(got) != 0 {
		t.Errorf("unknown status should add no condition, got %v", got)
	}
	// Search adds a body condition.
	if got := (ReviewFilter{Search: "scam"}).build(); len(got) == 0 {
		t.Error("search filter should add a condition")
	}
}
