package kyc

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

// maxUploadBytes caps a KYC document upload (multipart body) at 10 MB,
// mirroring the product image upload cap.
const maxUploadBytes = 10 << 20

// Handler exposes the customer-facing KYC operations over HTTP.
type Handler struct {
	service Service
	store   blob.Storage
}

// NewHandler constructs a Handler backed by the given service and blob store
// (the latter serves POST /api/v1/kyc/documents).
func NewHandler(service Service, store blob.Storage) *Handler {
	return &Handler{service: service, store: store}
}

// Submit handles POST /api/v1/kyc — an authenticated customer submits their KYC
// background info, stored pending admin moderation.
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	var in SubmitInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	profile, err := h.service.Submit(r.Context(), userID, in)
	if err != nil {
		writeKycError(w, err)
		return
	}
	response.OK(w, profile)
}

// GetMe handles GET /api/v1/kyc/me — the caller's derived KYC profile.
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	profile, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		writeKycError(w, err)
		return
	}
	response.OK(w, profile)
}

// uploadDocumentResponse is the POST /api/v1/kyc/documents reply: the public
// URL the caller places in SubmitInput.DocumentFrontURL/DocumentBackURL.
type uploadDocumentResponse struct {
	ImageURL string `json:"imageUrl"`
}

// UploadDocument handles POST /api/v1/kyc/documents — an authenticated
// customer uploads one document photo ahead of submitting the KYC form. The
// file is validated by content sniffing (not filename) and re-encoded into the
// 1024px display JPEG (see platform/imaging; the thumbnail is skipped — the
// admin queue renders the display size directly) before being persisted under
// a kyc/ key via the configured blob.Storage adapter.
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
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

	url, err := h.store.Upload(r.Context(), "kyc/"+bson.NewObjectID().Hex()+".jpg", "image/jpeg", display)
	if err != nil {
		slog.Error("kyc: document upload failed", "error", err)
		response.InternalError(w)
		return
	}

	response.OK(w, uploadDocumentResponse{ImageURL: url})
}

// writeKycError maps domain errors to HTTP responses (404 missing, 400
// bad-request, 500 otherwise). Shared by the customer and admin handlers.
func writeKycError(w http.ResponseWriter, err error) {
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
