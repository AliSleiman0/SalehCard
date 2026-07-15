package wallet

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/blob"
	"github.com/AliSleiman0/salehcard/api/internal/platform/imaging"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// maxUploadBytes caps a top-up document upload (multipart body) at 10 MB,
// mirroring the KYC + product image upload caps.
const maxUploadBytes = 10 << 20

// Handler exposes wallet domain operations over HTTP. It holds the concrete
// WalletService because the top-up request queue lives outside the shared
// Service interface, plus the blob store for document uploads.
type Handler struct {
	service *WalletService
	store   blob.Storage
}

// NewHandler constructs a Handler backed by the given service and blob store
// (the latter serves POST /api/v1/wallet/topups/documents; may be nil for
// consumers that never upload).
func NewHandler(service *WalletService, store blob.Storage) *Handler {
	return &Handler{service: service, store: store}
}

// walletView is the combined balance + history payload returned by GET /wallet.
type walletView struct {
	Balance      float64              `json:"balance"`
	Transactions []*WalletTransaction `json:"transactions"`
}

// GetWallet handles GET /api/v1/wallet, returning the balance and ledger.
func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	balance, err := h.service.GetBalance(r.Context(), id)
	if err != nil {
		writeWalletError(w, err)
		return
	}
	txs, err := h.service.ListTransactions(r.Context(), id)
	if err != nil {
		writeWalletError(w, err)
		return
	}
	response.OK(w, walletView{Balance: balance, Transactions: txs})
}

// TopUp handles POST /api/v1/wallet/topups — files a PENDING top-up request
// (the wallet is credited only when an admin approves; the previous instant
// mock credit is gone).
func (h *Handler) TopUp(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	var in TopUpInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	req, err := h.service.CreateTopUpRequest(r.Context(), id, in)
	if err != nil {
		writeWalletError(w, err)
		return
	}
	response.OK(w, req)
}

// ListMethods handles GET /api/v1/wallet/topup-methods — the enabled manual
// funding methods the app renders on the top-up screen.
func (h *Handler) ListMethods(w http.ResponseWriter, r *http.Request) {
	methods, err := h.service.ListEnabledMethods(r.Context())
	if err != nil {
		writeWalletError(w, err)
		return
	}
	response.OK(w, methods)
}

// uploadDocumentResponse is the POST /api/v1/wallet/topups/documents reply: the
// public URL the caller submits as a file-field value on the top-up request.
type uploadDocumentResponse struct {
	URL string `json:"url"`
}

// UploadDocument handles POST /api/v1/wallet/topups/documents — an authenticated
// customer uploads one payment-proof photo (e.g. a bank-transfer screenshot).
// The file is validated by content sniffing (not filename) and re-encoded into
// the 1024px display JPEG (platform/imaging) before being persisted under a
// topups/ key via the configured blob.Storage adapter — the same pipeline as
// KYC document uploads.
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		response.InternalError(w)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		response.BadRequest(w, "image must be a valid multipart upload no larger than 10 MB")
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		response.BadRequest(w, "an \"image\" file field is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		response.BadRequest(w, "failed to read the uploaded file")
		return
	}

	display, _, err := imaging.Process(data)
	if err != nil {
		response.BadRequest(w, "invalid image: "+err.Error())
		return
	}

	url, err := h.store.Upload(r.Context(), "topups/"+bson.NewObjectID().Hex()+".jpg", "image/jpeg", display)
	if err != nil {
		slog.Error("wallet: topup document upload failed", "error", err)
		response.InternalError(w)
		return
	}

	response.OK(w, uploadDocumentResponse{URL: url})
}

// ListTopUps handles GET /api/v1/wallet/topups — the user's request history.
func (h *Handler) ListTopUps(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	reqs, err := h.service.ListTopUpRequests(r.Context(), id)
	if err != nil {
		writeWalletError(w, err)
		return
	}
	response.OK(w, reqs)
}

// writeWalletError maps domain errors to HTTP responses.
func writeWalletError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		switch {
		case errors.Is(appErr, apperrors.ErrBadRequest):
			response.Error(w, http.StatusBadRequest, appErr.Code, appErr.Message)
		case errors.Is(appErr, apperrors.ErrNotFound):
			response.Error(w, http.StatusNotFound, appErr.Code, appErr.Message)
		default:
			response.InternalError(w)
		}
		return
	}
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	case errors.Is(err, apperrors.ErrBadRequest):
		response.BadRequest(w, err.Error())
	default:
		response.InternalError(w)
	}
}
