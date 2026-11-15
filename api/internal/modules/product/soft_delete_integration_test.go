package product

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// TestSoftDelete_Mongo exercises the real trash writes: {deletedAt: null}
// matches legacy docs without the field, the $and guards keep writes
// idempotent, restore never touches `available`, purge only erases trashed
// products, "select all matching" by filter, and the interplay with manual
// category order (a trashed product is not arrangeable and loses its rank).
//
// Needs a real Mongo: set MONGO_TEST_URI (e.g.
// mongodb://localhost:27017/?directConnection=true). Uses a throwaway db.
func TestSoftDelete_Mongo(t *testing.T) {
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)
	defer func() { _ = client.Disconnect(ctx) }()
	db := client.Database("salehcard_test_softdel_" + bson.NewObjectID().Hex())
	defer func() { _ = db.Drop(ctx) }()
	invalidateNameCache()
	t.Cleanup(invalidateNameCache)

	repo := NewMongoRepository(db)
	node := bson.NewObjectID()
	// Legacy-shaped docs: no deletedAt field at all.
	mk := func(title string, available bool) bson.ObjectID {
		id := bson.NewObjectID()
		_, err := repo.col.InsertOne(ctx, bson.D{
			{Key: "_id", Value: id}, {Key: "title", Value: bson.D{{Key: "en", Value: title}}},
			{Key: "categoryId", Value: node}, {Key: "available", Value: available},
			{Key: "createdAt", Value: time.Now()},
		})
		require.NoError(t, err)
		return id
	}
	live, draft, other := mk("Live", true), mk("Draft", false), mk("Other", true)

	count := func(f ListFilter) int64 {
		_, n, err := repo.FindAll(ctx, f, pagination.Params{Page: 1, Limit: 50})
		require.NoError(t, err)
		return n
	}
	yes := true
	require.EqualValues(t, 3, count(ListFilter{}), "legacy docs without deletedAt are not trashed")
	require.EqualValues(t, 0, count(ListFilter{Status: StatusDeleted}))

	// Rank the node, then trash the live product: it leaves every listing,
	// keeps `available`, loses its rank and is no longer arrangeable.
	_, _, err = repo.SetPositions(ctx, node, []bson.ObjectID{other, live, draft})
	require.NoError(t, err)
	require.NoError(t, repo.SoftDelete(ctx, live.Hex()))
	first, err := repo.FindByID(ctx, live.Hex())
	require.NoError(t, err)
	require.NotNil(t, first.DeletedAt)
	assert.True(t, first.Available, "soft delete never touches available")
	assert.Zero(t, first.Position, "a trashed product loses its rank")
	assert.EqualValues(t, 2, count(ListFilter{}))
	assert.EqualValues(t, 1, count(ListFilter{Available: &yes}), "customer list: only Other")
	assert.EqualValues(t, 1, count(ListFilter{Status: StatusDeleted}))
	ids, err := repo.DirectProductIDs(ctx, node, 100)
	require.NoError(t, err)
	assert.ElementsMatch(t, []bson.ObjectID{other, draft}, ids, "trashed products are not arrangeable")
	matched, _, err := repo.SetPositions(ctx, node, []bson.ObjectID{live})
	require.NoError(t, err)
	assert.Zero(t, matched, "a trashed product is never ranked")

	// Idempotent: trashing again keeps the original deletedAt.
	require.NoError(t, repo.SoftDelete(ctx, live.Hex()))
	again, err := repo.FindByID(ctx, live.Hex())
	require.NoError(t, err)
	assert.True(t, first.DeletedAt.Equal(*again.DeletedAt))

	// Purge refuses a product that is not in the trash; bulk purge skips it.
	assert.True(t, errors.Is(repo.Purge(ctx, other.Hex()), apperrors.ErrConflict))
	n, err := repo.BulkPurge(ctx, []string{other.Hex()})
	require.NoError(t, err)
	assert.Zero(t, n)
	assert.True(t, errors.Is(repo.Purge(ctx, bson.NewObjectID().Hex()), apperrors.ErrNotFound))

	// "Select all matching" (empty filter = every non-trashed product): only
	// the two not-yet-trashed products are moved.
	n, err = repo.BulkSoftDelete(ctx, Selector{Filter: &ListFilter{}})
	require.NoError(t, err)
	assert.EqualValues(t, 2, n)
	assert.EqualValues(t, 0, count(ListFilter{}))

	// Restore all from the trash view: each product returns to its prior
	// available state (the draft stays a draft).
	n, err = repo.BulkRestore(ctx, Selector{Filter: &ListFilter{Status: StatusDeleted}})
	require.NoError(t, err)
	assert.EqualValues(t, 3, n)
	assert.EqualValues(t, 2, count(ListFilter{Available: &yes}))
	d, err := repo.FindByID(ctx, draft.Hex())
	require.NoError(t, err)
	assert.False(t, d.Available, "a restored draft stays a draft")
	assert.Nil(t, d.DeletedAt)

	// A restore filter from the normal view (non-trash) matches nothing.
	n, err = repo.BulkRestore(ctx, Selector{Filter: &ListFilter{}})
	require.NoError(t, err)
	assert.Zero(t, n)

	// Purge works once trashed.
	require.NoError(t, repo.SoftDelete(ctx, other.Hex()))
	require.NoError(t, repo.Purge(ctx, other.Hex()))
	_, err = repo.FindByID(ctx, other.Hex())
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))
}
