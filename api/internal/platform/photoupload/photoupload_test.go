package photoupload

import (
	"context"
	"testing"
)

func TestIsStoredURL(t *testing.T) {
	ok := []string{
		"https://acct.blob.core.windows.net/product-images/order-files/6af0.jpg",
		"http://localhost:8090/uploads/order-files/abc.jpg",
	}
	bad := []string{
		"",
		"order-files/abc.jpg",                       // not absolute
		"javascript:alert(1)//order-files/x.jpg",    // bad scheme
		"https://evil.example/qr.png",               // wrong keyspace + ext
		"https://x/uploads/topups/abc.jpg",          // other keyspace
		"https://x/uploads/order-files/abc.jpg.exe", // wrong suffix
	}
	for _, s := range ok {
		if !IsStoredURL(s, "order-files/") {
			t.Errorf("want ok: %s", s)
		}
	}
	for _, s := range bad {
		if IsStoredURL(s, "order-files/") {
			t.Errorf("want rejected: %s", s)
		}
	}
}

type fakeURLStore struct{ base string }

func (f fakeURLStore) Upload(context.Context, string, string, []byte) (string, error) { return "", nil }
func (f fakeURLStore) Delete(context.Context, string) error                           { return nil }
func (f fakeURLStore) PublicURL(key string) string                                    { return f.base + "/" + key }

func TestNewChecker_PinsHost(t *testing.T) {
	ok := NewChecker(fakeURLStore{base: "https://acct.blob.core.windows.net/product-images"}, "order-files/")
	if !ok("https://acct.blob.core.windows.net/product-images/order-files/6af0.jpg") {
		t.Error("our own upload must pass")
	}
	for _, s := range []string{
		"https://attacker.example/order-files/x.jpg",                                 // external host, same path
		"https://acct.blob.core.windows.net/product-images/kyc/x.jpg",                // other keyspace
		"https://acct.blob.core.windows.net/product-images/order-files/../kyc/x.jpg", // traversal
	} {
		if ok(s) {
			t.Errorf("must reject %s", s)
		}
	}
}
