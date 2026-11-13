package photoupload

import "testing"

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
