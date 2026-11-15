package chatbot

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// TestRepos_Mongo exercises the kb_docs slug upsert, the owner-scoped report
// flag and the log filters against a real Mongo.
//
// Needs MONGO_TEST_URI (e.g. mongodb://localhost:27017/?directConnection=true).
// Uses a throwaway database.
func TestRepos_Mongo(t *testing.T) {
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Disconnect(ctx) }()
	db := client.Database("salehcard_test_chatbot_" + bson.NewObjectID().Hex())
	defer func() { _ = db.Drop(ctx) }()
	if err := EnsureKBIndexes(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := EnsureLogIndexes(ctx, db); err != nil {
		t.Fatal(err)
	}

	// Seed is insert-only: a second run with different text never overwrites.
	kb := NewMongoKBRepository(db)
	first := &KBDoc{ID: bson.NewObjectID(), Slug: "wallet", Title: Text{En: "Original"}, Body: Text{En: "b"}, Enabled: true}
	if ins, err := kb.InsertIfSlugMissing(ctx, first); err != nil || !ins {
		t.Fatalf("first seed should insert: %v %v", ins, err)
	}
	again := &KBDoc{ID: bson.NewObjectID(), Slug: "wallet", Title: Text{En: "Changed"}, Body: Text{En: "b"}, Enabled: true}
	if ins, err := kb.InsertIfSlugMissing(ctx, again); err != nil || ins {
		t.Fatalf("second seed must not insert: %v %v", ins, err)
	}
	got, err := kb.Get(ctx, first.ID)
	if err != nil || got.Title.En != "Original" {
		t.Fatalf("seeded doc must be untouched: %+v %v", got, err)
	}
	dup := &KBDoc{ID: bson.NewObjectID(), Slug: "wallet", Title: Text{En: "x"}, Body: Text{En: "b"}}
	if err := kb.Create(ctx, dup); !errors.Is(err, errDuplicateSlug) {
		t.Fatalf("duplicate slug must be rejected, got %v", err)
	}
	noSlug := []*KBDoc{
		{ID: bson.NewObjectID(), Title: Text{En: "a"}, Body: Text{En: "b"}},
		{ID: bson.NewObjectID(), Title: Text{En: "c"}, Body: Text{En: "d"}},
	}
	for _, d := range noSlug {
		if err := kb.Create(ctx, d); err != nil {
			t.Fatalf("slugless docs must coexist (sparse index): %v", err)
		}
	}

	logs := NewMongoLogRepository(db)
	mine := &ChatLog{ID: bson.NewObjectID(), UserID: "u1", Outcome: outcomeAnswered, CreatedAt: time.Now().UTC()}
	miss := &ChatLog{ID: bson.NewObjectID(), UserID: "u1", Outcome: outcomeUnanswered, CreatedAt: time.Now().UTC()}
	for _, l := range []*ChatLog{mine, miss} {
		if err := logs.Insert(ctx, l); err != nil {
			t.Fatal(err)
		}
	}
	if err := logs.Flag(ctx, mine.ID, "someone-else", time.Now()); !errors.Is(err, errNotFound) {
		t.Fatalf("flagging another user's log must 404, got %v", err)
	}
	if err := logs.Flag(ctx, mine.ID, "u1", time.Now()); err != nil {
		t.Fatal(err)
	}
	flagged, _, err := logs.List(ctx, logFilterFlagged, 0, 10)
	if err != nil || len(flagged) != 1 || flagged[0].ID != mine.ID {
		t.Fatalf("flagged filter: %+v %v", flagged, err)
	}
	unanswered, _, err := logs.List(ctx, logFilterUnanswered, 0, 10)
	if err != nil || len(unanswered) != 1 || unanswered[0].ID != miss.ID {
		t.Fatalf("unanswered filter: %+v %v", unanswered, err)
	}
	page, more, err := logs.List(ctx, "", 0, 1)
	if err != nil || len(page) != 1 || !more {
		t.Fatalf("paging: %d more=%v %v", len(page), more, err)
	}
}
