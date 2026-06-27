package order

import (
	"context"
	"fmt"
	"os"
	"time"
	_ "time/tzdata" // embed the IANA tz database so BUSINESS_TZ resolves regardless of the host OS

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// businessLocation is the timezone used to bucket day-grained dashboard figures
// ("today" revenue/orders, the daily revenue chart). It defaults to UTC; set
// BUSINESS_TZ to an IANA name (e.g. "Asia/Beirut") so day boundaries line up
// with the operating market rather than UTC. An unparseable value falls back to
// UTC so the dashboard never fails to load.
var businessLocation = loadBusinessLocation()

func loadBusinessLocation() *time.Location {
	if name := os.Getenv("BUSINESS_TZ"); name != "" {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	return time.UTC
}

// Repository defines persistence operations for the Order entity.
type Repository interface {
	FindByID(ctx context.Context, id bson.ObjectID) (*Order, error)
	FindByUserID(ctx context.Context, userID bson.ObjectID) ([]*Order, error)
	FindByIdempotencyKey(ctx context.Context, userID bson.ObjectID, key string) (*Order, error)
	Create(ctx context.Context, order *Order) error
	UpdateStatus(ctx context.Context, id bson.ObjectID, status OrderStatus) error
	UpdateFulfillment(ctx context.Context, id bson.ObjectID, status OrderStatus, fulfillment Fulfillment) error
	ListAll(ctx context.Context, f OrderFilter, p pagination.Params) ([]*Order, int64, error)

	// Admin dashboard aggregations.
	DayStats(ctx context.Context, day time.Time) (count int, revenue float64, err error)
	CountByStatus(ctx context.Context, status OrderStatus) (int64, error)
	CountPendingTransfers(ctx context.Context) (int64, error)
	FulfillmentBreakdown(ctx context.Context) (map[string]int, error)
	RevenueSeries(ctx context.Context, rng string) (labels []string, series []float64, err error)
}

// OrderFilter narrows an admin order listing. Zero-valued fields are ignored.
// OrderID (set when a search term parses as a valid ObjectID) and the
// search-mode fields (UserIDs + SearchRaw) are mutually exclusive: the handler
// sets one or the other.
type OrderFilter struct {
	Status          OrderStatus
	PaymentMethod   PaymentMethod
	FulfillmentType string
	OrderID         *bson.ObjectID // exact order-id match (search parsed as hex)
	UserIDs         []bson.ObjectID // email-search matches (search mode)
	SearchRaw       string          // raw term; matched against deliveredCode in search mode
}

// build assembles the MongoDB filter document for f.
func (f OrderFilter) build() bson.D {
	filter := bson.D{}
	if f.Status != "" {
		filter = append(filter, bson.E{Key: "status", Value: f.Status})
	}
	if f.PaymentMethod != "" {
		filter = append(filter, bson.E{Key: "paymentMethod", Value: f.PaymentMethod})
	}
	if f.FulfillmentType != "" {
		filter = append(filter, bson.E{Key: "items.fulfillmentType", Value: f.FulfillmentType})
	}
	if f.OrderID != nil {
		filter = append(filter, bson.E{Key: "_id", Value: *f.OrderID})
	} else if f.SearchRaw != "" {
		// Non-hex search: match a delivered code exactly, or any order owned by
		// a user whose email matched the term.
		or := bson.A{bson.D{{Key: "fulfillment.deliveredCode", Value: f.SearchRaw}}}
		if len(f.UserIDs) > 0 {
			or = append(or, bson.D{{Key: "userId", Value: bson.D{{Key: "$in", Value: f.UserIDs}}}})
		}
		filter = append(filter, bson.E{Key: "$or", Value: or})
	}
	return filter
}

// MongoRepository is a MongoDB-backed implementation of Repository.
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository using the given collection.
func NewMongoRepository(col *mongo.Collection) *MongoRepository {
	return &MongoRepository{collection: col}
}

// EnsureIndexes creates the indexes the order module relies on: a lookup index
// on (userId, createdAt) for order history, and a partial-unique index on
// (userId, idempotencyKey) that guards against duplicate submissions while
// allowing the many orders that carry no key.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	orders := db.Collection("orders")
	_, err := orders.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
		// Admin dashboard + list filters match on status and bucket by createdAt
		// (DayStats, RevenueSeries, CountByStatus, the status-filtered listing);
		// none of these carry a userId prefix, so they need their own index.
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "createdAt", Value: -1}}},
		{
			Keys: bson.D{{Key: "userId", Value: 1}, {Key: "idempotencyKey", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "idempotencyKey", Value: bson.D{{Key: "$exists", Value: true}}}}),
		},
	})
	return err
}

// FindByID retrieves a single order by ObjectID, returning ErrNotFound when absent.
func (r *MongoRepository) FindByID(ctx context.Context, id bson.ObjectID) (*Order, error) {
	var o Order
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&o)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

// FindByUserID returns a user's orders, newest first.
func (r *MongoRepository) FindByUserID(ctx context.Context, userID bson.ObjectID) ([]*Order, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.collection.Find(ctx, bson.D{{Key: "userId", Value: userID}}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*Order{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FindByIdempotencyKey returns the order a user previously created under key, or
// ErrNotFound when none exists.
func (r *MongoRepository) FindByIdempotencyKey(ctx context.Context, userID bson.ObjectID, key string) (*Order, error) {
	var o Order
	err := r.collection.FindOne(ctx, bson.D{
		{Key: "userId", Value: userID},
		{Key: "idempotencyKey", Value: key},
	}).Decode(&o)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

// Create inserts a new order, stamping CreatedAt/UpdatedAt. A duplicate
// (userId, idempotencyKey) violation is surfaced as ErrConflict so the caller
// can re-fetch the winning order.
func (r *MongoRepository) Create(ctx context.Context, order *Order) error {
	now := time.Now().UTC()
	if order.ID.IsZero() {
		order.ID = bson.NewObjectID()
	}
	order.CreatedAt = now
	order.UpdatedAt = now
	if order.Items == nil {
		order.Items = []OrderItem{}
	}
	if order.Fulfillment.StatusTimeline == nil {
		order.Fulfillment.StatusTimeline = []TimelineEvent{}
	}

	if _, err := r.collection.InsertOne(ctx, order); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return apperrors.ErrConflict
		}
		return err
	}
	return nil
}

// UpdateStatus sets the order's status and refreshes UpdatedAt.
func (r *MongoRepository) UpdateStatus(ctx context.Context, id bson.ObjectID, status OrderStatus) error {
	res, err := r.collection.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: status},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// UpdateFulfillment sets the order's status and fulfillment in a single update.
func (r *MongoRepository) UpdateFulfillment(ctx context.Context, id bson.ObjectID, status OrderStatus, fulfillment Fulfillment) error {
	res, err := r.collection.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: status},
			{Key: "fulfillment", Value: fulfillment},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// ListAll returns a paginated slice of orders matching f (admin), newest first.
func (r *MongoRepository) ListAll(ctx context.Context, f OrderFilter, p pagination.Params) ([]*Order, int64, error) {
	filter := f.build()
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip(pagination.Skip(p)).
		SetLimit(int64(p.Limit)).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	out := []*Order{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// DayStats returns the count and summed total of completed orders created within
// the business-timezone day that contains `day` (i.e. [local midnight, next
// local midnight)). See businessLocation / BUSINESS_TZ.
func (r *MongoRepository) DayStats(ctx context.Context, day time.Time) (int, float64, error) {
	d := day.In(businessLocation)
	start := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, businessLocation)
	end := start.AddDate(0, 0, 1)
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "status", Value: OrderStatusCompleted},
			{Key: "createdAt", Value: bson.D{{Key: "$gte", Value: start}, {Key: "$lt", Value: end}}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "revenue", Value: bson.D{{Key: "$sum", Value: "$total"}}},
		}}},
	}
	cur, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, 0, err
	}
	defer cur.Close(ctx)
	var rows []struct {
		Count   int     `bson:"count"`
		Revenue float64 `bson:"revenue"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return 0, 0, err
	}
	if len(rows) == 0 {
		return 0, 0, nil
	}
	return rows[0].Count, rows[0].Revenue, nil
}

// CountByStatus returns the number of orders in the given status.
func (r *MongoRepository) CountByStatus(ctx context.Context, status OrderStatus) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.D{{Key: "status", Value: status}})
}

// CountPendingTransfers counts money-transfer orders awaiting manual completion
// (status processing with a transfer line item). Both transfer and
// account_credit orders park in `processing`, so a plain status count would
// over-report; this scopes it to actual transfers.
func (r *MongoRepository) CountPendingTransfers(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.D{
		{Key: "status", Value: OrderStatusProcessing},
		{Key: "items.fulfillmentType", Value: "transfer"},
	})
}

// FulfillmentBreakdown counts completed-order line-items grouped by fulfillment
// type (raw keys: "code" / "account_credit" / "transfer"). It is scoped to
// completed orders to stay consistent with the revenue figures it sits beside.
func (r *MongoRepository) FulfillmentBreakdown(ctx context.Context) (map[string]int, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "status", Value: OrderStatusCompleted}}}},
		{{Key: "$unwind", Value: "$items"}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$items.fulfillmentType"},
			{Key: "n", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	cur, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rows []struct {
		Type string `bson:"_id"`
		N    int    `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.Type] = row.N
	}
	return out, nil
}

// revBucket is one column of the revenue chart: a [start, end) window in the
// business timezone plus its display label.
type revBucket struct {
	label string
	start time.Time // inclusive
	end   time.Time // exclusive
}

// revenueBuckets returns the MongoDB $dateToString format and the ordered
// buckets (oldest→newest) to render for a range, in loc. Only %Y/%m/%d format
// codes are used (portable across MongoDB and Cosmos vCore — ISO-week codes
// like %G/%V are not relied on); weekly groups by day and folds into weeks.
func revenueBuckets(now time.Time, rng string, loc *time.Location) (mongoFmt string, buckets []revBucket) {
	n := now.In(loc)
	switch rng {
	case "monthly":
		mongoFmt = "%Y-%m"
		first := time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, loc)
		for i := 11; i >= 0; i-- {
			s := first.AddDate(0, -i, 0)
			buckets = append(buckets, revBucket{label: s.Format("Jan"), start: s, end: s.AddDate(0, 1, 0)})
		}
	case "weekly":
		mongoFmt = "%Y-%m-%d"
		today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
		monday := today.AddDate(0, 0, -((int(today.Weekday())+6)%7)) // ISO week starts Monday
		for i := 11; i >= 0; i-- {
			s := monday.AddDate(0, 0, -7*i)
			_, wk := s.ISOWeek()
			buckets = append(buckets, revBucket{label: fmt.Sprintf("W%02d", wk), start: s, end: s.AddDate(0, 0, 7)})
		}
	default: // daily
		mongoFmt = "%Y-%m-%d"
		today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
		for i := 13; i >= 0; i-- {
			s := today.AddDate(0, 0, -i)
			buckets = append(buckets, revBucket{label: s.Format("Jan 2"), start: s, end: s.AddDate(0, 0, 1)})
		}
	}
	return mongoFmt, buckets
}

// RevenueSeries returns parallel label/value slices of completed-order revenue
// bucketed by the requested range ("daily" 14d / "weekly" 12w / "monthly" 12m)
// in the business timezone. Empty buckets are filled with zero so the series is
// always full-width.
func (r *MongoRepository) RevenueSeries(ctx context.Context, rng string) ([]string, []float64, error) {
	loc := businessLocation
	mongoFmt, buckets := revenueBuckets(time.Now(), rng, loc)
	matchStart := buckets[0].start
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "status", Value: OrderStatusCompleted},
			{Key: "createdAt", Value: bson.D{{Key: "$gte", Value: matchStart}}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$dateToString", Value: bson.D{
				{Key: "format", Value: mongoFmt},
				{Key: "date", Value: "$createdAt"},
				{Key: "timezone", Value: loc.String()},
			}}}},
			{Key: "revenue", Value: bson.D{{Key: "$sum", Value: "$total"}}},
		}}},
	}
	cur, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, nil, err
	}
	defer cur.Close(ctx)
	var rows []struct {
		Key     string  `bson:"_id"`
		Revenue float64 `bson:"revenue"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, nil, err
	}

	labels := make([]string, len(buckets))
	series := make([]float64, len(buckets))
	for i, b := range buckets {
		labels[i] = b.label
	}
	// Fold each grouped day/month into the bucket whose window contains it. This
	// keeps weekly correct without depending on Mongo ISO-week formatting.
	for _, row := range rows {
		t, err := time.ParseInLocation(mongoFmt2Go(mongoFmt), row.Key, loc)
		if err != nil {
			continue
		}
		for i, b := range buckets {
			if !t.Before(b.start) && t.Before(b.end) {
				series[i] += row.Revenue
				break
			}
		}
	}
	return labels, series, nil
}

// mongoFmt2Go maps the small set of $dateToString formats we use to their Go
// reference-time layouts for parsing grouped keys back into times.
func mongoFmt2Go(mongoFmt string) string {
	if mongoFmt == "%Y-%m" {
		return "2006-01"
	}
	return "2006-01-02"
}
