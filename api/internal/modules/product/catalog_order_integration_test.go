package product

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// TestCatalogOrder_Mongo exercises the real writes: SetPositions ranks only
// the node's products, the catalog listing (?sort=name) honours the ranks, and
// moving a product to another node (single update or bulk assign) drops its
// rank while a no-op re-save into the same node keeps it.
//
// Needs a real Mongo: set MONGO_TEST_URI (e.g.
// mongodb://localhost:27017/?directConnection=true). Uses a throwaway db.
func TestCatalogOrder_Mongo(t *testing.T) {
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)
	defer func() { _ = client.Disconnect(ctx) }()
	db := client.Database("salehcard_test_catorder_" + bson.NewObjectID().Hex())
	defer func() { _ = db.Drop(ctx) }()
	invalidateNameCache()
	t.Cleanup(invalidateNameCache)

	repo := NewMongoRepository(db)
	nodeA, nodeB := bson.NewObjectID(), bson.NewObjectID()
	mk := func(title string, node bson.ObjectID) bson.ObjectID {
		id := bson.NewObjectID()
		_, err := repo.col.InsertOne(ctx, bson.D{
			{Key: "_id", Value: id}, {Key: "title", Value: bson.D{{Key: "en", Value: title}}},
			{Key: "categoryId", Value: node}, {Key: "available", Value: true},
			{Key: "createdAt", Value: time.Now()},
		})
		require.NoError(t, err)
		return id
	}
	alpha, bravo, charlie := mk("Alpha", nodeA), mk("Bravo", nodeA), mk("Charlie", nodeA)
	other := mk("Other", nodeB)

	list := func(f ListFilter) []string {
		ps, _, err := repo.FindAll(ctx, f, pagination.Params{Page: 1, Limit: 50})
		require.NoError(t, err)
		out := make([]string, len(ps))
		for i := range ps {
			out[i] = ps[i].Title.En
		}
		return out
	}
	// The service sets manualOrder for one-node listings; at the repo level it
	// is set explicitly.
	titles := func(node bson.ObjectID) []string {
		return list(ListFilter{CategoryID: node.Hex(), Sort: SortByName, manualOrder: true})
	}
	assert.Equal(t, []string{"Alpha", "Bravo", "Charlie"}, titles(nodeA), "unarranged = A to Z")

	o := &orderService{store: repo}
	_, err = o.Reorder(ctx, ReorderInput{CategoryID: nodeA.Hex(), ProductIDs: []string{charlie.Hex(), alpha.Hex(), bravo.Hex()}})
	require.NoError(t, err)
	assert.Equal(t, []string{"Charlie", "Alpha", "Bravo"}, titles(nodeA), "cache invalidated, ranks honoured")
	// A cross-node listing ignores the per-node ranks.
	cross := list(ListFilter{CategoryIDs: []bson.ObjectID{nodeA, nodeB}, Sort: SortByName})
	assert.Equal(t, []string{"Alpha", "Bravo", "Charlie", "Other"}, cross)

	// Stamping is node-guarded: a product of another node is never ranked.
	matched, _, err := repo.SetPositions(ctx, nodeA, []bson.ObjectID{other})
	require.NoError(t, err)
	assert.Zero(t, matched)

	pos := func(id bson.ObjectID) int {
		p, err := repo.FindByID(ctx, id.Hex())
		require.NoError(t, err)
		return p.Position
	}

	// Re-saving into the same node keeps the rank.
	same := nodeA.Hex()
	_, err = repo.Update(ctx, alpha.Hex(), UpdateProductInput{CategoryID: &same})
	require.NoError(t, err)
	assert.Equal(t, 2, pos(alpha))

	// A real move drops it.
	dest := nodeB.Hex()
	_, err = repo.Update(ctx, alpha.Hex(), UpdateProductInput{CategoryID: &dest})
	require.NoError(t, err)
	assert.Zero(t, pos(alpha))

	// Bulk assign of a mix: the one already in dest keeps its rank, the mover
	// loses its rank, and both are counted once.
	_, _, err = repo.SetPositions(ctx, nodeB, []bson.ObjectID{other})
	require.NoError(t, err)
	n, err := repo.BulkSetCategory(ctx, Selector{IDs: []string{charlie.Hex(), other.Hex()}}, nodeB.Hex(), "b", "games")
	require.NoError(t, err)
	assert.EqualValues(t, 2, n)
	assert.Equal(t, 1, pos(other), "already in dest keeps its rank")
	assert.Zero(t, pos(charlie), "moved drops its rank")
	moved, err := repo.FindByID(ctx, charlie.Hex())
	require.NoError(t, err)
	assert.Equal(t, nodeB, *moved.CategoryID)

	// Unassign clears it too.
	_, err = repo.BulkSetCategory(ctx, Selector{IDs: []string{bravo.Hex()}}, "", "", "")
	require.NoError(t, err)
	assert.Zero(t, pos(bravo))
}
