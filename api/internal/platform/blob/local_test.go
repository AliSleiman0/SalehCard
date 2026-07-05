package blob

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew_ProviderSwitch(t *testing.T) {
	t.Run("empty provider builds local", func(t *testing.T) {
		s, err := New(Config{Provider: ""})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if _, ok := s.(*LocalStorage); !ok {
			t.Fatalf("New(\"\") = %T, want *LocalStorage", s)
		}
	})

	t.Run("local provider builds local", func(t *testing.T) {
		s, err := New(Config{Provider: "local"})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if _, ok := s.(*LocalStorage); !ok {
			t.Fatalf("New(\"local\") = %T, want *LocalStorage", s)
		}
	})

	t.Run("azure without credentials errors", func(t *testing.T) {
		_, err := New(Config{Provider: "azure"})
		if err == nil {
			t.Fatal("New(azure) with empty config: want error, got nil")
		}
		if !strings.Contains(err.Error(), "connection string and container are required") {
			t.Fatalf("New(azure) error = %q, want it to mention the missing creds", err)
		}
	})

	t.Run("unknown provider errors", func(t *testing.T) {
		_, err := New(Config{Provider: "s3"})
		if err == nil {
			t.Fatal("New(s3): want error, got nil")
		}
		if !strings.Contains(err.Error(), "unknown storage provider") {
			t.Fatalf("New(s3) error = %q, want \"unknown storage provider\"", err)
		}
	})
}

func TestLocalStorage_Upload(t *testing.T) {
	dir := t.TempDir()
	// Trailing slash on PublicBase must be trimmed so the built URL has no
	// double slash before /uploads.
	s := NewLocal(LocalConfig{Dir: dir, PublicBase: "http://x:8090/"})

	const key = "products/abc.jpg"
	want := []byte("fake-jpeg-bytes")

	url, err := s.Upload(context.Background(), key, "image/jpeg", want)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if url != "http://x:8090/uploads/products/abc.jpg" {
		t.Fatalf("URL = %q, want http://x:8090/uploads/products/abc.jpg", url)
	}

	// The nested products/ dir must have been created and hold the exact bytes.
	got, err := os.ReadFile(filepath.Join(dir, "products", "abc.jpg"))
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("written bytes = %q, want %q", got, want)
	}
}

func TestLocalStorage_Upload_DefaultDir(t *testing.T) {
	// NewLocal with an empty Dir falls back to defaultUploadsDir; assert the
	// infallible ctor yields a usable adapter (write into a temp cwd-free path
	// by giving an explicit dir is covered above — here just prove no panic and
	// a URL shape with an empty PublicBase).
	s := NewLocal(LocalConfig{Dir: t.TempDir()})
	url, err := s.Upload(context.Background(), "products/x.jpg", "image/jpeg", []byte("z"))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if url != "/uploads/products/x.jpg" {
		t.Fatalf("URL = %q, want relative /uploads/products/x.jpg when PublicBase empty", url)
	}
}
