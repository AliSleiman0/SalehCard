package blob

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestNewAzureStorage_Validation(t *testing.T) {
	t.Run("missing connection string", func(t *testing.T) {
		_, err := newAzureStorage(AzureConfig{Container: "product-images"})
		if err == nil {
			t.Fatal("want error for empty connection string, got nil")
		}
	})

	t.Run("missing container", func(t *testing.T) {
		_, err := newAzureStorage(AzureConfig{ConnectionString: "x"})
		if err == nil {
			t.Fatal("want error for empty container, got nil")
		}
	})
}

// TestAzureStorage_Upload_Live exercises the real Azure Blob adapter end to end.
// It is skipped unless AZURE_STORAGE_CONNECTION_STRING is set, so `go test ./...`
// and CI stay green without credentials. When run against a real, public-read
// container it proves: the built public URL is correct, the blob is reachable
// over HTTP, and the content-type header round-trips.
//
//	AZURE_STORAGE_CONNECTION_STRING='...' AZURE_STORAGE_CONTAINER='product-images' \
//	  go test ./internal/platform/blob/ -run TestAzureStorage_Upload_Live -v
func TestAzureStorage_Upload_Live(t *testing.T) {
	cs := os.Getenv("AZURE_STORAGE_CONNECTION_STRING")
	if cs == "" {
		t.Skip("set AZURE_STORAGE_CONNECTION_STRING to run the live Azure test")
	}
	container := os.Getenv("AZURE_STORAGE_CONTAINER")
	if container == "" {
		container = "product-images"
	}

	s, err := New(Config{
		Provider: "azure",
		Azure:    AzureConfig{ConnectionString: cs, Container: container},
	})
	if err != nil {
		t.Fatalf("New(azure): %v", err)
	}

	// t.Name() is unique per test run and needs no time/random source (both are
	// unavailable in this repo's deterministic test posture). v1 does not delete
	// blobs on replace, so this leaves a small orphan (pennies) — acceptable.
	key := "products/qa-test-" + strings.ReplaceAll(t.Name(), "/", "_") + ".jpg"
	// A minimal valid JPEG header + payload; the adapter does not decode it.
	want := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}

	url, err := s.Upload(context.Background(), key, "image/jpeg", want)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	wantSuffix := "/" + container + "/" + key
	if !strings.HasPrefix(url, "https://") || !strings.HasSuffix(url, wantSuffix) {
		t.Fatalf("URL = %q, want https://<account>.blob.core.windows.net%s", url, wantSuffix)
	}

	resp, err := http.Get(url) //nolint:gosec // URL is built from our own config
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200 (is the container public-read?)", url, resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Fatalf("Content-Type = %q, want image/jpeg", ct)
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("round-tripped bytes = %v, want %v", got, want)
	}
}
