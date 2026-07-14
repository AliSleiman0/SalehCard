package code

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// RegisterAdminRoutes mounts the inventory/code admin routes onto r (the
// /api/admin group, already guarded by AdminOnly + RequireDomain("inventory")).
// It is fully implemented and extends the product reference slice. rec records
// lifecycle actions.
//
// The two GETs that return RAW code values — the product code list and the
// single-code lookup — additionally require inventory.manage, not the default
// inventory.view: the codes are the sellable asset, so a view-only "stock
// monitor" role must see stock counts (GET /inventory, aggregate only) without
// being able to exfiltrate the code pool. resend-code is NOT registered here —
// it is an order operation and lives under the orders domain (see
// RegisterOrderResendRoutes).
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder) {
	repo := NewMongoRepository(db)
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("code: failed to ensure indexes", "error", err)
	}
	svc := NewCodeService(repo)
	h := NewHandler(svc)

	r.Get("/inventory", h.Inventory)
	r.Get("/upload-history", uploadHistoryHandler(repo))
	r.Post("/products/{id}/codes", h.Upload)
	r.With(auth.RequirePermission("inventory.manage")).Get("/products/{id}/codes", h.ListCodes)
	// Single-code management (the inventory codes popup). These return / mutate raw
	// code values, so they carry the explicit inventory.manage guard (the domain
	// wrapper already forces manage on non-GET; this matches the raw-code reads above).
	r.With(auth.RequirePermission("inventory.manage")).Post("/products/{id}/codes/single", addCodeHandler(svc, rec))
	r.With(auth.RequirePermission("inventory.manage")).Put("/products/{id}/codes/{codeId}", editCodeHandler(svc, rec))
	r.With(auth.RequirePermission("inventory.manage")).Delete("/products/{id}/codes/{codeId}", deleteCodeHandler(svc, rec))
	r.Put("/products/{id}/stock-threshold", h.SetThreshold)
	r.With(auth.RequirePermission("inventory.manage")).Get("/codes/{code}", h.Lookup)
	r.Put("/codes/{code}/expire", expireHandler(svc, rec))
}

// RegisterOrderResendRoutes mounts POST /orders/{id}/resend-code, which
// re-delivers a customer's already-purchased codes. Although the handler lives
// in the code module, it is an order-fulfillment action keyed by order id, so
// the server mounts it inside the orders domain group — it requires
// orders.manage, not inventory.manage. ntf re-delivers the codes.
func RegisterOrderResendRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder, ntf notification.Notifier) {
	svc := NewCodeService(NewMongoRepository(db))
	r.Post("/orders/{id}/resend-code", resendHandler(svc, rec, ntf))
}

// expireHandler serves PUT /api/admin/codes/{code}/expire — retires an unsold
// code from the available pool.
func expireHandler(svc *CodeService, rec audit.Recorder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := svc.Expire(r.Context(), chi.URLParam(r, "code"))
		if err != nil {
			writeCodeError(w, err)
			return
		}
		rec.Record(r.Context(), audit.Entry{
			Action:     audit.ActionCodeExpire,
			TargetType: "code",
			TargetID:   c.ID.Hex(),
			Summary:    map[string]any{"code": c.Code, "productId": c.ProductID},
		})
		response.OK(w, c)
	}
}

// addCodeHandler serves POST /api/admin/products/{id}/codes/single — adds one
// available code to a product's pool from the inventory codes popup.
func addCodeHandler(svc *CodeService, rec audit.Recorder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID := chi.URLParam(r, "id")
		var in CodeInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			response.BadRequest(w, "invalid request body")
			return
		}
		c, err := svc.AddCode(r.Context(), productID, in.Code, in.Pin)
		if err != nil {
			writeCodeError(w, err)
			return
		}
		rec.Record(r.Context(), audit.Entry{
			Action:     audit.ActionCodeCreate,
			TargetType: "code",
			TargetID:   c.ID.Hex(),
			Summary:    map[string]any{"code": c.Code, "productId": c.ProductID},
		})
		response.JSON(w, http.StatusCreated, response.Response{Success: true, Data: c})
	}
}

// editCodeHandler serves PUT /api/admin/products/{id}/codes/{codeId} — edits an
// available code's value/pin.
func editCodeHandler(svc *CodeService, rec audit.Recorder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID := chi.URLParam(r, "id")
		codeID := chi.URLParam(r, "codeId")
		var in CodeInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			response.BadRequest(w, "invalid request body")
			return
		}
		c, err := svc.EditCode(r.Context(), productID, codeID, in.Code, in.Pin)
		if err != nil {
			writeCodeError(w, err)
			return
		}
		rec.Record(r.Context(), audit.Entry{
			Action:     audit.ActionCodeUpdate,
			TargetType: "code",
			TargetID:   c.ID.Hex(),
			Summary:    map[string]any{"code": c.Code, "productId": c.ProductID},
		})
		response.OK(w, c)
	}
}

// deleteCodeHandler serves DELETE /api/admin/products/{id}/codes/{codeId} —
// removes a non-delivered code from a product's pool.
func deleteCodeHandler(svc *CodeService, rec audit.Recorder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID := chi.URLParam(r, "id")
		codeID := chi.URLParam(r, "codeId")
		c, err := svc.DeleteCode(r.Context(), productID, codeID)
		if err != nil {
			writeCodeError(w, err)
			return
		}
		rec.Record(r.Context(), audit.Entry{
			Action:     audit.ActionCodeDelete,
			TargetType: "code",
			TargetID:   c.ID.Hex(),
			Summary:    map[string]any{"code": c.Code, "productId": c.ProductID},
		})
		response.OK(w, c)
	}
}

// resendHandler serves POST /api/admin/orders/{id}/resend-code — re-notifies the
// buyer with the code(s) delivered on their order (e.g. a lost delivery).
func resendHandler(svc *CodeService, rec audit.Recorder, ntf notification.Notifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := chi.URLParam(r, "id")
		codes, err := svc.CodesForOrder(r.Context(), orderID)
		if err != nil {
			response.InternalError(w)
			return
		}
		if len(codes) == 0 {
			response.Error(w, http.StatusNotFound, "NO_CODES", "no delivered codes found for this order")
			return
		}
		resent := 0
		for _, c := range codes {
			uid, err := bson.ObjectIDFromHex(c.DeliveredTo)
			if err != nil {
				continue // a code without a resolvable recipient is skipped
			}
			ntf.Notify(r.Context(), uid, notification.Note{
				Kind:  notification.KindCodeDelivered,
				Title: "Your code",
				Body:  "Here is your delivered code again.",
				Data:  map[string]string{"orderId": orderID, "code": c.Code, "productId": c.ProductID},
			})
			resent++
		}
		rec.Record(r.Context(), audit.Entry{
			Action:     audit.ActionCodeResend,
			TargetType: "order",
			TargetID:   orderID,
			Summary:    map[string]any{"codes": len(codes)},
		})
		response.OK(w, map[string]int{"resent": resent})
	}
}

// writeCodeError maps code lifecycle errors: missing → 404, bad request → 400,
// conflict (duplicate / not-available / delivered) → 409, otherwise 500.
func writeCodeError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	case errors.As(err, &appErr) && errors.Is(err, apperrors.ErrBadRequest):
		response.Error(w, http.StatusBadRequest, appErr.Code, appErr.Message)
	case errors.As(err, &appErr) && errors.Is(err, apperrors.ErrConflict):
		response.Error(w, http.StatusConflict, appErr.Code, appErr.Message)
	default:
		response.InternalError(w)
	}
}

// uploadHistoryHandler serves GET /api/admin/upload-history?limit= — the recent
// upload batches for the inventory screen. It reads the repository directly (no
// Service method) to keep the Service interface — and its test fakes — unchanged.
func uploadHistoryHandler(repo *MongoRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		history, err := repo.ListUploadHistory(r.Context(), limit)
		if err != nil {
			response.InternalError(w)
			return
		}
		response.OK(w, history)
	}
}

// NewService exposes a constructed Service for callers (e.g. the dashboard) that
// need inventory data without re-wiring the repository.
func NewService(db *mongo.Database) Service {
	return NewCodeService(NewMongoRepository(db))
}
