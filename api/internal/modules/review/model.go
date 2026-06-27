package review

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Moderation states for a review. A review is created pending and an admin
// approves or rejects it; only approved reviews count toward a product's rating.
// An empty/legacy status is treated as pending on read.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// Review holds a customer rating and written comment for a product.
type Review struct {
	ID               bson.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID        bson.ObjectID `bson:"productId"     json:"productId"`
	UserID           bson.ObjectID `bson:"userId"        json:"userId"`
	Rating           int           `bson:"rating"        json:"rating"`
	Body             string        `bson:"body"          json:"body"`
	VerifiedPurchase bool          `bson:"verifiedPurchase" json:"verifiedPurchase"`
	Status           string        `bson:"status"        json:"status"`
	CreatedAt        time.Time     `bson:"createdAt"     json:"createdAt"`
}

// CreateReviewInput carries the data required to submit a new review.
type CreateReviewInput struct {
	ProductID bson.ObjectID `json:"productId"`
	Rating    int           `json:"rating"`
	Body      string        `json:"body"`
}
