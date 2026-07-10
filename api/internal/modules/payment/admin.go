package payment

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// AdminHandler serves the admin payment-intent views plus the shared-address
// reconciliation queue. Intent settlement stays automatic (no admin actions on
// intents); the only mutations here are on unmatched deposits — transfers to
// the shared address the watcher couldn't match, which an admin attributes to
// a customer (wallet credit) or ignores. The users collection is read directly
// for email→id resolution (the user package imports payment's dependents, so
// this keeps the graph acyclic — same pattern as wallet/admin.go).
type AdminHandler struct {
	intents *mongo.Collection
	users   *mongo.Collection
	svc     *Service
	rec     audit.Recorder
}

// RegisterAdminRoutes mounts the admin payment routes onto r (the /api/admin
// group, guarded by AdminOnly; RequireDomain maps GET→payments.view and
// POST→payments.manage).
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, svc *Service, rec audit.Recorder) {
	h := &AdminHandler{
		intents: db.Collection("payment_intents"),
		users:   db.Collection("users"),
		svc:     svc,
		rec:     rec,
	}
	r.Get("/payments", h.list)
	r.Get("/payments/deposits", h.listDeposits)
	r.Post("/payments/deposits/{id}/attribute", h.attributeDeposit)
	r.Post("/payments/deposits/{id}/ignore", h.ignoreDeposit)
	r.Get("/payments/{id}", h.get)
}

// adminUser is the slice of the owning user surfaced per intent.
type adminUser struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// adminIntentView is the admin listing row.
type adminIntentView struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Purpose     Purpose   `json:"purpose"`
	OrderID     string    `json:"orderId,omitempty"`
	Network     string    `json:"network"`
	Address     string    `json:"address"`
	AddressMode string    `json:"addressMode"`
	AmountUSD   float64   `json:"amountUsd"`
	SaltUSD     float64   `json:"saltUsd,omitempty"`
	ReceivedUSD float64   `json:"receivedUsd"`
	Status      string    `json:"status"`
	TxHash      string    `json:"txHash,omitempty"`
	FromAddress string    `json:"fromAddress,omitempty"`
	Settlement  string    `json:"settlement,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
	ConfirmedAt any       `json:"confirmedAt,omitempty"`
	User        adminUser `json:"user"`
}

type adminIntentRow struct {
	Intent  `bson:",inline"`
	UserDoc struct {
		Email string `bson:"email"`
		Phone string `bson:"phone"`
	} `bson:"userDoc"`
}

func (row *adminIntentRow) view() adminIntentView {
	mode := row.AddressMode
	if mode == "" { // documents that predate shared mode
		mode = AddressModeDerived
	}
	v := adminIntentView{
		ID:          row.ID.Hex(),
		UserID:      row.UserID.Hex(),
		Purpose:     row.Purpose,
		Network:     row.Network,
		Address:     row.Address,
		AddressMode: mode,
		SaltUSD:     MicrosToUSD(row.AmountSaltMicros),
		AmountUSD:   MicrosToUSD(row.AmountExpectedMicros),
		ReceivedUSD: MicrosToUSD(row.AmountReceivedMicros),
		Status:      string(row.Status),
		TxHash:      row.TxHash,
		FromAddress: row.FromAddress,
		Settlement:  row.Settlement,
		CreatedAt:   row.CreatedAt,
		ExpiresAt:   row.ExpiresAt,
		User:        adminUser{Email: row.UserDoc.Email, Phone: row.UserDoc.Phone},
	}
	if row.OrderID != nil {
		v.OrderID = row.OrderID.Hex()
	}
	if row.ConfirmedAt != nil {
		v.ConfirmedAt = row.ConfirmedAt
	}
	return v
}

// list handles GET /api/admin/payments — paginated, newest first, optional
// status / purpose filters (one pass: filter → page → join the owning user,
// same shape as the finance ledger feed).
func (h *AdminHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p := pagination.ParseParams(r)
	q := r.URL.Query()

	match := bson.D{}
	if v := strings.TrimSpace(q.Get("status")); v != "" {
		match = append(match, bson.E{Key: "status", Value: v})
	}
	if v := strings.TrimSpace(q.Get("purpose")); v != "" {
		match = append(match, bson.E{Key: "purpose", Value: v})
	}

	total, err := h.intents.CountDocuments(ctx, match)
	if err != nil {
		response.InternalError(w)
		return
	}

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
	cur, err := h.intents.Aggregate(ctx, pipeline)
	if err != nil {
		response.InternalError(w)
		return
	}
	defer cur.Close(ctx)

	var rows []adminIntentRow
	if err := cur.All(ctx, &rows); err != nil {
		response.InternalError(w)
		return
	}
	views := make([]adminIntentView, len(rows))
	for i := range rows {
		views[i] = rows[i].view()
	}
	response.OKWithMeta(w, views, pagination.CalcMeta(p, total))
}

// get handles GET /api/admin/payments/{id}.
func (h *AdminHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "_id", Value: id}}}},
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
	cur, err := h.intents.Aggregate(r.Context(), pipeline)
	if err != nil {
		response.InternalError(w)
		return
	}
	defer cur.Close(r.Context())

	var rows []adminIntentRow
	if err := cur.All(r.Context(), &rows); err != nil {
		response.InternalError(w)
		return
	}
	if len(rows) == 0 {
		response.NotFound(w)
		return
	}
	response.OK(w, rows[0].view())
}

// adminDepositView is a Deposit with USD amounts for the console.
type adminDepositView struct {
	*Deposit
	AmountUSD float64 `json:"amountUsd"`
}

func depositView(d *Deposit) adminDepositView {
	return adminDepositView{Deposit: d, AmountUSD: MicrosToUSD(d.AmountMicros)}
}

// listDeposits handles GET /api/admin/payments/deposits — the shared-address
// reconciliation queue, newest first, optional status filter.
func (h *AdminHandler) listDeposits(w http.ResponseWriter, r *http.Request) {
	p := pagination.ParseParams(r)
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	deposits, total, err := h.svc.ListDeposits(r.Context(), status, p)
	if err != nil {
		response.InternalError(w)
		return
	}
	views := make([]adminDepositView, len(deposits))
	for i, d := range deposits {
		views[i] = depositView(d)
	}
	response.OKWithMeta(w, views, pagination.CalcMeta(p, total))
}

// attributeDeposit handles POST /api/admin/payments/deposits/{id}/attribute:
// credit an unmatched deposit to a customer's wallet. body.user is a user id
// (hex) or an account email, resolved here.
func (h *AdminHandler) attributeDeposit(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	var body struct {
		User string `json:"user"`
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.User) == "" {
		response.BadRequest(w, "user (id or email) is required")
		return
	}
	userID, err := h.resolveUser(r, strings.TrimSpace(body.User))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "USER_NOT_FOUND", "no user matches that id or email")
		return
	}

	d, err := h.svc.AttributeDeposit(r.Context(), id, userID, actorLabel(r), strings.TrimSpace(body.Note))
	if err != nil {
		if errors.Is(err, apperrors.ErrConflict) {
			response.Error(w, http.StatusConflict, "DEPOSIT_NOT_UNMATCHED", "this deposit was already attributed or ignored")
			return
		}
		response.InternalError(w)
		return
	}
	h.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionDepositAttribute,
		TargetType: "usdt_deposit",
		TargetID:   id.Hex(),
		Summary: map[string]any{
			"txHash": d.TxHash, "amountUsd": MicrosToUSD(d.AmountMicros), "userId": userID.Hex(),
		},
	})
	response.OK(w, depositView(d))
}

// ignoreDeposit handles POST /api/admin/payments/deposits/{id}/ignore.
func (h *AdminHandler) ignoreDeposit(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if err := h.svc.IgnoreDeposit(r.Context(), id, actorLabel(r), strings.TrimSpace(body.Note)); err != nil {
		if errors.Is(err, apperrors.ErrConflict) {
			response.Error(w, http.StatusConflict, "DEPOSIT_NOT_UNMATCHED", "this deposit was already attributed or ignored")
			return
		}
		response.InternalError(w)
		return
	}
	h.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionDepositIgnore,
		TargetType: "usdt_deposit",
		TargetID:   id.Hex(),
	})
	response.OK(w, map[string]bool{"ignored": true})
}

// resolveUser turns an id-hex or email into a user ObjectID.
func (h *AdminHandler) resolveUser(r *http.Request, ref string) (bson.ObjectID, error) {
	if id, err := bson.ObjectIDFromHex(ref); err == nil {
		// Verify it exists so a typo'd hex can't credit a phantom account.
		err := h.users.FindOne(r.Context(), bson.D{{Key: "_id", Value: id}}).Err()
		return id, err
	}
	var doc struct {
		ID bson.ObjectID `bson:"_id"`
	}
	err := h.users.FindOne(r.Context(),
		bson.D{{Key: "email", Value: strings.ToLower(ref)}},
	).Decode(&doc)
	return doc.ID, err
}

// actorLabel resolves the acting admin for audit rows.
func actorLabel(r *http.Request) string {
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		return auth.ActorLabel(claims)
	}
	return ""
}
