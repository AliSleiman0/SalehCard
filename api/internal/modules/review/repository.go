package review

import (
	"context"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// ReviewRow is one review enriched with its product + reviewer for the admin
// moderation queue (a $lookup join, so the list renders without N+1 fetches).
type ReviewRow struct {
	Review      `bson:",inline"`
	ProductName string `bson:"productName" json:"productName"`
	Art         string `bson:"art"         json:"art"`
	UserName    string `bson:"userName"    json:"userName"`
}

// Repository defines persistence operations for Review entities.
type Repository interface {
	Create(ctx context.Context, review *Review) error
	Delete(ctx context.Context, id bson.ObjectID) error
	FindByID(ctx context.Context, id bson.ObjectID) (*Review, error)
	List(ctx context.Context, f ReviewFilter, p pagination.Params) ([]ReviewRow, int64, error)
	UpdateStatus(ctx context.Context, id bson.ObjectID, status string) (*Review, error)
	RecomputeProductRating(ctx context.Context, productID bson.ObjectID) error
}

// ReviewFilter narrows an admin review listing. Zero-valued fields are ignored.
type ReviewFilter struct {
	Status    string
	ProductID *bson.ObjectID
	Search    string
}

// build assembles the MongoDB match document for f. The pending bucket also
// matches legacy rows with no status (empty string).
func (f ReviewFilter) build() bson.D {
	and := bson.A{}
	switch f.Status {
	case StatusPending:
		and = append(and, bson.D{{Key: "status", Value: bson.D{{Key: "$in", Value: bson.A{StatusPending, ""}}}}})
	case StatusApproved, StatusRejected:
		and = append(and, bson.D{{Key: "status", Value: f.Status}})
	}
	if f.ProductID != nil {
		and = append(and, bson.D{{Key: "productId", Value: *f.ProductID}})
	}
	if f.Search != "" {
		rx := bson.D{{Key: "$regex", Value: regexp.QuoteMeta(f.Search)}, {Key: "$options", Value: "i"}}
		and = append(and, bson.D{{Key: "body", Value: rx}})
	}
	if len(and) == 0 {
		return bson.D{}
	}
	return bson.D{{Key: "$and", Value: and}}
}

// MongoRepository is a MongoDB-backed implementation of Repository. It also
// holds the products collection so it can recompute a product's denormalized
// rating after a review is moderated.
type MongoRepository struct {
	collection *mongo.Collection
	products   *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository over the reviews collection,
// using products for the rating recompute.
func NewMongoRepository(reviews, products *mongo.Collection) *MongoRepository {
	return &MongoRepository{collection: reviews, products: products}
}

// EnsureIndexes creates the supporting indexes on the reviews collection:
// by product (the recompute aggregation + storefront read), by status (the
// moderation queue filter), and by createdAt (the newest-first feed).
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("reviews").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "productId", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
	})
	return err
}

// Create inserts a new review, stamping the id/timestamp/default status.
func (r *MongoRepository) Create(ctx context.Context, review *Review) error {
	if review.ID.IsZero() {
		review.ID = bson.NewObjectID()
	}
	if review.CreatedAt.IsZero() {
		review.CreatedAt = time.Now().UTC()
	}
	if review.Status == "" {
		review.Status = StatusPending
	}
	_, err := r.collection.InsertOne(ctx, review)
	return err
}

// Delete removes a review by ObjectID, ErrNotFound when absent.
func (r *MongoRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	res, err := r.collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// FindByID retrieves a review by ObjectID, ErrNotFound when absent.
func (r *MongoRepository) FindByID(ctx context.Context, id bson.ObjectID) (*Review, error) {
	var rv Review
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&rv)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &rv, nil
}

// List returns a paginated slice of reviews matching f, newest first, each
// joined to its product (title/art) and reviewer (name).
func (r *MongoRepository) List(ctx context.Context, f ReviewFilter, p pagination.Params) ([]ReviewRow, int64, error) {
	match := f.build()
	total, err := r.collection.CountDocuments(ctx, match)
	if err != nil {
		return nil, 0, err
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$sort", Value: bson.D{{Key: "createdAt", Value: -1}}}},
		{{Key: "$skip", Value: pagination.Skip(p)}},
		{{Key: "$limit", Value: int64(p.Limit)}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "products"},
			{Key: "localField", Value: "productId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "productDoc"},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$productDoc"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "userId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "userDoc"},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$userDoc"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
	}
	cur, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var rows []struct {
		Review     `bson:",inline"`
		ProductDoc struct {
			Title struct {
				En string `bson:"en"`
			} `bson:"title"`
			Images []string `bson:"images"`
		} `bson:"productDoc"`
		UserDoc struct {
			Email string `bson:"email"`
			Phone string `bson:"phone"`
		} `bson:"userDoc"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, 0, err
	}

	out := make([]ReviewRow, len(rows))
	for i, row := range rows {
		art := ""
		if len(row.ProductDoc.Images) > 0 {
			art = row.ProductDoc.Images[0]
		}
		out[i] = ReviewRow{
			Review:      row.Review,
			ProductName: row.ProductDoc.Title.En,
			Art:         art,
			UserName:    userName(row.UserDoc.Email, row.UserDoc.Phone),
		}
	}
	return out, total, nil
}

// UpdateStatus sets a review's moderation status and returns the updated
// document. ErrNotFound when no review matches.
func (r *MongoRepository) UpdateStatus(ctx context.Context, id bson.ObjectID, status string) (*Review, error) {
	var rv Review
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: status}}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&rv)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &rv, nil
}

// RecomputeProductRating recalculates a product's denormalized rating
// (average + count) over its approved reviews and writes it atomically. With no
// approved reviews the rating resets to zero.
func (r *MongoRepository) RecomputeProductRating(ctx context.Context, productID bson.ObjectID) error {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "productId", Value: productID},
			{Key: "status", Value: StatusApproved},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "avg", Value: bson.D{{Key: "$avg", Value: "$rating"}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	cur, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)
	var rows []struct {
		Avg   float64 `bson:"avg"`
		Count int     `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return err
	}

	var avg float64
	var count int
	if len(rows) > 0 {
		avg = rows[0].Avg
		count = rows[0].Count
	}
	_, err = r.products.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: productID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "ratings.average", Value: avg},
			{Key: "ratings.count", Value: count},
		}}},
	)
	return err
}

// userName derives a display name from an email (local part) or phone. An
// orphaned review (no user) is "Unknown".
func userName(email, phone string) string {
	if email != "" {
		if at := strings.IndexByte(email, '@'); at > 0 {
			return email[:at]
		}
		return email
	}
	if phone != "" {
		return phone
	}
	return "Unknown"
}
