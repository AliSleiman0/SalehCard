package complaint

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// TestNextNumber_UniqueUnderConcurrency allocates numbers from many goroutines
// at once and inserts them, so both the $inc counter and the unique number
// index are exercised against a real Mongo.
//
// Needs MONGO_TEST_URI (e.g. mongodb://localhost:27017/?directConnection=true).
// Uses a throwaway database.
func TestNextNumber_UniqueUnderConcurrency(t *testing.T) {
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
	db := client.Database("salehcard_test_complaints_" + bson.NewObjectID().Hex())
	defer func() { _ = db.Drop(ctx) }()

	if err := EnsureIndexes(ctx, db); err != nil {
		t.Fatal(err)
	}
	repo := NewMongoRepository(db)
	const n = 50
	var wg sync.WaitGroup
	nums := make(chan int, n)
	errs := make(chan error, n)
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			num, err := repo.NextNumber(ctx)
			if err != nil {
				errs <- err
				return
			}
			now := time.Now().UTC()
			if err := repo.Insert(ctx, &Complaint{
				ID: bson.NewObjectID(), Number: num, UserID: bson.NewObjectID(), Category: CategoryOther,
				Subject: "s", Status: StatusOpen, Messages: []Message{}, CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				errs <- err
				return
			}
			nums <- num
		}()
	}
	wg.Wait()
	close(nums)
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent create failed: %v", err)
	}
	seen := map[int]bool{}
	for num := range nums {
		if seen[num] {
			t.Fatalf("duplicate number %d", num)
		}
		seen[num] = true
	}
	if len(seen) != n {
		t.Fatalf("want %d numbers, got %d", n, len(seen))
	}
	for i := 1; i <= n; i++ {
		if !seen[numberBase+i] {
			t.Errorf("gap: %d missing", numberBase+i)
		}
	}

	// The guarded append refuses a stale expected status.
	var c Complaint
	if err := db.Collection("complaints").FindOne(ctx, bson.D{}).Decode(&c); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AppendMessage(ctx, c.ID, nil, StatusClosed, Message{ID: bson.NewObjectID()}, nil); err != errConflict {
		t.Fatalf("stale status guard: want errConflict, got %v", err)
	}
	out, err := repo.AppendMessage(ctx, c.ID, &c.UserID, StatusOpen, Message{ID: bson.NewObjectID(), Body: "hi"},
		bson.D{{Key: "updatedAt", Value: time.Now().UTC()}})
	if err != nil || len(out.Messages) != 1 {
		t.Fatalf("guarded append: %v %+v", err, out)
	}
	if _, _, err := repo.Counts(ctx); err != nil {
		t.Fatalf("counts: %v", err)
	}
}
