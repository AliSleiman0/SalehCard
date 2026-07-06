package payment

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// AdminHandler serves the read-only admin payment-intent views. There are no
// admin actions here by design: settlement is automatic and the money trail
// lives in the wallet ledger (finance module); this surface exists for
// support/monitoring (find a customer's deposit, follow the tx on tronscan).
type AdminHandler struct {
	intents *mongo.Collection
}

// RegisterAdminRoutes mounts the admin payment routes onto r (the /api/admin
// group, guarded by AdminOnly).
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	h := &AdminHandler{intents: db.Collection("payment_intents")}
	r.Get("/payments", h.list)
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
	AmountUSD   float64   `json:"amountUsd"`
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
	v := adminIntentView{
		ID:          row.ID.Hex(),
		UserID:      row.UserID.Hex(),
		Purpose:     row.Purpose,
		Network:     row.Network,
		Address:     row.Address,
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
