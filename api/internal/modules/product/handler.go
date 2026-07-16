package product

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/sync/errgroup"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/blob"
	"github.com/AliSleiman0/salehcard/api/internal/platform/idcheck"
	"github.com/AliSleiman0/salehcard/api/internal/platform/imaging"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// maxUploadBytes caps a product image upload. No size-limit middleware exists
// in the repo, so this is the sole guard against an oversized request body.
const maxUploadBytes = 10 << 20 // 10 MB

// Handler exposes product domain operations over HTTP.
type Handler struct {
	svc Service
	// rec records destructive admin actions; nil on the public (read-only)
	// registration, set by RegisterAdminRoutes.
	rec audit.Recorder
	// verifier resolves game player IDs for VerifyAccount; nil on the admin
	// registration (verify is a public/customer route), set by RegisterRoutes.
	verifier idcheck.Verifier
	// store persists uploaded product images; nil on the public (read-only)
	// registration, set by RegisterAdminRoutes.
	store blob.Storage
}

// NewHandler constructs a Handler backed by the given service.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// List handles GET /api/v1/products
// Query params: category (string), rootDomain (string), available (bool),
// page (int), limit (int).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var f ListFilter
	f.Category = q.Get("category")
	f.CategoryID = q.Get("categoryId")
	f.RootDomain = q.Get("rootDomain")
	f.Search = q.Get("q")

	if raw := q.Get("directOnly"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			response.BadRequest(w, "directOnly must be a boolean")
			return
		}
		f.CategoryDirect = v
	}

	if raw := q.Get("available"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			response.BadRequest(w, "available must be a boolean")
			return
		}
		f.Available = &v
	}

	p := pagination.ParseParams(r)

	products, total, err := h.svc.FindAll(r.Context(), f, p)
	if err != nil {
		response.InternalError(w)
		return
	}

	meta := pagination.CalcMeta(p, total)
	response.OKWithMeta(w, products, meta)
}

// GetByID handles GET /api/v1/products/{id}.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	product, err := h.svc.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, product)
}

// verifyAccountRequest is the POST /products/{id}/verify-account body.
type verifyAccountRequest struct {
	PlayerID string `json:"playerId"`
}

// verifyAccountResponse reports the outcome of an ID-verification lookup. Found is
// false both for a positively-unknown id (Reason "id_not_found") and when the
// upstream check is unavailable (Reason "unavailable"); the client blocks the
// purchase only on the former (fail-open on the latter).
type verifyAccountResponse struct {
	Found    bool   `json:"found"`
	Username string `json:"username,omitempty"`
	Banned   bool   `json:"banned,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// VerifyAccount handles POST /api/v1/products/{id}/verify-account. It resolves the
// supplied player ID against the product's configured verification provider and
// returns the account nickname, so the customer can confirm the account before
// buying. The player ID is a sensitive input — it is never persisted or logged.
func (h *Handler) VerifyAccount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req verifyAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	req.PlayerID = strings.TrimSpace(req.PlayerID)
	if req.PlayerID == "" {
		response.BadRequest(w, "playerId is required")
		return
	}

	product, err := h.svc.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}
	if product.Verification == nil {
		response.Error(w, http.StatusConflict, "VERIFICATION_NOT_CONFIGURED",
			"this product does not require ID verification")
		return
	}
	if h.verifier == nil { // verify not wired — fail open so checkout is never blocked
		response.OK(w, verifyAccountResponse{Found: false, Reason: "unavailable"})
		return
	}

	acct, err := h.verifier.Verify(r.Context(), product.Verification.App, req.PlayerID)
	switch {
	case errors.Is(err, idcheck.ErrIDNotFound):
		response.OK(w, verifyAccountResponse{Found: false, Reason: "id_not_found"})
	case err != nil:
		// Fail open: an upstream/availability error must not block the sale.
		slog.Warn("product: id verification unavailable",
			"product", id, "provider", product.Verification.Provider, "error", err)
		response.OK(w, verifyAccountResponse{Found: false, Reason: "unavailable"})
	default:
		response.OK(w, verifyAccountResponse{Found: true, Username: acct.Username, Banned: acct.Banned})
	}
}

// writeProductError maps a service error to the right HTTP response, mirroring
// the order/wallet handlers: ErrNotFound → 404, ErrBadRequest → 400 (preserving
// the AppError's code+message so a validation failure surfaces its real reason
// instead of an opaque 500), everything else → 500.
func writeProductError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	case errors.Is(err, apperrors.ErrBadRequest):
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			response.Error(w, http.StatusBadRequest, appErr.Code, appErr.Message)
			return
		}
		response.BadRequest(w, err.Error())
	default:
		response.InternalError(w)
	}
}

// Create handles POST /api/v1/products.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in CreateProductInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	product, err := h.svc.Create(r.Context(), in)
	if err != nil {
		writeProductError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, response.Response{
		Success: true,
		Data:    product,
	})
}

// Update handles PATCH /api/v1/products/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var in UpdateProductInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	updated, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		writeProductError(w, err)
		return
	}

	response.OK(w, updated)
}

// Delete handles DELETE /api/v1/products/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}

	if h.rec != nil {
		h.rec.Record(r.Context(), audit.Entry{
			Action:     audit.ActionProductDelete,
			TargetType: "product",
			TargetID:   id,
		})
	}
	w.WriteHeader(http.StatusNoContent)
}

// uploadImageResponse is the POST /api/admin/products/images reply: the two
// compressed sizes the caller stores on the product (images[0] + thumbnail).
type uploadImageResponse struct {
	ImageURL     string `json:"imageUrl"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

// UploadImage handles POST /api/admin/products/images. It is product-agnostic
// (no product id required) so the editor can upload before a new product
// exists: the caller includes the returned URLs in the create/update
// ProductInput. The uploaded file is validated by content sniffing (not
// filename/header) and re-encoded into a 1024px display JPEG + a 256px
// thumbnail JPEG (see platform/imaging) before being persisted via the
// configured blob.Storage adapter.
func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
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

	display, thumb, err := imaging.Process(data)
	if err != nil {
		response.BadRequest(w, "invalid image: "+err.Error())
		return
	}

	// The two uploads are independent (different keys, no data dependency), so
	// run them concurrently — on the Azure adapter each is a network round trip.
	key := bson.NewObjectID().Hex()
	var imageURL, thumbnailURL string
	g, gctx := errgroup.WithContext(r.Context())
	g.Go(func() error {
		var err error
		imageURL, err = h.store.Upload(gctx, "products/"+key+".jpg", "image/jpeg", display)
		return err
	})
	g.Go(func() error {
		var err error
		thumbnailURL, err = h.store.Upload(gctx, "products/"+key+"_thumb.jpg", "image/jpeg", thumb)
		return err
	})
	if err := g.Wait(); err != nil {
		slog.Error("product: image upload failed", "error", err)
		response.InternalError(w)
		return
	}

	if h.rec != nil {
		h.rec.Record(r.Context(), audit.Entry{
			Action:     audit.ActionProductImageUpload,
			TargetType: "product",
			Summary:    map[string]any{"imageUrl": imageURL},
		})
	}

	response.OK(w, uploadImageResponse{ImageURL: imageURL, ThumbnailURL: thumbnailURL})
}
