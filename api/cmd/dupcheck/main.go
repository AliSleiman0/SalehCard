// Command dupcheck is a READ-ONLY integrity probe: it reports the total product
// count, any duplicate legacyId documents, and the raw title/label state of a
// few sample legacyIds — to explain why a loadseed write may not "take" on some
// docs (typically duplicate legacyIds: the upsert updates one, reads hit another).
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	client, db, err := mongoplatform.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("connect mongo: %v", err)
	}
	defer mongoplatform.Disconnect(context.Background(), client)

	col := db.Collection("products")

	total, err := col.CountDocuments(ctx, bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("total product docs: %d\n", total)

	// Group by legacyId, keep those appearing more than once.
	cur, err := col.Aggregate(ctx, mongo_pipeline())
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(ctx)

	dupes := 0
	extraDocs := 0
	fmt.Println("\nduplicate legacyIds (count > 1):")
	for cur.Next(ctx) {
		var r struct {
			ID    int `bson:"_id"`
			Count int `bson:"count"`
		}
		if err := cur.Decode(&r); err != nil {
			log.Fatal(err)
		}
		dupes++
		extraDocs += r.Count - 1
		if dupes <= 40 {
			fmt.Printf("  legacyId %d  ×%d\n", r.ID, r.Count)
		}
	}
	if dupes == 0 {
		fmt.Println("  (none)")
	}
	fmt.Printf("\nlegacyIds with duplicates: %d   extra (redundant) docs: %d\n", dupes, extraDocs)

	// Sample the known blank-title legacyIds: decode title + updatedAt so we can
	// see whether the load actually touched the doc (recent updatedAt) yet left
	// the title unset.
	sample := []int{741, 1002, 1017}
	fmt.Println("\nsample docs for known blank-title legacyIds:")
	for _, id := range sample {
		var d struct {
			ID    bson.ObjectID `bson:"_id"`
			Title struct {
				En string `bson:"en"`
				Ar string `bson:"ar"`
				Tr string `bson:"tr"`
			} `bson:"title"`
			InputFields []bson.M  `bson:"inputFields"`
			UpdatedAt   time.Time `bson:"updatedAt"`
			CreatedAt   time.Time `bson:"createdAt"`
		}
		err := col.FindOne(ctx, bson.D{{Key: "legacyId", Value: id}}).Decode(&d)
		if err != nil {
			fmt.Printf("  legacyId %d → %v\n", id, err)
			continue
		}
		fmt.Printf("  legacyId %d  en=%q ar=%q tr=%q  inputFields=%d  updatedAt=%s  createdAt=%s\n",
			id, d.Title.En, d.Title.Ar, d.Title.Tr, len(d.InputFields),
			d.UpdatedAt.Format(time.RFC3339), d.CreatedAt.Format(time.RFC3339))
	}
}

func mongo_pipeline() []bson.D {
	return []bson.D{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$legacyId"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		{{Key: "$match", Value: bson.D{{Key: "count", Value: bson.D{{Key: "$gt", Value: 1}}}}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
	}
}
