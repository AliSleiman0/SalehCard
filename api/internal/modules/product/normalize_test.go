package product

import (
	"encoding/json"
	"strings"
	"testing"
)

// A legacy-import document can lack the images field entirely; the decoded nil
// slice must marshal as [] — a JSON null crashes clients indexing images[0].
func TestNormalize_NilImagesMarshalAsEmptyArray(t *testing.T) {
	var p Product // decoded doc with no images field → nil slice

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"images":null`) {
		t.Fatalf("precondition changed: nil Images no longer marshals as null (%s)", b)
	}

	p.normalize()
	b, err = json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal after normalize: %v", err)
	}
	if !strings.Contains(string(b), `"images":[]`) {
		t.Fatalf(`normalized product = %s, want "images":[]`, b)
	}
}

// normalize must not clobber real image URLs.
func TestNormalize_KeepsExistingImages(t *testing.T) {
	p := Product{Images: []string{"http://cdn.test/a.jpg"}}
	p.normalize()
	if len(p.Images) != 1 || p.Images[0] != "http://cdn.test/a.jpg" {
		t.Fatalf("images = %v, want the original single URL", p.Images)
	}
}
