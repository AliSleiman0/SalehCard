package order

import (
	"context"
	"fmt"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/timeutil"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// businessLocation is the timezone used to bucket day-grained dashboard figures
// ("today" revenue/orders, the daily revenue chart) — the single source of truth
// lives in pkg/timeutil so the wallet KPI shares the same day boundaries.
var businessLocation = timeutil.BusinessLocation()

// Repository defines persistence operations for the Order entity.
type Repository interface {
	FindByID(ctx context.Context, id bson.ObjectID) (*Order, error)
	FindByUserID(ctx context.Context, userID bson.ObjectID) ([]*Order, error)
	FindByIdempotencyKey(ctx context.Context, userID bson.ObjectID, key string) (*Order, error)
	Create(ctx context.Context, order *Order) error
	UpdateStatus(ctx context.Context, id bson.ObjectID, status OrderStatus) error
	UpdateFulfillment(ctx context.Context, id bson.ObjectID, status OrderStatus, fulfillment Fulfillment) error
	// TransitionStatus atomically moves an order from one of `from` to `to`,
	// appending a timeline event (and optionally setting extra fields). It
	// returns the PRE-transition document — callers need the original status,
	// total, and payment method for compensation. ErrConflict when the order is
	// not in one of the `from` states (e.g. an already-refunded order), which
	// makes the single document write the double-refund lock.
	TransitionStatus(ctx context.Context, id bson.ObjectID, from []OrderStatus, to OrderStatus, event TimelineEvent, extraSet bson.D) (*Order, error)
	ListAll(ctx context.Context, f OrderFilter, p pagination.Params) ([]*Order, int64, error)

	// Admin dashboard aggregations.
	DayStats(ctx context.Context, day time.Time) (count int, revenue float64, err error)
	CountByStatus(ctx context.Context, status OrderStatus) (int64, error)
	CountPendingTransfers(ctx context.Context) (int64, error)
	FulfillmentBreakdown(ctx context.Context) (map[string]int, error)
	RevenueSeries(ctx context.Context, rng string) (labels []string, series []float64, err error)
	KpiSparkSeries(ctx context.Context) (revenue []float64, orders []int, err error)
	RevenueSummary(ctx context.Context) (RevenueSummary, error)
}

// LabelValue is one labeled bucket of a revenue breakdown (e.g. a payment
// method, currency, or category and its summed revenue).
type LabelValue struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// RevenueSummary is the admin finance rollup, all derived from the orders
// collection. Money figures are summed completed-order totals; RefundRate is a
// fraction in [0,1] (refunded / (completed + refunded)).
type RevenueSummary struct {
	TotalRevenue float64      `json:"totalRevenue"`
	MonthRevenue float64      `json:"monthRevenue"`
	RefundRate   float64      `json:"refundRate"`
	ByMethod     []LabelValue `json:"byMethod"`
	ByCategory   []LabelValue `json:"byCategory"`
	ByCurrency   []LabelValue `json:"byCurrency"`
}

// OrderFilter narrows an admin order listing. Zero-valued fields are ignored.
// OrderID (set when a search term parses as a valid ObjectID) and the
// search-mode fields (UserIDs + SearchRaw) are mutually exclusive: the handler
// sets one or the other.
type OrderFilter struct {
	Status          OrderStatus
	PaymentMethod   PaymentMethod
	FulfillmentType string
	OrderID         *bson.ObjectID  // exact order-id match (search parsed as hex)
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

// TransitionStatus atomically moves an order from one of `from` to `to` in a
// single FindOneAndUpdate, appending event to the fulfillment timeline and
// applying extraSet (may be nil). Returns the pre-transition document;
// ErrConflict when no document matches (wrong current status), ErrNotFound
// when the order does not exist at all.
func (r *MongoRepository) TransitionStatus(ctx context.Context, id bson.ObjectID, from []OrderStatus, to OrderStatus, event TimelineEvent, extraSet bson.D) (*Order, error) {
	set := bson.D{
		{Key: "status", Value: to},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}
	set = append(set, extraSet...)

	var before Order
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{
			{Key: "_id", Value: id},
			{Key: "status", Value: bson.D{{Key: "$in", Value: from}}},
		},
		bson.D{
			{Key: "$set", Value: set},
			{Key: "$push", Value: bson.D{{Key: "fulfillment.statusTimeline", Value: event}}},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.Before),
	).Decode(&before)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Distinguish "order missing" from "order in the wrong state".
			if _, ferr := r.FindByID(ctx, id); ferr != nil {
				return nil, apperrors.ErrNotFound
			}
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return &before, nil
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

// SoldByProduct returns a map of product id (hex) -> total units sold, summed
// from the line-item quantities of completed orders. Products with no completed
// sales are simply absent from the map (the caller defaults them to 0). It is
// a read model for the admin product list's "Sold" column; kept off the Service
// interface (the admin handler holds the concrete repo) to avoid touching the
// order Service fakes in order_test.go.
func (r *MongoRepository) SoldByProduct(ctx context.Context) (map[string]int, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "status", Value: OrderStatusCompleted}}}},
		{{Key: "$unwind", Value: "$items"}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$items.productId"},
			{Key: "qty", Value: bson.D{{Key: "$sum", Value: "$items.qty"}}},
		}}},
	}
	cur, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rows []struct {
		ProductID bson.ObjectID `bson:"_id"`
		Qty       int           `bson:"qty"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.ProductID.Hex()] = row.Qty
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
		monday := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7)) // ISO week starts Monday
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

// KpiSparkSeries returns parallel last-14-day daily series of completed-order
// revenue and order count, in the business timezone, zero-filled. It feeds the
// "today's revenue" / "today's orders" KPI sparklines on the admin dashboard.
func (r *MongoRepository) KpiSparkSeries(ctx context.Context) (revenue []float64, orders []int, err error) {
	loc := businessLocation
	mongoFmt, buckets := revenueBuckets(time.Now(), "daily", loc)
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
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
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
		Count   int     `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, nil, err
	}

	revenue = make([]float64, len(buckets))
	orders = make([]int, len(buckets))
	for _, row := range rows {
		t, err := time.ParseInLocation(mongoFmt2Go(mongoFmt), row.Key, loc)
		if err != nil {
			continue
		}
		for i, b := range buckets {
			if !t.Before(b.start) && t.Before(b.end) {
				revenue[i] += row.Revenue
				orders[i] += row.Count
				break
			}
		}
	}
	return revenue, orders, nil
}

// RevenueSummary rolls up the admin finance figures from the orders collection:
// all-time and current-month completed revenue, the refund rate, and revenue
// broken down by payment method, currency, and product category. The
// month boundary uses the business timezone, consistent with DayStats.
func (r *MongoRepository) RevenueSummary(ctx context.Context) (RevenueSummary, error) {
	var out RevenueSummary

	// All-time completed revenue + count, in one pass.
	totalPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "status", Value: OrderStatusCompleted}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "revenue", Value: bson.D{{Key: "$sum", Value: "$total"}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	var totals []struct {
		Revenue float64 `bson:"revenue"`
		Count   int64   `bson:"count"`
	}
	if err := r.aggregateInto(ctx, totalPipeline, &totals); err != nil {
		return out, err
	}
	var completedCount int64
	if len(totals) > 0 {
		out.TotalRevenue = totals[0].Revenue
		completedCount = totals[0].Count
	}

	// Current-month completed revenue (business-timezone month boundary).
	now := time.Now().In(businessLocation)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, businessLocation)
	monthPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "status", Value: OrderStatusCompleted},
			{Key: "createdAt", Value: bson.D{{Key: "$gte", Value: monthStart}}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "revenue", Value: bson.D{{Key: "$sum", Value: "$total"}}},
		}}},
	}
	var month []struct {
		Revenue float64 `bson:"revenue"`
	}
	if err := r.aggregateInto(ctx, monthPipeline, &month); err != nil {
		return out, err
	}
	if len(month) > 0 {
		out.MonthRevenue = month[0].Revenue
	}

	// Refund rate = refunded / (completed + refunded), guarded against /0.
	refundedCount, err := r.CountByStatus(ctx, OrderStatusRefunded)
	if err != nil {
		return out, err
	}
	if denom := completedCount + refundedCount; denom > 0 {
		out.RefundRate = float64(refundedCount) / float64(denom)
	}

	// Breakdowns over completed orders.
	if out.ByMethod, err = r.revenueGroup(ctx, "$paymentMethod", false); err != nil {
		return out, err
	}
	if out.ByCurrency, err = r.revenueGroup(ctx, "$currency", false); err != nil {
		return out, err
	}
	if out.ByCategory, err = r.revenueGroup(ctx, "$items.category", true); err != nil {
		return out, err
	}
	return out, nil
}

// revenueGroup sums completed-order revenue grouped by a single field, returning
// labeled buckets sorted highest-first. When perItem is true the orders are
// unwound to line items and revenue is summed as price*qty (used for category,
// which lives on the line item); otherwise the order total is summed by an
// order-level field (payment method, currency).
func (r *MongoRepository) revenueGroup(ctx context.Context, field string, perItem bool) ([]LabelValue, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "status", Value: OrderStatusCompleted}}}},
	}
	sum := bson.D{{Key: "$sum", Value: "$total"}}
	if perItem {
		pipeline = append(pipeline, bson.D{{Key: "$unwind", Value: "$items"}})
		sum = bson.D{{Key: "$sum", Value: bson.D{{Key: "$multiply", Value: bson.A{"$items.price", "$items.qty"}}}}}
	}
	pipeline = append(pipeline,
		bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: field}, {Key: "v", Value: sum}}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "v", Value: -1}}}},
	)
	var rows []struct {
		ID    string  `bson:"_id"`
		Value float64 `bson:"v"`
	}
	if err := r.aggregateInto(ctx, pipeline, &rows); err != nil {
		return nil, err
	}
	out := make([]LabelValue, 0, len(rows))
	for _, row := range rows {
		out = append(out, LabelValue{Label: row.ID, Value: row.Value})
	}
	return out, nil
}

// aggregateInto runs a pipeline and decodes all rows into dst (a pointer to a
// slice), closing the cursor.
func (r *MongoRepository) aggregateInto(ctx context.Context, pipeline mongo.Pipeline, dst any) error {
	cur, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)
	return cur.All(ctx, dst)
}

// mongoFmt2Go maps the small set of $dateToString formats we use to their Go
// reference-time layouts for parsing grouped keys back into times.
func mongoFmt2Go(mongoFmt string) string {
	if mongoFmt == "%Y-%m" {
		return "2006-01"
	}
	return "2006-01-02"
}
