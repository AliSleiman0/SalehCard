package kyc

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

// jpegBytes encodes a tiny opaque JPEG the imaging pipeline accepts.
func jpegBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			img.Set(x, y, color.RGBA{R: 90, G: 160, B: 220, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

// newDocUpload builds a multipart POST carrying data under the given form
// field. Pass field "image" for the real field; any other name to exercise the
// missing-field path.
func newDocUpload(t *testing.T, field string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile(field, "document.jpg")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/kyc/documents", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestUploadDocument_HappyPath(t *testing.T) {
	store := &fakeStorage{}
	h := &Handler{store: store} // UploadDocument never touches the service

	w := httptest.NewRecorder()
	h.UploadDocument(w, newDocUpload(t, "image", jpegBytes(t)))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	var env struct {
		Data uploadDocumentResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	if env.Data.ImageURL == "" {
		t.Fatalf("response = %+v, want a non-empty imageUrl", env.Data)
	}

	// Exactly one blob (display only — no thumbnail) under the kyc/ prefix.
	if len(store.keys) != 1 {
		t.Fatalf("uploaded %d blobs, want 1: %v", len(store.keys), store.keys)
	}
	key := store.keys[0]
	if !strings.HasPrefix(key, "kyc/") || !strings.HasSuffix(key, ".jpg") {
		t.Fatalf("key = %q, want kyc/<hex>.jpg", key)
	}
	if !validDocURL(env.Data.ImageURL) {
		t.Fatalf("imageUrl = %q, want it to pass validDocURL (what Submit enforces)", env.Data.ImageURL)
	}
}

func TestUploadDocument_MissingImageField(t *testing.T) {
	h := &Handler{store: &fakeStorage{}}
	w := httptest.NewRecorder()
	h.UploadDocument(w, newDocUpload(t, "wrongfield", jpegBytes(t)))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "image") {
		t.Fatalf("body = %q, want it to mention the missing \"image\" field", w.Body.String())
	}
}

func TestUploadDocument_NotMultipart(t *testing.T) {
	h := &Handler{store: &fakeStorage{}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/kyc/documents",
		strings.NewReader(`{"not":"multipart"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UploadDocument(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestUploadDocument_InvalidImage(t *testing.T) {
	h := &Handler{store: &fakeStorage{}}
	w := httptest.NewRecorder()
	h.UploadDocument(w, newDocUpload(t, "image", []byte("plain text, not an image")))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid image") {
		t.Fatalf("body = %q, want \"invalid image\"", w.Body.String())
	}
}

func TestUploadDocument_OversizedBody(t *testing.T) {
	h := &Handler{store: &fakeStorage{}}
	// Just over the 10 MB cap → MaxBytesReader trips ParseMultipartForm.
	oversized := bytes.Repeat([]byte{0}, maxUploadBytes+1024)
	w := httptest.NewRecorder()
	h.UploadDocument(w, newDocUpload(t, "image", oversized))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for an over-cap body", w.Code)
	}
}

func TestUploadDocument_StoreFailure(t *testing.T) {
	// Silence the expected error log.
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	defer slog.SetDefault(prev)

	store := &fakeStorage{err: errors.New("blob backend down")}
	h := &Handler{store: store}
	w := httptest.NewRecorder()
	h.UploadDocument(w, newDocUpload(t, "image", jpegBytes(t)))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 when the store fails", w.Code)
	}
}
