package user

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/sms"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// bulkSMSTimeout bounds the detached bulk-SMS fan-out.
const bulkSMSTimeout = 2 * time.Minute

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
	repo       Repository
	wallet     wallet.Repository
	orders     *mongo.Collection
	roles      *mongo.Collection
	rec        audit.Recorder
	smsSender  sms.Sender
	maxBulkSMS int
}

// RegisterAdminRoutes mounts the admin user routes onto r (the /api/admin group,
// guarded by AdminOnly): paginated/filterable list, detail, role + status
// changes, and manual wallet adjustment (credit/debit with a reason logged to the
// ledger). Mutations are recorded via rec.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder, smsSender sms.Sender, maxBulkSMS int) {
	a := &adminHandler{
		repo:       NewMongoRepository(db),
		wallet:     wallet.NewMongoRepository(db),
		orders:     db.Collection("orders"),
		roles:      db.Collection("roles"),
		rec:        rec,
		smsSender:  smsSender,
		maxBulkSMS: maxBulkSMS,
	}

	r.Get("/users", a.list)
	r.Post("/users/bulk", a.bulkStatus)
	r.Post("/users/bulk-sms", a.bulkSMS)
	r.Get("/users/{id}", a.detail)
	r.Put("/users/{id}/role", a.updateRole)
	r.Put("/users/{id}/status", a.updateStatus)
	r.Post("/users/{id}/wallet-adjust", a.walletAdjust)
	r.Delete("/users/{id}", a.deleteUser)
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

// updateRole handles PUT /api/admin/users/{id}/role. The body carries the full
// target state: {role, adminRoleId?} — for an admin, a missing/empty adminRoleId
// means built-in Super Admin, otherwise it references a custom RBAC role. Any
// change that grants or revokes admin access (or reassigns an admin's RBAC
// role) requires a super-admin actor, so a users.manage role can never escalate
// its own authority. Permission changes bite on the target's next token refresh.
func (a *adminHandler) updateRole(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		Role        Role    `json:"role"`
		AdminRoleID *string `json:"adminRoleId"`
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
	var adminRoleID *bson.ObjectID
	if body.Role == RoleAdmin && body.AdminRoleID != nil && *body.AdminRoleID != "" {
		rid, err := bson.ObjectIDFromHex(*body.AdminRoleID)
		if err != nil {
			response.BadRequest(w, "adminRoleId is not a valid id")
			return
		}
		// Exists-then-assign is not atomic: a concurrent role delete could remove
		// this role between the check and the write, leaving a dangling
		// adminRoleId (resolvePerms fails closed to no permissions). Accepted, the
		// mirror of role delete's ROLE_IN_USE race — a low-concurrency,
		// super-admin-only surface under the no-transactions model.
		n, err := a.roles.CountDocuments(r.Context(), bson.D{{Key: "_id", Value: rid}})
		if err != nil {
			response.InternalError(w)
			return
		}
		if n == 0 {
			response.BadRequest(w, "adminRoleId does not reference an existing role")
			return
		}
		adminRoleID = &rid
	}
	current, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
		a.writeRepoError(w, err)
		return
	}
	claims, _ := auth.ClaimsFromContext(r.Context())
	if (current.Role == RoleAdmin || body.Role == RoleAdmin) && !claims.IsSuperAdmin() {
		response.Forbidden(w, "only a super admin can grant or change admin access")
		return
	}
	// Never remove the platform's last remaining super admin — that would leave
	// nobody able to manage roles or admin accounts. (Count-then-update; the
	// admin-only surface makes the race window acceptable.)
	wasSuper := current.Role == RoleAdmin && current.AdminRoleID == nil
	staysSuper := body.Role == RoleAdmin && adminRoleID == nil
	if wasSuper && !staysSuper {
		if ok := a.requireAnotherSuperAdmin(w, r, "cannot demote the last remaining super admin"); !ok {
			return
		}
	}
	u, err := a.repo.UpdateRole(r.Context(), id, body.Role, adminRoleID)
	if err != nil {
		a.writeRepoError(w, err)
		return
	}
	summary := map[string]any{"from": string(current.Role), "to": string(body.Role)}
	if current.AdminRoleID != nil {
		summary["fromAdminRole"] = current.AdminRoleID.Hex()
	}
	if adminRoleID != nil {
		summary["toAdminRole"] = adminRoleID.Hex()
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionRoleChange,
		TargetType: "user",
		TargetID:   id.Hex(),
		Summary:    summary,
	})
	response.OK(w, u)
}

// bulkStatus handles POST /api/admin/users/bulk — suspend or activate many users
// in one call. For the suspend action, admin accounts are dropped from the batch
// (they must be suspended individually via the last-admin-guarded single
// endpoint) so a bulk action can never lock everyone out of the console.
func (a *adminHandler) bulkStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs    []string `json:"ids"`
		Action string   `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	var status Status
	switch body.Action {
	case "suspend":
		status = StatusSuspended
	case "activate":
		status = StatusActive
	default:
		response.BadRequest(w, "action must be one of suspend, activate")
		return
	}
	ids := make([]bson.ObjectID, 0, len(body.IDs))
	for _, s := range body.IDs {
		if id, err := bson.ObjectIDFromHex(s); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		response.BadRequest(w, "no valid user ids supplied")
		return
	}
	// Bulk status is a customer-management tool: exclude admin accounts from the
	// batch entirely, for BOTH actions. Admin accounts are managed only through
	// the super-admin-gated single endpoints — otherwise a users.manage admin
	// could reactivate (or suspend) an admin here, bypassing that gate.
	users, err := a.repo.FindByIDs(r.Context(), ids)
	if err != nil {
		response.InternalError(w)
		return
	}
	ids = ids[:0]
	for _, u := range users {
		if u.Role != RoleAdmin {
			ids = append(ids, u.ID)
		}
	}
	if len(ids) == 0 {
		response.OK(w, map[string]int64{"modified": 0})
		return
	}
	modified, err := a.repo.BulkUpdateStatus(r.Context(), ids, status)
	if err != nil {
		response.InternalError(w)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionStatusChange,
		TargetType: "user",
		TargetID:   "bulk",
		Summary:    map[string]any{"action": body.Action, "count": modified},
	})
	response.OK(w, map[string]int64{"modified": modified})
}

// bulkSMSMaxLen caps a bulk-SMS message at a single 160-char GSM segment, so one
// send is billed as one segment per recipient (the frontend enforces the same).
const bulkSMSMaxLen = 160

// bulkSMS handles POST /api/admin/users/bulk-sms — send a single-segment SMS to
// many users at once over the live SMS provider. Recipients are resolved from the
// id list; users without a phone (email-only or deleted accounts) are skipped.
// Because this sends real, paid SMS, the recipient count is hard-capped at
// maxBulkSMS (400 BULK_SMS_LIMIT) and the message at 160 chars. The fan-out runs
// detached from the request (best-effort, serial, log-and-continue per recipient —
// serial avoids hammering the provider), so the response returns immediately with
// how many were queued.
func (a *adminHandler) bulkSMS(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs     []string `json:"ids"`
		Message string   `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	message := strings.TrimSpace(body.Message)
	if message == "" {
		response.BadRequest(w, "message is required")
		return
	}
	if len(message) > bulkSMSMaxLen {
		response.BadRequest(w, "message must be 160 characters or fewer")
		return
	}
	ids := make([]bson.ObjectID, 0, len(body.IDs))
	for _, s := range body.IDs {
		if id, err := bson.ObjectIDFromHex(s); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		response.BadRequest(w, "no valid user ids supplied")
		return
	}
	users, err := a.repo.FindByIDs(r.Context(), ids)
	if err != nil {
		response.InternalError(w)
		return
	}
	recipients := make([]string, 0, len(users))
	for _, u := range users {
		if u.Status == StatusDeleted || u.Phone == nil {
			continue
		}
		if p := strings.TrimSpace(*u.Phone); p != "" {
			recipients = append(recipients, p)
		}
	}

	// Spend guardrail: refuse a batch larger than the configured cap so a stray
	// broad selection can't fire thousands of paid messages.
	if len(recipients) > a.maxBulkSMS {
		response.Error(w, http.StatusBadRequest, "BULK_SMS_LIMIT",
			"too many recipients for one send — narrow the selection (limit "+strconv.Itoa(a.maxBulkSMS)+")")
		return
	}

	if a.smsSender != nil && len(recipients) > 0 {
		bg := context.WithoutCancel(r.Context())
		go func() {
			ctx, cancel := context.WithTimeout(bg, bulkSMSTimeout)
			defer cancel()
			for _, to := range recipients {
				if err := a.smsSender.Send(ctx, to, message); err != nil {
					slog.Warn("bulk-sms: send failed", "to", to, "error", err)
				}
			}
		}()
	}

	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionUserBulkSMS,
		TargetType: "user",
		TargetID:   "bulk",
		Summary:    map[string]any{"recipients": len(recipients)},
	})
	response.OK(w, map[string]int{"queued": len(recipients)})
}

// deleteUser handles DELETE /api/admin/users/{id} — a soft-delete (anonymize).
// The row is kept so the account's orders and ledger rows still resolve; its PII
// is scrubbed and it can no longer sign in. The last remaining admin can't be
// deleted (same guard as demote/suspend).
func (a *adminHandler) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	current, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
		a.writeRepoError(w, err)
		return
	}
	if current.Role == RoleAdmin {
		// Admin accounts are managed by super admins only, and the last super
		// admin can never be deleted.
		if claims, _ := auth.ClaimsFromContext(r.Context()); !claims.IsSuperAdmin() {
			response.Forbidden(w, "only a super admin can delete an admin account")
			return
		}
		if current.AdminRoleID == nil {
			if ok := a.requireAnotherSuperAdmin(w, r, "cannot delete the last remaining super admin"); !ok {
				return
			}
		}
	}
	u, err := a.repo.SoftDelete(r.Context(), id)
	if err != nil {
		a.writeRepoError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionUserDelete,
		TargetType: "user",
		TargetID:   id.Hex(),
		Summary:    map[string]any{"email": current.Email, "role": string(current.Role)},
	})
	response.OK(w, u)
}

// requireAnotherSuperAdmin writes a 409 and returns false when the platform has
// at most one active super admin left (so the caller must not demote/suspend/
// delete it — limited admins cannot manage roles or admin accounts, so losing
// the last super admin would lock those functions permanently).
func (a *adminHandler) requireAnotherSuperAdmin(w http.ResponseWriter, r *http.Request, msg string) bool {
	n, err := a.repo.CountActiveSuperAdmins(r.Context())
	if err != nil {
		response.InternalError(w)
		return false
	}
	if n <= 1 {
		response.Error(w, http.StatusConflict, "LAST_ADMIN", msg)
		return false
	}
	return true
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
	current, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
		a.writeRepoError(w, err)
		return
	}
	if current.Role == RoleAdmin {
		// Admin accounts are managed by super admins only.
		if claims, _ := auth.ClaimsFromContext(r.Context()); !claims.IsSuperAdmin() {
			response.Forbidden(w, "only a super admin can change an admin account's status")
			return
		}
		// Suspending the last active super admin would lock role/admin
		// management permanently, same as a demote.
		if body.Status == StatusSuspended && current.Status != StatusSuspended && current.AdminRoleID == nil {
			if ok := a.requireAnotherSuperAdmin(w, r, "cannot suspend the last remaining super admin"); !ok {
				return
			}
		}
	}
	u, err := a.repo.UpdateStatus(r.Context(), id, body.Status)
	if err != nil {
		a.writeRepoError(w, err)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionStatusChange,
		TargetType: "user",
		TargetID:   id.Hex(),
		Summary:    map[string]any{"from": string(current.Status), "to": string(body.Status)},
	})
	response.OK(w, u)
}

// walletAdjust handles POST /api/admin/users/{id}/wallet-adjust — a manual
// credit or debit. The balance change is atomic; the ledger row is mandatory:
// if the insert fails, the balance change is reversed (compensation, per the
// no-multi-document-transactions model) and the request fails — money never
// moves without a ledger row.
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

	if err := a.wallet.Create(r.Context(), &wallet.WalletTransaction{
		UserID:       id,
		Type:         wallet.TxTypeAdjustment,
		Amount:       signed,
		BalanceAfter: newBal,
		Method:       "admin",
		Ref:          strings.TrimSpace(body.Reason),
	}); err != nil {
		// Mandatory ledger: reverse the balance change so success always implies
		// a ledger row. A failed reversal (e.g. a credit already spent) is logged
		// loudly for manual reconciliation.
		var undoErr error
		if signed > 0 {
			_, undoErr = a.wallet.Debit(r.Context(), id, body.Amount)
		} else {
			_, undoErr = a.wallet.Credit(r.Context(), id, body.Amount)
		}
		slog.Error("wallet-adjust: ledger insert failed; balance change reverted",
			"user", id.Hex(), "amount", signed, "ledgerError", err, "revertError", undoErr)
		response.Error(w, http.StatusInternalServerError, "LEDGER_WRITE_FAILED",
			"adjustment was reverted: the ledger row could not be written")
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionWalletAdjust,
		TargetType: "user",
		TargetID:   id.Hex(),
		Summary: map[string]any{
			"direction": body.Direction, "amount": body.Amount,
			"balanceAfter": newBal, "reason": strings.TrimSpace(body.Reason),
		},
	})
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
