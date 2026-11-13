// Package photoupload is the shared "customer uploads one photo" HTTP handler:
// multipart field "image" (10 MB cap) → content-sniffed + re-encoded to the
// 1024px display JPEG (platform/imaging) → stored under <prefix><hex>.jpg on
// the configured blob.Storage → {url}. Used for top-up payment proofs
// (topups/) and checkout photo fields such as QR / barcode images
// (order-files/). Callers add auth + rate limiting.
package photoupload

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/platform/blob"
	"github.com/AliSleiman0/salehcard/api/internal/platform/imaging"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// MaxBytes caps one upload (multipart body) at 10 MB.
const MaxBytes = 10 << 20

// Response is the JSON body returned for a stored photo.
type Response struct {
	URL string `json:"url"`
}

// Handler returns an http.HandlerFunc storing one photo under prefix (which
// must end in "/", e.g. "order-files/"). tag labels error logs.
func Handler(store blob.Storage, prefix, tag string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			response.InternalError(w)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, MaxBytes)
		if err := r.ParseMultipartForm(MaxBytes); err != nil {
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
		u, err := store.Upload(r.Context(), prefix+bson.NewObjectID().Hex()+".jpg", "image/jpeg", display)
		if err != nil {
			slog.Error(tag+": photo upload failed", "error", err)
			response.InternalError(w)
			return
		}
		response.OK(w, Response{URL: u})
	}
}

// IsStoredURL reports whether s looks like a photo this package stored under
// prefix: an absolute http(s) URL whose path contains "/<prefix>" and ends in
// ".jpg". It keeps arbitrary customer-typed links out of fields meant for an
// uploaded photo (the same idea as the KYC document check).
func IsStoredURL(s, prefix string) bool {
	if s == "" || len(s) > 512 {
		return false
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false
	}
	return strings.Contains(u.Path, "/"+prefix) && strings.HasSuffix(u.Path, ".jpg")
}
