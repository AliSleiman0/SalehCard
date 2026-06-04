package mongoplatform

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Connect creates a MongoDB client, pings the server, and returns the client and database.
func Connect(uri, dbName string) (*mongo.Client, *mongo.Database, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx, nil); err != nil {
		return nil, nil, err
	}

	db := client.Database(dbName)
	return client, db, nil
}

// Disconnect closes the MongoDB client connection.
func Disconnect(ctx context.Context, client *mongo.Client) {
	client.Disconnect(ctx)
}
