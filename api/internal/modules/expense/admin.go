package expense

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// adminHandler serves the admin expense endpoints over the repository directly.
type adminHandler struct {
	repo Repository
}

// RegisterAdminRoutes mounts the admin expense CRUD onto r (the /api/admin group,
// guarded by AdminOnly): paginated/filterable list, summary totals, create,
// detail, update, delete.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	_ = EnsureIndexes(context.Background(), db)
	a := &adminHandler{repo: NewMongoRepository(db.Collection("expenses"))}

	r.Get("/expenses", a.list)
	r.Get("/expenses/summary", a.summary) // before /{id} so it isn't shadowed
	r.Post("/expenses", a.create)
	r.Get("/expenses/{id}", a.detail)
	r.Put("/expenses/{id}", a.update)
	r.Delete("/expenses/{id}", a.delete)
}

// parseFilter builds an ExpenseFilter from the request query (shared by list
// and summary). Unparseable dates are ignored.
func parseFilter(r *http.Request) ExpenseFilter {
	q := r.URL.Query()
	f := ExpenseFilter{
		Category: strings.TrimSpace(q.Get("category")),
		Currency: strings.TrimSpace(q.Get("currency")),
		Search:   strings.TrimSpace(q.Get("q")),
	}
	if raw := strings.TrimSpace(q.Get("from")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			f.From = &t
		}
	}
	if raw := strings.TrimSpace(q.Get("to")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			f.To = &t
		}
	}
	return f
}

// list handles GET /api/admin/expenses — paginated, with optional category,
// currency, date-range, and note-search filters.
func (a *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	p := pagination.ParseParams(r)
	expenses, total, err := a.repo.List(r.Context(), parseFilter(r), p)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OKWithMeta(w, expenses, pagination.CalcMeta(p, total))
}

// summary handles GET /api/admin/expenses/summary — totals over all matching
// rows (same filters as list, no pagination).
func (a *adminHandler) summary(w http.ResponseWriter, r *http.Request) {
	s, err := a.repo.Summarize(r.Context(), parseFilter(r))
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, s)
}

// detail handles GET /api/admin/expenses/{id}.
func (a *adminHandler) detail(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	e, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
		writeExpenseError(w, err)
		return
	}
	response.OK(w, e)
}

// create handles POST /api/admin/expenses.
func (a *adminHandler) create(w http.ResponseWriter, r *http.Request) {
	var b expenseBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if err := b.validate(); err != nil {
		writeExpenseError(w, err)
		return
	}
	e := b.toExpense()
	if err := a.repo.Create(r.Context(), e); err != nil {
		writeExpenseError(w, err)
		return
	}
	response.OK(w, e)
}

// update handles PUT /api/admin/expenses/{id}.
func (a *adminHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var b expenseBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if err := b.validate(); err != nil {
		writeExpenseError(w, err)
		return
	}
	e, err := a.repo.Update(r.Context(), id, b.toUpdate())
	if err != nil {
		writeExpenseError(w, err)
		return
	}
	response.OK(w, e)
}

// delete handles DELETE /api/admin/expenses/{id}.
func (a *adminHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := a.repo.Delete(r.Context(), id); err != nil {
		writeExpenseError(w, err)
		return
	}
	response.OK(w, map[string]bool{"deleted": true})
}

// expenseBody is the create/update request payload.
type expenseBody struct {
	Amount        float64         `json:"amount"`
	Currency      string          `json:"currency"`
	Category      ExpenseCategory `json:"category"`
	CategoryOther string          `json:"categoryOther"`
	Note          string          `json:"note"`
	IncurredAt    time.Time       `json:"incurredAt"`
}

// validate enforces the field constraints shared by create and update. Blank
// currency defaults to USD; an "other" category requires a custom label.
func (b *expenseBody) validate() error {
	if b.Amount <= 0 {
		return badRequest("amount must be greater than zero")
	}
	b.Currency = strings.TrimSpace(b.Currency)
	if b.Currency == "" {
		b.Currency = "USD"
	}
	if !b.Category.IsValid() {
		return badRequest("category is invalid")
	}
	b.CategoryOther = strings.TrimSpace(b.CategoryOther)
	if b.Category == CategoryOther && b.CategoryOther == "" {
		return badRequest("a custom category label is required when category is other")
	}
	if b.Category != CategoryOther {
		b.CategoryOther = ""
	}
	if b.IncurredAt.IsZero() {
		return badRequest("incurredAt is required")
	}
	return nil
}

// toExpense builds a new Expense from the create payload.
func (b *expenseBody) toExpense() *Expense {
	return &Expense{
		Amount:        b.Amount,
		Currency:      b.Currency,
		Category:      b.Category,
		CategoryOther: b.CategoryOther,
		Note:          strings.TrimSpace(b.Note),
		IncurredAt:    b.IncurredAt.UTC(),
	}
}

// toUpdate builds the editable-field set from the update payload.
func (b *expenseBody) toUpdate() ExpenseUpdate {
	return ExpenseUpdate{
		Amount:        b.Amount,
		Currency:      b.Currency,
		Category:      b.Category,
		CategoryOther: b.CategoryOther,
		Note:          strings.TrimSpace(b.Note),
		IncurredAt:    b.IncurredAt.UTC(),
	}
}

// badRequest builds a 400-class AppError with the BAD_REQUEST machine code.
func badRequest(msg string) error {
	return &apperrors.AppError{Code: "BAD_REQUEST", Message: msg, Err: apperrors.ErrBadRequest}
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

// writeExpenseError maps repository/validation errors to HTTP responses.
func writeExpenseError(w http.ResponseWriter, err error) {
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
