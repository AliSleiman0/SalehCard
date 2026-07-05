package product

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
)

// fakeStorage records Upload calls and returns canned URLs; when err is set,
// every Upload fails (the store-outage path).
type fakeStorage struct {
	mu   sync.Mutex
	keys []string
	err  error
}

func (f *fakeStorage) Upload(_ context.Context, key, _ string, _ []byte) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return "", f.err
	}
	f.keys = append(f.keys, key)
	return "http://cdn.test/uploads/" + key, nil
}

// fakeRecorder captures the audit entries the handler records.
type fakeRecorder struct {
	mu      sync.Mutex
	entries []audit.Entry
}

func (f *fakeRecorder) Record(_ context.Context, e audit.Entry) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, e)
}

// jpegBytes encodes a tiny opaque JPEG the imaging pipeline accepts.
func jpegBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: 90, G: 160, B: 220, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

// newImageUpload builds a multipart POST carrying data under the given form
// field. Pass field "image" for the real field; any other name to exercise the
// missing-field path.
func newImageUpload(t *testing.T, field string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile(field, "upload.jpg")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/products/images", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestUploadImage_HappyPath(t *testing.T) {
	store := &fakeStorage{}
	rec := &fakeRecorder{}
	h := &Handler{store: store, rec: rec}

	w := httptest.NewRecorder()
	h.UploadImage(w, newImageUpload(t, "image", jpegBytes(t)))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	var env struct {
		Data uploadImageResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	if env.Data.ImageURL == "" || env.Data.ThumbnailURL == "" {
		t.Fatalf("response = %+v, want both URLs non-empty", env.Data)
	}

	// Exactly two blobs, sharing the object-id prefix, with the _thumb suffix on one.
	if len(store.keys) != 2 {
		t.Fatalf("uploaded %d blobs, want 2: %v", len(store.keys), store.keys)
	}
	var display, thumb string
	for _, k := range store.keys {
		if strings.HasSuffix(k, "_thumb.jpg") {
			thumb = k
		} else {
			display = k
		}
	}
	if display == "" || thumb == "" {
		t.Fatalf("keys = %v, want one display + one _thumb", store.keys)
	}
	prefix := strings.TrimSuffix(display, ".jpg")
	if !strings.HasPrefix(display, "products/") || thumb != prefix+"_thumb.jpg" {
		t.Fatalf("keys = %v, want products/<hex>.jpg + products/<hex>_thumb.jpg", store.keys)
	}

	if len(rec.entries) != 1 || rec.entries[0].Action != audit.ActionProductImageUpload {
		t.Fatalf("audit entries = %+v, want one %q", rec.entries, audit.ActionProductImageUpload)
	}
}

func TestUploadImage_MissingImageField(t *testing.T) {
	h := &Handler{store: &fakeStorage{}, rec: &fakeRecorder{}}
	w := httptest.NewRecorder()
	h.UploadImage(w, newImageUpload(t, "wrongfield", jpegBytes(t)))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "image") {
		t.Fatalf("body = %q, want it to mention the missing \"image\" field", w.Body.String())
	}
}

func TestUploadImage_NotMultipart(t *testing.T) {
	h := &Handler{store: &fakeStorage{}, rec: &fakeRecorder{}}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/products/images",
		strings.NewReader(`{"not":"multipart"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UploadImage(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestUploadImage_InvalidImage(t *testing.T) {
	h := &Handler{store: &fakeStorage{}, rec: &fakeRecorder{}}
	w := httptest.NewRecorder()
	h.UploadImage(w, newImageUpload(t, "image", []byte("plain text, not an image")))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid image") {
		t.Fatalf("body = %q, want \"invalid image\"", w.Body.String())
	}
}

func TestUploadImage_OversizedBody(t *testing.T) {
	h := &Handler{store: &fakeStorage{}, rec: &fakeRecorder{}}
	// Just over the 10 MB cap → MaxBytesReader trips ParseMultipartForm.
	oversized := bytes.Repeat([]byte{0}, maxUploadBytes+1024)
	w := httptest.NewRecorder()
	h.UploadImage(w, newImageUpload(t, "image", oversized))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for an over-cap body", w.Code)
	}
}

func TestUploadImage_StoreFailure(t *testing.T) {
	// Silence the expected error log.
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	defer slog.SetDefault(prev)

	store := &fakeStorage{err: errors.New("blob backend down")}
	h := &Handler{store: store, rec: &fakeRecorder{}}
	w := httptest.NewRecorder()
	h.UploadImage(w, newImageUpload(t, "image", jpegBytes(t)))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 when the store fails", w.Code)
	}
}
