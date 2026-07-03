package reseller

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// orderStats is the per-reseller order aggregate shown in the admin views.
type orderStats struct {
	Orders int     `json:"orders"`
	Volume float64 `json:"volume"`
}

// adminResellerView is a reseller (a user with role=reseller) plus their tier
// margin and order aggregate. Embedding *user.User promotes the user's JSON
// fields (resellerTier, walletBalance — the sub-balance — etc.).
type adminResellerView struct {
	*user.User
	Margin float64 `json:"margin"`
	orderStats
}

// adminResellerDetail adds the recent wallet ledger rows for the detail page.
type adminResellerDetail struct {
	*user.User
	Margin float64 `json:"margin"`
	orderStats
	Transactions []*wallet.WalletTransaction `json:"transactions"`
}

// tierWithCount is a tier definition plus its current reseller headcount.
type tierWithCount struct {
	*ResellerTier
	Count int64 `json:"count"`
}

// maxDetailTransactions caps how many ledger rows the detail view returns.
const maxDetailTransactions = 10

// adminHandler serves the admin reseller endpoints. Resellers are users with
// role=reseller, so the user repository backs the list/detail; the reseller
// repository owns the shared tier definitions; the wallet repository backs the
// sub-balance adjustments; and a raw orders handle backs the per-reseller stats
// (the order package imports user, so this package reads the collection directly
// to keep the dependency graph acyclic).
type adminHandler struct {
	tiers  Repository
	users  user.Repository
	wallet wallet.Repository
	orders *mongo.Collection
	rec    audit.Recorder
}

// RegisterAdminRoutes mounts the admin reseller routes onto r (the /api/admin
// group, guarded by AdminOnly): reseller list/detail, tier assignment, sub-
// balance adjustment, and tier-definition CRUD. Money mutations are recorded
// via rec.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder) {
	_ = EnsureIndexes(context.Background(), db)
	a := &adminHandler{
		tiers:  NewMongoRepository(db),
		users:  user.NewMongoRepository(db),
		wallet: wallet.NewMongoRepository(db),
		orders: db.Collection("orders"),
		rec:    rec,
	}

	r.Get("/resellers", a.list)
	r.Get("/resellers/{id}", a.detail)
	r.Put("/resellers/{id}/tier", a.assignTier)
	r.Post("/resellers/{id}/balance-adjust", a.balanceAdjust)
	r.Get("/resellers/{id}/prices", a.listPrices)
	r.Post("/resellers/{id}/prices", a.setPrice)
	r.Delete("/resellers/{id}/prices/{variantId}", a.deletePrice)

	r.Get("/reseller-tiers", a.listTiers)
	r.Post("/reseller-tiers", a.createTier)
	r.Put("/reseller-tiers/{id}", a.updateTier)
	r.Delete("/reseller-tiers/{id}", a.deleteTier)
}

// list handles GET /api/admin/resellers — paginated resellers (users with
// role=reseller), newest first, with optional tier / status filters and a search
// term (q) matching an id, email, or phone.
func (a *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	p := pagination.ParseParams(r)
	q := r.URL.Query()

	f := user.UserFilter{
		Role:         user.RoleReseller,
		Status:       user.Status(strings.TrimSpace(q.Get("status"))),
		ResellerTier: strings.TrimSpace(q.Get("tier")),
		Search:       strings.TrimSpace(q.Get("q")),
	}

	resellers, total, err := a.users.ListAll(r.Context(), f, p)
	if err != nil {
		response.InternalError(w)
		return
	}

	margins, err := a.tierMargins(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	ids := make([]bson.ObjectID, len(resellers))
	for i, u := range resellers {
		ids[i] = u.ID
	}
	stats, err := a.orderStatsByUser(r.Context(), ids)
	if err != nil {
		response.InternalError(w)
		return
	}

	views := make([]adminResellerView, len(resellers))
	for i, u := range resellers {
		views[i] = adminResellerView{
			User:       u,
			Margin:     margins[u.ResellerTier],
			orderStats: stats[u.ID],
		}
	}
	response.OKWithMeta(w, views, pagination.CalcMeta(p, total))
}

// detail handles GET /api/admin/resellers/{id}.
func (a *adminHandler) detail(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	u, err := a.users.FindByID(r.Context(), id)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	if u.Role != user.RoleReseller {
		response.NotFound(w)
		return
	}

	margins, err := a.tierMargins(r.Context())
	if err != nil {
		response.InternalError(w)
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

	response.OK(w, adminResellerDetail{
		User:         u,
		Margin:       margins[u.ResellerTier],
		orderStats:   stats[id],
		Transactions: txns,
	})
}

// assignTier handles PUT /api/admin/resellers/{id}/tier — sets (or clears) the
// reseller's tier. An empty tier unassigns; a non-empty tier must exist.
func (a *adminHandler) assignTier(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		Tier string `json:"tier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	tier := strings.TrimSpace(body.Tier)
	if tier != "" {
		if _, err := a.tiers.FindTierByName(r.Context(), tier); err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				response.BadRequest(w, "unknown tier: "+tier)
				return
			}
			response.InternalError(w)
			return
		}
	}
	u, err := a.users.UpdateResellerTier(r.Context(), id, tier)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	if u.Role != user.RoleReseller {
		response.NotFound(w)
		return
	}
	response.OK(w, u)
}

// balanceAdjust handles POST /api/admin/resellers/{id}/balance-adjust — a manual
// credit or debit of the reseller's wallet (their sub-balance). The balance
// change is atomic; the ledger row is mandatory: if the insert fails, the
// balance change is reversed (compensation, per the no-transactions model) and
// the request fails — money never moves without a ledger row.
func (a *adminHandler) balanceAdjust(w http.ResponseWriter, r *http.Request) {
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
		writeRepoError(w, err)
		return
	}

	if err := a.wallet.Create(r.Context(), &wallet.WalletTransaction{
		UserID:       id,
		Type:         wallet.TxTypeAdjustment,
		Amount:       signed,
		BalanceAfter: newBal,
		Method:       "admin",
		Ref:          strings.TrimSpace(body.Reason),
	}); err != nil {
		// Mandatory ledger: reverse the balance change so success always implies
		// a ledger row. A failed reversal is logged loudly for reconciliation.
		var undoErr error
		if signed > 0 {
			_, undoErr = a.wallet.Debit(r.Context(), id, body.Amount)
		} else {
			_, undoErr = a.wallet.Credit(r.Context(), id, body.Amount)
		}
		slog.Error("reseller balance-adjust: ledger insert failed; balance change reverted",
			"reseller", id.Hex(), "amount", signed, "ledgerError", err, "revertError", undoErr)
		response.Error(w, http.StatusInternalServerError, "LEDGER_WRITE_FAILED",
			"adjustment was reverted: the ledger row could not be written")
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionBalanceAdjust,
		TargetType: "user",
		TargetID:   id.Hex(),
		Summary: map[string]any{
			"direction": body.Direction, "amount": body.Amount,
			"balanceAfter": newBal, "reason": strings.TrimSpace(body.Reason),
		},
	})
	response.OK(w, map[string]float64{"walletBalance": newBal})
}

// listTiers handles GET /api/admin/reseller-tiers — every tier definition plus
// its current reseller headcount.
func (a *adminHandler) listTiers(w http.ResponseWriter, r *http.Request) {
	tiers, err := a.tiers.ListTiers(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	counts, err := a.tiers.CountByTier(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	out := make([]tierWithCount, len(tiers))
	for i, t := range tiers {
		out[i] = tierWithCount{ResellerTier: t, Count: counts[t.Name]}
	}
	response.OK(w, out)
}

// tierBody is the create/update payload for a tier definition.
type tierBody struct {
	Name          string  `json:"name"`
	MarginPercent float64 `json:"marginPercent"`
}

// validate checks a tier body, returning a client-facing message when invalid.
func (b tierBody) validate() (string, bool) {
	if strings.TrimSpace(b.Name) == "" {
		return "name is required", false
	}
	if b.MarginPercent < 0 || b.MarginPercent > 100 {
		return "marginPercent must be between 0 and 100", false
	}
	return "", true
}

// createTier handles POST /api/admin/reseller-tiers.
func (a *adminHandler) createTier(w http.ResponseWriter, r *http.Request) {
	var body tierBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if msg, ok := body.validate(); !ok {
		response.BadRequest(w, msg)
		return
	}
	tier := &ResellerTier{
		Name:          strings.TrimSpace(body.Name),
		MarginPercent: body.MarginPercent,
	}
	if err := a.tiers.CreateTier(r.Context(), tier); err != nil {
		writeRepoError(w, err)
		return
	}
	response.OK(w, tier)
}

// updateTier handles PUT /api/admin/reseller-tiers/{id}.
func (a *adminHandler) updateTier(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body tierBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if msg, ok := body.validate(); !ok {
		response.BadRequest(w, msg)
		return
	}
	tier, err := a.tiers.UpdateTier(r.Context(), id, strings.TrimSpace(body.Name), body.MarginPercent)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	response.OK(w, tier)
}

// deleteTier handles DELETE /api/admin/reseller-tiers/{id}.
func (a *adminHandler) deleteTier(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := a.tiers.DeleteTier(r.Context(), id); err != nil {
		writeRepoError(w, err)
		return
	}
	response.OK(w, map[string]bool{"deleted": true})
}

// listPrices handles GET /api/admin/resellers/{id}/prices — the reseller's
// per-variant price overrides.
func (a *adminHandler) listPrices(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	prices, err := a.tiers.ListPricesForUser(r.Context(), id)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, prices)
}

// setPrice handles POST /api/admin/resellers/{id}/prices — create or update a
// per-variant price override for the reseller (upsert on userId+variantId).
func (a *adminHandler) setPrice(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		ProductID string  `json:"productId"`
		VariantID string  `json:"variantId"`
		Price     float64 `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if strings.TrimSpace(body.VariantID) == "" {
		response.BadRequest(w, "variantId is required")
		return
	}
	if body.Price < 0 {
		response.BadRequest(w, "price must be zero or greater")
		return
	}
	p, err := a.tiers.SetPrice(r.Context(), id, strings.TrimSpace(body.ProductID), strings.TrimSpace(body.VariantID), body.Price)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionResellerPriceSet,
		TargetType: "user",
		TargetID:   id.Hex(),
		Summary:    map[string]any{"variantId": p.VariantID, "productId": p.ProductID, "price": p.Price},
	})
	response.OK(w, p)
}

// deletePrice handles DELETE /api/admin/resellers/{id}/prices/{variantId}.
func (a *adminHandler) deletePrice(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	variantID := chi.URLParam(r, "variantId")
	if err := a.tiers.DeletePrice(r.Context(), id, variantID); err != nil {
		writeRepoError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionResellerPriceDelete,
		TargetType: "user",
		TargetID:   id.Hex(),
		Summary:    map[string]any{"variantId": variantID},
	})
	response.OK(w, map[string]bool{"deleted": true})
}

// tierMargins returns a map of tier name → marginPercent, for cheap per-row
// margin lookup when listing resellers.
func (a *adminHandler) tierMargins(ctx context.Context) (map[string]float64, error) {
	tiers, err := a.tiers.ListTiers(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]float64, len(tiers))
	for _, t := range tiers {
		out[t.Name] = t.MarginPercent
	}
	return out, nil
}

// orderStatsByUser aggregates completed-order count + total volume per reseller
// for the given ids, in a single query against the orders collection. Resellers
// with no completed orders are simply absent from the map (zero value).
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
			{Key: "volume", Value: bson.D{{Key: "$sum", Value: "$total"}}},
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
		Volume float64       `bson:"volume"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = orderStats{Orders: row.Orders, Volume: row.Volume}
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
// missing entity, 409 for a duplicate tier name, 400 for an error wrapping a
// bad-request — e.g. insufficient funds — and 500 otherwise).
func writeRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	case errors.Is(err, apperrors.ErrConflict):
		response.Error(w, http.StatusConflict, "CONFLICT", "a tier with that name already exists")
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
