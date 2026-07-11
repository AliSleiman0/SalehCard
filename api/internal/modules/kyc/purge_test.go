package kyc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestPurgeByUser_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	uid := bson.NewObjectID()
	repo.byUser[uid] = &Submission{
		UserID:           uid,
		DocumentFrontURL: "http://cdn.test/uploads/kyc/front.jpg",
		DocumentBackURL:  "http://cdn.test/uploads/kyc/back.jpg",
	}
	store := &fakeStorage{}

	purged, err := NewPurger(repo, store).PurgeByUser(context.Background(), uid)
	require.NoError(t, err)
	assert.True(t, purged)

	// Both photos deleted under their derived keys, and the row is gone.
	assert.Equal(t, []string{"kyc/front.jpg", "kyc/back.jpg"}, store.deleted)
	_, err = repo.FindByUserID(context.Background(), uid)
	assert.Error(t, err)
}

func TestPurgeByUser_PassportSkipsMissingBack(t *testing.T) {
	repo := newFakeRepo()
	uid := bson.NewObjectID()
	repo.byUser[uid] = &Submission{
		UserID:           uid,
		DocumentFrontURL: "http://cdn.test/uploads/kyc/front.jpg", // passports have no back photo
	}
	store := &fakeStorage{}

	purged, err := NewPurger(repo, store).PurgeByUser(context.Background(), uid)
	require.NoError(t, err)
	assert.True(t, purged)
	assert.Equal(t, []string{"kyc/front.jpg"}, store.deleted)
}

func TestPurgeByUser_NoSubmissionIsNoOp(t *testing.T) {
	repo := newFakeRepo()
	store := &fakeStorage{}

	purged, err := NewPurger(repo, store).PurgeByUser(context.Background(), bson.NewObjectID())
	require.NoError(t, err)
	assert.False(t, purged)
	assert.Empty(t, store.deleted)
}

func TestPurgeByUser_BlobErrorAbortsBeforeRowDelete(t *testing.T) {
	repo := newFakeRepo()
	uid := bson.NewObjectID()
	repo.byUser[uid] = &Submission{
		UserID:           uid,
		DocumentFrontURL: "http://cdn.test/uploads/kyc/front.jpg",
	}
	store := &fakeStorage{delErr: errors.New("blob backend down")}

	_, err := NewPurger(repo, store).PurgeByUser(context.Background(), uid)
	require.Error(t, err)

	// The row must survive a blob failure — retryable, never orphaning photos.
	sub, err := repo.FindByUserID(context.Background(), uid)
	require.NoError(t, err)
	assert.Equal(t, uid, sub.UserID)
}

func TestKeyFromDocURL(t *testing.T) {
	cases := map[string]struct {
		url  string
		want string
	}{
		"azure URL":     {"https://acct.blob.core.windows.net/product-images/kyc/ab12.jpg", "kyc/ab12.jpg"},
		"local URL":     {"http://localhost:8090/uploads/kyc/ab12.jpg", "kyc/ab12.jpg"},
		"empty":         {"", ""},
		"junk":          {"://not a url", ""},
		"off keyspace":  {"https://cdn.test/uploads/products/ab12.jpg", ""},
		"kyc elsewhere": {"https://cdn.test/x/kyc/deep/nested.jpg", "kyc/deep/nested.jpg"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, keyFromDocURL(tc.url))
		})
	}
}
