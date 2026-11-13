package category

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// TestEnsureIndexes_MigratesNonSparseLegacyID reproduces the "an unexpected
// error occurred" on admin subcategory create: a pre-tree database carries a
// NON-sparse unique legacyId_1, so the second admin-created node (no legacyId)
// collides on null. EnsureIndexes must migrate it to sparse and build the tree
// indexes it used to skip.
//
// Needs a real Mongo: set MONGO_TEST_URI (e.g.
// mongodb://localhost:27017/?directConnection=true). Uses a throwaway db.
func TestEnsureIndexes_MigratesNonSparseLegacyID(t *testing.T) {
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Disconnect(ctx) }()

	db := client.Database("salehcard_test_catidx_" + bson.NewObjectID().Hex())
	defer func() { _ = db.Drop(ctx) }()
	col := db.Collection("categories")

	// The legacy, pre-tree index shape: unique, NOT sparse.
	if _, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "legacyId", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		t.Fatal(err)
	}
	legacy := 447
	if _, err := col.InsertOne(ctx, bson.M{"legacyId": legacy, "slug": "cat-447"}); err != nil {
		t.Fatal(err)
	}

	// Run twice: the second run must be a clean no-op.
	for i := range 2 {
		if err := EnsureIndexes(ctx, db); err != nil {
			t.Fatalf("EnsureIndexes run %d: %v", i+1, err)
		}
	}

	repo := NewMongoRepository(db)
	for _, slug := range []string{"jawaker", "clash-of-clans"} {
		if err := repo.Insert(ctx, &Category{ID: bson.NewObjectID(), Slug: slug}); err != nil {
			t.Fatalf("insert %s without legacyId: %v", slug, err)
		}
	}

	// The legacyId uniqueness guard must still hold for imported nodes.
	if _, err := col.InsertOne(ctx, bson.M{"legacyId": legacy}); !mongo.IsDuplicateKeyError(err) {
		t.Fatalf("duplicate legacyId: want E11000, got %v", err)
	}

	cur, err := col.Indexes().List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var specs []bson.M
	if err := cur.All(ctx, &specs); err != nil {
		t.Fatal(err)
	}
	names := map[string]bson.M{}
	for _, s := range specs {
		names[s["name"].(string)] = s
	}
	if sp, _ := names[legacyIDIndexName]["sparse"].(bool); !sp {
		t.Fatalf("legacyId_1 not sparse after migration: %v", names[legacyIDIndexName])
	}
	for _, want := range []string{"parentId_1", "ancestors_1", "parentId_1_slug_1"} {
		if _, ok := names[want]; !ok {
			t.Errorf("missing index %s (CreateMany was skipped?)", want)
		}
	}
}
