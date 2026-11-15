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

// TestRepos_Mongo exercises the single knowledge document, the owner-scoped report
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
	if err := EnsureLogIndexes(ctx, db); err != nil {
		t.Fatal(err)
	}

	// Knowledge: absent → nil; seed is insert-only; Save upserts the one doc.
	kn := NewMongoKnowledgeRepository(db)
	if k, err := kn.Get(ctx); err != nil || k != nil {
		t.Fatalf("no knowledge yet must be nil: %+v %v", k, err)
	}
	if ins, err := kn.InsertIfMissing(ctx, &Knowledge{General: "Original"}); err != nil || !ins {
		t.Fatalf("first seed should insert: %v %v", ins, err)
	}
	if ins, err := kn.InsertIfMissing(ctx, &Knowledge{General: "Changed"}); err != nil || ins {
		t.Fatalf("second seed must not insert: %v %v", ins, err)
	}
	if k, _ := kn.Get(ctx); k == nil || k.General != "Original" {
		t.Fatalf("seeded knowledge must be untouched: %+v", k)
	}
	if err := kn.Save(ctx, &Knowledge{General: "Edited", AndroidOnly: "A"}); err != nil {
		t.Fatal(err)
	}
	if k, _ := kn.Get(ctx); k == nil || k.General != "Edited" || k.AndroidOnly != "A" {
		t.Fatalf("save must replace: %+v", k)
	}
	if n, _ := db.Collection("chatbot_knowledge").CountDocuments(ctx, bson.D{}); n != 1 {
		t.Fatalf("knowledge is a single document, got %d", n)
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
