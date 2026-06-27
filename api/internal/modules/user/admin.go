package user

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// orderStats is the per-user order aggregate shown in the admin user views.
type orderStats struct {
	Orders int     `json:"orders"`
	Spent  float64 `json:"spent"`
}

// adminUserView is a user plus their order aggregate. Embedding *User promotes
// all of the user's JSON fields (PasswordHash stays hidden via its json:"-" tag)
// and just adds the order stats.
type adminUserView struct {
	*User
	orderStats
}

// adminUserDetail is a single user with order stats and their recent wallet
// ledger rows, for the detail page's overview/wallet tabs.
type adminUserDetail struct {
	*User
	orderStats
	Transactions []*wallet.WalletTransaction `json:"transactions"`
}

// maxDetailTransactions caps how many ledger rows the detail view returns.
const maxDetailTransactions = 10

// adminHandler serves the admin user endpoints. It owns the user repository, the
// wallet repository (for manual adjustments + ledger reads), and a raw handle on
// the orders collection — the order package imports user, so user must not import
// order; a direct aggregation over the collection keeps the dependency one-way.
type adminHandler struct {
	repo   Repository
	wallet wallet.Repository
	orders *mongo.Collection
}

// RegisterAdminRoutes mounts the admin user routes onto r (the /api/admin group,
// guarded by AdminOnly): paginated/filterable list, detail, role + status
// changes, and manual wallet adjustment (credit/debit with a reason logged to the
// ledger).
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	a := &adminHandler{
		repo:   NewMongoRepository(db),
		wallet: wallet.NewMongoRepository(db),
		orders: db.Collection("orders"),
	}

	r.Get("/users", a.list)
	r.Get("/users/{id}", a.detail)
	r.Put("/users/{id}/role", a.updateRole)
	r.Put("/users/{id}/status", a.updateStatus)
	r.Post("/users/{id}/wallet-adjust", a.walletAdjust)
}

// list handles GET /api/admin/users — paginated, newest first, with optional
// role / status filters and a search term (q) matching an id, email, or phone.
func (a *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	p := pagination.ParseParams(r)
	q := r.URL.Query()

	f := UserFilter{
		Role:   Role(strings.TrimSpace(q.Get("role"))),
		Status: Status(strings.TrimSpace(q.Get("status"))),
		Search: strings.TrimSpace(q.Get("q")),
	}

	users, total, err := a.repo.ListAll(r.Context(), f, p)
	if err != nil {
		response.InternalError(w)
		return
	}

	ids := make([]bson.ObjectID, len(users))
	for i, u := range users {
		ids[i] = u.ID
	}
	stats, err := a.orderStatsByUser(r.Context(), ids)
	if err != nil {
		response.InternalError(w)
		return
	}

	views := make([]adminUserView, len(users))
	for i, u := range users {
		views[i] = adminUserView{User: u, orderStats: stats[u.ID]}
	}
	response.OKWithMeta(w, views, pagination.CalcMeta(p, total))
}

// detail handles GET /api/admin/users/{id}.
func (a *adminHandler) detail(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	u, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
		a.writeRepoError(w, err)
		return
	}

	stats, err := a.orderStatsByUser(r.Context(), []bson.ObjectID{id})
	if err != nil {
		response.InternalError(w)
		return
	}

	txns, err := a.wallet.FindByUserID(r.Context(), id)
	if err != nil {
		response.InternalError(w)
		return
	}
	if len(txns) > maxDetailTransactions {
		txns = txns[:maxDetailTransactions]
	}

	response.OK(w, adminUserDetail{User: u, orderStats: stats[id], Transactions: txns})
}

// updateRole handles PUT /api/admin/users/{id}/role.
func (a *adminHandler) updateRole(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		Role Role `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	switch body.Role {
	case RoleCustomer, RoleReseller, RoleAdmin:
	default:
		response.BadRequest(w, "role must be one of customer, reseller, admin")
		return
	}
	u, err := a.repo.UpdateRole(r.Context(), id, body.Role)
	if err != nil {
		a.writeRepoError(w, err)
		return
	}
	response.OK(w, u)
}

// updateStatus handles PUT /api/admin/users/{id}/status.
func (a *adminHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		Status Status `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	switch body.Status {
	case StatusActive, StatusSuspended:
	default:
		response.BadRequest(w, "status must be one of active, suspended")
		return
	}
	u, err := a.repo.UpdateStatus(r.Context(), id, body.Status)
	if err != nil {
		a.writeRepoError(w, err)
		return
	}
	response.OK(w, u)
}

// walletAdjust handles POST /api/admin/users/{id}/wallet-adjust — a manual
// credit or debit. The balance change is atomic; the ledger row is a best-effort
// follow-up (logged, not failed, on insert error — consistent with the
// no-multi-document-transactions model).
func (a *adminHandler) walletAdjust(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		Direction string  `json:"direction"`
		Amount    float64 `json:"amount"`
		Reason    string  `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if body.Amount <= 0 {
		response.BadRequest(w, "amount must be greater than zero")
		return
	}

	var (
		newBal float64
		err    error
		signed float64
	)
	switch body.Direction {
	case "credit":
		newBal, err = a.wallet.Credit(r.Context(), id, body.Amount)
		signed = body.Amount
	case "debit":
		newBal, err = a.wallet.Debit(r.Context(), id, body.Amount)
		signed = -body.Amount
	default:
		response.BadRequest(w, "direction must be credit or debit")
		return
	}
	if err != nil {
		a.writeRepoError(w, err)
		return
	}

	// Record the adjustment in the immutable ledger (best-effort): the balance is
	// already updated, so a ledger insert failure is logged, not surfaced.
	if err := a.wallet.Create(r.Context(), &wallet.WalletTransaction{
		UserID:       id,
		Type:         wallet.TxTypeAdjustment,
		Amount:       signed,
		BalanceAfter: newBal,
		Method:       "admin",
		Ref:          strings.TrimSpace(body.Reason),
	}); err != nil {
		log.Printf("wallet-adjust: balance updated for %s but ledger insert failed: %v", id.Hex(), err)
	}
	response.OK(w, map[string]float64{"walletBalance": newBal})
}

// orderStatsByUser aggregates completed-order count + total spent per user for
// the given ids, in a single query against the orders collection. Users with no
// completed orders are simply absent from the map (zero value).
func (a *adminHandler) orderStatsByUser(ctx context.Context, ids []bson.ObjectID) (map[bson.ObjectID]orderStats, error) {
	out := map[bson.ObjectID]orderStats{}
	if len(ids) == 0 {
		return out, nil
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "userId", Value: bson.D{{Key: "$in", Value: ids}}},
			{Key: "status", Value: "completed"},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$userId"},
			{Key: "orders", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "spent", Value: bson.D{{Key: "$sum", Value: "$total"}}},
		}}},
	}
	cur, err := a.orders.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rows []struct {
		ID     bson.ObjectID `bson:"_id"`
		Orders int           `bson:"orders"`
		Spent  float64       `bson:"spent"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = orderStats{Orders: row.Orders, Spent: row.Spent}
	}
	return out, nil
}

// parseID extracts the {id} path param as an ObjectID, writing a 404 and
// returning ok=false when it is malformed.
func parseID(w http.ResponseWriter, r *http.Request) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return bson.ObjectID{}, false
	}
	return id, true
}

// writeRepoError maps repository/wallet errors to HTTP responses (404 for a
// missing user, 400 for an error wrapping a bad-request — e.g. insufficient
// funds — and 500 otherwise).
func (a *adminHandler) writeRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	case errors.Is(err, apperrors.ErrBadRequest):
		msg := err.Error()
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			msg = appErr.Message
		}
		response.BadRequest(w, msg)
	default:
		response.InternalError(w)
	}
}
