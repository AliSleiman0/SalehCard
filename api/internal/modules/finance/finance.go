// Package finance serves the admin finance endpoints: the wallet-ledger
// transactions feed and the revenue summary. It sits in its own package (like
// dashboard) so it can import both order and wallet — order imports wallet, so
// the handler cannot live in the wallet package without creating a cycle.
package finance

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/internal/modules/order"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Handler serves the admin finance endpoints. The transactions feed reads the
// wallet_transactions ledger (enriched with the owning user); the revenue
// summary is composed from the order repository's aggregations. The orders
// dependency is the order.Repository (not a raw collection) so the existing
// RevenueSeries/RevenueSummary aggregations are reused as-is.
type Handler struct {
	tx     *mongo.Collection // wallet_transactions — the ledger feed + top-ups KPI
	users  *mongo.Collection // enrich ledger rows with name/role; resolve search
	orders order.Repository  // revenue summary + revenue series
}

// RegisterAdminRoutes mounts the finance routes onto r (the /api/admin group,
// guarded by AdminOnly): the wallet-ledger transactions feed and the revenue
// summary. (USDT flows need no verification queue here: manual usdt top-ups go
// through the admin-approved topup_requests queue, and on-chain usdt_trc20
// settlements are confirmed automatically by the payment module's watcher and
// browsable at /api/admin/payments.)
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	h := &Handler{
		tx:     db.Collection("wallet_transactions"),
		users:  db.Collection("users"),
		orders: order.NewMongoRepository(db.Collection("orders")),
	}
	r.Get("/transactions", h.listTransactions)
	r.Get("/revenue-summary", h.revenueSummary)
}

// txUser is the slice of the owning user the ledger feed surfaces per row.
type txUser struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

// adminTxView is one ledger row enriched with its owning user, for the admin
// transactions feed.
type adminTxView struct {
	ID           bson.ObjectID `json:"id"`
	UserID       bson.ObjectID `json:"userId"`
	Type         wallet.TxType `json:"type"`
	Amount       float64       `json:"amount"`
	BalanceAfter float64       `json:"balanceAfter"`
	Method       string        `json:"method"`
	Ref          string        `json:"ref"`
	CreatedAt    time.Time     `json:"createdAt"`
	User         txUser        `json:"user"`
}

// listTransactions handles GET /api/admin/transactions — the wallet ledger,
// newest first, paginated, with optional type / method filters and a search
// term (q) matching a transaction id or an owning user's email/phone.
func (h *Handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p := pagination.ParseParams(r)
	q := r.URL.Query()

	match, err := h.txMatch(ctx,
		strings.TrimSpace(q.Get("q")),
		strings.TrimSpace(q.Get("type")),
		strings.TrimSpace(q.Get("method")),
	)
	if err != nil {
		response.InternalError(w)
		return
	}

	total, err := h.tx.CountDocuments(ctx, match)
	if err != nil {
		response.InternalError(w)
		return
	}

	// One pass: filter → newest-first page → join the owning user.
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$sort", Value: bson.D{{Key: "createdAt", Value: -1}}}},
		{{Key: "$skip", Value: pagination.Skip(p)}},
		{{Key: "$limit", Value: int64(p.Limit)}},
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
	cur, err := h.tx.Aggregate(ctx, pipeline)
	if err != nil {
		response.InternalError(w)
		return
	}
	defer cur.Close(ctx)

	var rows []struct {
		wallet.WalletTransaction `bson:",inline"`
		UserDoc                  struct {
			Email string `bson:"email"`
			Phone string `bson:"phone"`
			Role  string `bson:"role"`
		} `bson:"userDoc"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		response.InternalError(w)
		return
	}

	views := make([]adminTxView, len(rows))
	for i, row := range rows {
		views[i] = adminTxView{
			ID:           row.ID,
			UserID:       row.UserID,
			Type:         row.Type,
			Amount:       row.Amount,
			BalanceAfter: row.BalanceAfter,
			Method:       row.Method,
			Ref:          row.Ref,
			CreatedAt:    row.CreatedAt,
			User:         txUser{Name: userName(row.UserDoc.Email, row.UserDoc.Phone), Role: row.UserDoc.Role},
		}
	}
	response.OKWithMeta(w, views, pagination.CalcMeta(p, total))
}

// txMatch assembles the ledger filter from the type/method/search query params.
// A search term resolves to the ids of users whose email/phone match it (and,
// when the term is a valid ObjectID, the transaction or owner with that id); a
// term that matches nothing yields a filter that returns no rows.
func (h *Handler) txMatch(ctx context.Context, q, typ, method string) (bson.D, error) {
	match := bson.D{}
	if typ != "" {
		match = append(match, bson.E{Key: "type", Value: typ})
	}
	if method != "" {
		match = append(match, bson.E{Key: "method", Value: method})
	}
	if q == "" {
		return match, nil
	}

	or := bson.A{}
	ids, err := h.userIDsMatching(ctx, q)
	if err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		or = append(or, bson.D{{Key: "userId", Value: bson.D{{Key: "$in", Value: ids}}}})
	}
	if oid, err := bson.ObjectIDFromHex(q); err == nil {
		or = append(or, bson.D{{Key: "_id", Value: oid}}, bson.D{{Key: "userId", Value: oid}})
	}
	if len(or) == 0 {
		// The term matched no user and is not an id: return nothing rather than
		// silently ignoring the search. _id is never null, so this matches none.
		match = append(match, bson.E{Key: "_id", Value: nil})
		return match, nil
	}
	match = append(match, bson.E{Key: "$or", Value: or})
	return match, nil
}

// userIDsMatching returns the ids of users whose email or phone contains q
// (case-insensitive). The term is escaped so regex metacharacters are literal.
func (h *Handler) userIDsMatching(ctx context.Context, q string) ([]bson.ObjectID, error) {
	rx := bson.D{{Key: "$regex", Value: regexp.QuoteMeta(q)}, {Key: "$options", Value: "i"}}
	filter := bson.D{{Key: "$or", Value: bson.A{
		bson.D{{Key: "email", Value: rx}},
		bson.D{{Key: "phone", Value: rx}},
	}}}
	cur, err := h.users.Find(ctx, filter, options.Find().SetProjection(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rows []struct {
		ID bson.ObjectID `bson:"_id"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	ids := make([]bson.ObjectID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	return ids, nil
}

// revenueResponse is the payload for GET /api/admin/revenue-summary.
type revenueResponse struct {
	TotalRevenue float64            `json:"totalRevenue"`
	MonthRevenue float64            `json:"monthRevenue"`
	RefundRate   float64            `json:"refundRate"`
	WalletTopups float64            `json:"walletTopups"`
	Range        string             `json:"range"`
	Labels       []string           `json:"labels"`
	Series       []float64          `json:"series"`
	ByMethod     []order.LabelValue `json:"byMethod"`
	ByCategory   []order.LabelValue `json:"byCategory"`
	ByCurrency   []order.LabelValue `json:"byCurrency"`
}

// revenueSummary handles GET /api/admin/revenue-summary — KPIs and breakdowns
// derived from completed orders, the completed-revenue time series for the
// requested range, and the all-time wallet top-ups total from the ledger.
func (h *Handler) revenueSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sum, err := h.orders.RevenueSummary(ctx)
	if err != nil {
		response.InternalError(w)
		return
	}
	rng := r.URL.Query().Get("range")
	labels, series, err := h.orders.RevenueSeries(ctx, rng)
	if err != nil {
		response.InternalError(w)
		return
	}
	topups, err := h.walletTopups(ctx)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, revenueResponse{
		TotalRevenue: sum.TotalRevenue,
		MonthRevenue: sum.MonthRevenue,
		RefundRate:   sum.RefundRate,
		WalletTopups: topups,
		Range:        rng,
		Labels:       labels,
		Series:       series,
		ByMethod:     sum.ByMethod,
		ByCategory:   sum.ByCategory,
		ByCurrency:   sum.ByCurrency,
	})
}

// walletTopups returns the all-time sum of wallet top-ups from the ledger.
func (h *Handler) walletTopups(ctx context.Context) (float64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "type", Value: wallet.TxTypeTopUp}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "v", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
	}
	cur, err := h.tx.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)
	var rows []struct {
		V float64 `bson:"v"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].V, nil
}

// userName derives a display name from an email (local part) or phone, matching
// the admin frontend's derivation. An orphaned ledger row (no user) is "Unknown".
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
