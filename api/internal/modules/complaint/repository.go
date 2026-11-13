package complaint

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// Contact is the customer / admin identity the console shows next to a
// complaint (read from the users collection).
type Contact struct {
	ID     bson.ObjectID `bson:"_id" json:"id"`
	Name   string        `bson:"name,omitempty" json:"name,omitempty"`
	Email  string        `bson:"email,omitempty" json:"email,omitempty"`
	Phone  string        `bson:"phone,omitempty" json:"phone,omitempty"`
	Role   string        `bson:"role,omitempty" json:"-"`
	Status string        `bson:"status,omitempty" json:"status,omitempty"`
}

// OrderSummary is the linked-order card on the admin detail page.
type OrderSummary struct {
	ID        bson.ObjectID `bson:"_id" json:"id"`
	UserID    bson.ObjectID `bson:"userId" json:"-"`
	Status    string        `bson:"status" json:"status"`
	Total     float64       `bson:"total" json:"total"`
	Currency  string        `bson:"currency" json:"currency"`
	CreatedAt time.Time     `bson:"createdAt" json:"createdAt"`
	Items     []struct {
		Title struct {
			En string `bson:"en" json:"en"`
			Ar string `bson:"ar" json:"ar"`
		} `bson:"title" json:"title"`
		Denomination string `bson:"denomination" json:"denomination"`
		Qty          int    `bson:"qty" json:"qty"`
	} `bson:"items" json:"items"`
}

// AdminFilter narrows the admin list. Assignee is an admin id hex, "none"
// (unassigned), or "" (any).
type AdminFilter struct {
	Status   string
	Category string
	Assignee string
	Search   string
}

// Counts is the per-status tally behind the console tabs + nav badge.
type Counts struct {
	Open       int64 `json:"open"`
	InProgress int64 `json:"in_progress"`
	Resolved   int64 `json:"resolved"`
	Closed     int64 `json:"closed"`
}

// Repository persists complaints and reads the users/orders it references.
type Repository interface {
	// NextNumber allocates the next human-readable number atomically (a counter
	// document + $inc, never count()+1).
	NextNumber(ctx context.Context) (int, error)
	Insert(ctx context.Context, c *Complaint) error
	Get(ctx context.Context, id bson.ObjectID) (*Complaint, error)
	ListForUser(ctx context.Context, userID bson.ObjectID, p pagination.Params) ([]Complaint, int64, error)
	CountOpenForUser(ctx context.Context, userID bson.ObjectID) (int64, error)
	// AppendMessage pushes msg (and applies set) only while the complaint is
	// still in expectStatus, belongs to owner (when non-nil), and has room in
	// its thread — one guarded FindOneAndUpdate. errConflict when the guard
	// matches nothing; returns the updated document.
	AppendMessage(ctx context.Context, id bson.ObjectID, owner *bson.ObjectID, expectStatus string, msg Message, set bson.D) (*Complaint, error)
	// UpdateGuarded applies set while the complaint is still in expectStatus
	// (errConflict otherwise); returns the updated document.
	UpdateGuarded(ctx context.Context, id bson.ObjectID, expectStatus string, set bson.D, unset bson.D) (*Complaint, error)
	AdminList(ctx context.Context, f AdminFilter, p pagination.Params) ([]Complaint, int64, error)
	Counts(ctx context.Context) (Counts, int64, error)
	Users(ctx context.Context, ids []bson.ObjectID) (map[bson.ObjectID]Contact, error)
	Order(ctx context.Context, id bson.ObjectID) (*OrderSummary, error)
}

// MongoRepository is the MongoDB-backed Repository.
type MongoRepository struct {
	col      *mongo.Collection
	counters *mongo.Collection
	users    *mongo.Collection
	orders   *mongo.Collection
}

// NewMongoRepository builds the repository.
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		col:      db.Collection("complaints"),
		counters: db.Collection("counters"),
		users:    db.Collection("users"),
		orders:   db.Collection("orders"),
	}
}

// EnsureIndexes creates the complaint indexes: the customer's list, the admin
// queue, and the unique human-readable number.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("complaints").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "updatedAt", Value: -1}}},
		{Keys: bson.D{{Key: "number", Value: 1}}, Options: options.Index().SetUnique(true)},
	})
	return err
}

const counterID = "complaints"

// NextNumber implements Repository.
func (r *MongoRepository) NextNumber(ctx context.Context) (int, error) {
	var doc struct {
		Seq int `bson:"seq"`
	}
	err := r.counters.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: counterID}},
		bson.D{{Key: "$inc", Value: bson.D{{Key: "seq", Value: 1}}}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		return 0, err
	}
	return numberBase + doc.Seq, nil
}

// Insert implements Repository.
func (r *MongoRepository) Insert(ctx context.Context, c *Complaint) error {
	_, err := r.col.InsertOne(ctx, c)
	return err
}

// Get implements Repository.
func (r *MongoRepository) Get(ctx context.Context, id bson.ObjectID) (*Complaint, error) {
	var c Complaint
	if err := r.col.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&c); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errNotFound
		}
		return nil, err
	}
	return &c, nil
}

// listProjection keeps list rows light: no message thread.
var listProjection = bson.D{{Key: "messages", Value: 0}}

// ListForUser implements Repository (newest first).
func (r *MongoRepository) ListForUser(ctx context.Context, userID bson.ObjectID, p pagination.Params) ([]Complaint, int64, error) {
	filter := bson.D{{Key: "userId", Value: userID}}
	return r.find(ctx, filter, bson.D{{Key: "createdAt", Value: -1}}, p)
}

// CountOpenForUser implements Repository.
func (r *MongoRepository) CountOpenForUser(ctx context.Context, userID bson.ObjectID) (int64, error) {
	return r.col.CountDocuments(ctx, bson.D{
		{Key: "userId", Value: userID},
		{Key: "status", Value: bson.D{{Key: "$in", Value: bson.A{StatusOpen, StatusInProgress}}}},
	})
}

// AppendMessage implements Repository. "messages.<max-1>" not existing means
// the thread still has room (a single-document, index-free check).
func (r *MongoRepository) AppendMessage(ctx context.Context, id bson.ObjectID, owner *bson.ObjectID, expectStatus string, msg Message, set bson.D) (*Complaint, error) {
	filter := bson.D{
		{Key: "_id", Value: id},
		{Key: "status", Value: expectStatus},
		{Key: "messages." + strconv.Itoa(maxMessages-1), Value: bson.D{{Key: "$exists", Value: false}}},
	}
	if owner != nil {
		filter = append(filter, bson.E{Key: "userId", Value: *owner})
	}
	update := bson.D{{Key: "$push", Value: bson.D{{Key: "messages", Value: msg}}}}
	if len(set) > 0 {
		update = append(update, bson.E{Key: "$set", Value: set})
	}
	return r.findOneAndUpdate(ctx, filter, update)
}

// UpdateGuarded implements Repository.
func (r *MongoRepository) UpdateGuarded(ctx context.Context, id bson.ObjectID, expectStatus string, set bson.D, unset bson.D) (*Complaint, error) {
	filter := bson.D{{Key: "_id", Value: id}, {Key: "status", Value: expectStatus}}
	update := bson.D{}
	if len(set) > 0 {
		update = append(update, bson.E{Key: "$set", Value: set})
	}
	if len(unset) > 0 {
		update = append(update, bson.E{Key: "$unset", Value: unset})
	}
	return r.findOneAndUpdate(ctx, filter, update)
}

func (r *MongoRepository) findOneAndUpdate(ctx context.Context, filter, update bson.D) (*Complaint, error) {
	var c Complaint
	err := r.col.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, errConflict
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// AdminList implements Repository (most recently active first).
func (r *MongoRepository) AdminList(ctx context.Context, f AdminFilter, p pagination.Params) ([]Complaint, int64, error) {
	filter := bson.D{}
	if f.Status != "" {
		filter = append(filter, bson.E{Key: "status", Value: f.Status})
	}
	if f.Category != "" {
		filter = append(filter, bson.E{Key: "category", Value: f.Category})
	}
	switch f.Assignee {
	case "":
	case "none":
		filter = append(filter, bson.E{Key: "assigneeId", Value: bson.D{{Key: "$exists", Value: false}}})
	default:
		id, err := bson.ObjectIDFromHex(f.Assignee)
		if err != nil {
			return []Complaint{}, 0, nil
		}
		filter = append(filter, bson.E{Key: "assigneeId", Value: id})
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		or, err := r.searchClause(ctx, s)
		if err != nil {
			return nil, 0, err
		}
		if len(or) == 0 {
			return []Complaint{}, 0, nil
		}
		filter = append(filter, bson.E{Key: "$or", Value: or})
	}
	return r.find(ctx, filter, bson.D{{Key: "updatedAt", Value: -1}}, p)
}

// numberRx matches a complaint number as staff type it: "1042", "C-1042", "c1042".
var numberRx = regexp.MustCompile(`^[cC]-?(\d+)$|^(\d+)$`)

// searchClause turns the console search box into an $or: the complaint number,
// and/or the ids of customers whose email or phone matches (capped).
func (r *MongoRepository) searchClause(ctx context.Context, s string) (bson.A, error) {
	or := bson.A{}
	if m := numberRx.FindStringSubmatch(s); m != nil {
		digits := m[1] + m[2]
		if n, err := strconv.Atoi(digits); err == nil {
			or = append(or, bson.D{{Key: "number", Value: n}})
		}
	}
	rx := bson.D{{Key: "$regex", Value: regexp.QuoteMeta(s)}, {Key: "$options", Value: "i"}}
	cur, err := r.users.Find(ctx, bson.D{{Key: "$or", Value: bson.A{
		bson.D{{Key: "email", Value: rx}},
		bson.D{{Key: "phone", Value: rx}},
		bson.D{{Key: "name", Value: rx}},
	}}}, options.Find().SetProjection(bson.D{{Key: "_id", Value: 1}}).SetLimit(100))
	if err != nil {
		return nil, err
	}
	var ids []struct {
		ID bson.ObjectID `bson:"_id"`
	}
	if err := cur.All(ctx, &ids); err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		in := make(bson.A, 0, len(ids))
		for _, u := range ids {
			in = append(in, u.ID)
		}
		or = append(or, bson.D{{Key: "userId", Value: bson.D{{Key: "$in", Value: in}}}})
	}
	return or, nil
}

func (r *MongoRepository) find(ctx context.Context, filter, sort bson.D, p pagination.Params) ([]Complaint, int64, error) {
	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	cur, err := r.col.Find(ctx, filter, options.Find().
		SetSort(sort).SetSkip(pagination.Skip(p)).SetLimit(int64(p.Limit)).
		SetProjection(listProjection))
	if err != nil {
		return nil, 0, err
	}
	out := []Complaint{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// Counts implements Repository: one count per status plus the awaiting-reply
// tally among open/in-progress complaints.
func (r *MongoRepository) Counts(ctx context.Context) (Counts, int64, error) {
	var c Counts
	for status, dst := range map[string]*int64{
		StatusOpen: &c.Open, StatusInProgress: &c.InProgress,
		StatusResolved: &c.Resolved, StatusClosed: &c.Closed,
	} {
		n, err := r.col.CountDocuments(ctx, bson.D{{Key: "status", Value: status}})
		if err != nil {
			return c, 0, err
		}
		*dst = n
	}
	awaiting, err := r.col.CountDocuments(ctx, bson.D{
		{Key: "status", Value: bson.D{{Key: "$in", Value: bson.A{StatusOpen, StatusInProgress}}}},
		{Key: "$expr", Value: bson.D{{Key: "$gt", Value: bson.A{
			"$lastCustomerMessageAt",
			bson.D{{Key: "$ifNull", Value: bson.A{"$lastAdminMessageAt", time.Unix(0, 0).UTC()}}},
		}}}},
	})
	if err != nil {
		// $expr support varies across Mongo-compatible backends; the badge
		// degrades to the open count rather than failing the page.
		return c, c.Open, nil
	}
	return c, awaiting, nil
}

// Users implements Repository.
func (r *MongoRepository) Users(ctx context.Context, ids []bson.ObjectID) (map[bson.ObjectID]Contact, error) {
	out := map[bson.ObjectID]Contact{}
	if len(ids) == 0 {
		return out, nil
	}
	cur, err := r.users.Find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}},
		options.Find().SetProjection(bson.D{
			{Key: "name", Value: 1}, {Key: "email", Value: 1}, {Key: "phone", Value: 1},
			{Key: "role", Value: 1}, {Key: "status", Value: 1},
		}))
	if err != nil {
		return nil, err
	}
	var rows []Contact
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, c := range rows {
		out[c.ID] = c
	}
	return out, nil
}

// Order implements Repository (errNotFound when missing).
func (r *MongoRepository) Order(ctx context.Context, id bson.ObjectID) (*OrderSummary, error) {
	var o OrderSummary
	err := r.orders.FindOne(ctx, bson.D{{Key: "_id", Value: id}}, options.FindOne().SetProjection(bson.D{
		{Key: "userId", Value: 1}, {Key: "status", Value: 1}, {Key: "total", Value: 1},
		{Key: "currency", Value: 1}, {Key: "createdAt", Value: 1},
		{Key: "items.title", Value: 1}, {Key: "items.denomination", Value: 1}, {Key: "items.qty", Value: 1},
	})).Decode(&o)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}
