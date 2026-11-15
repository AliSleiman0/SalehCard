package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestPanelOrderKey(t *testing.T) {
	a := panelOrderKey("6af9e7b410669ab1b900319f")
	if !uuidV4.MatchString(a) {
		t.Fatalf("not a UUID v4: %q", a)
	}
	if b := panelOrderKey("6af9e7b410669ab1b900319f"); b != a {
		t.Fatalf("not deterministic: %q vs %q", a, b)
	}
	if b := panelOrderKey("6af9e7b410669ab1b90031a0"); b == a {
		t.Fatalf("different keys collided")
	}
	in := "ECBDD545-E616-4AEE-8770-7EEFA977BCDE"
	if got := panelOrderKey(in); got != "ecbdd545-e616-4aee-8770-7eefa977bcde" {
		t.Fatalf("a UUID must pass through, got %q", got)
	}
}

func TestPanelFulfill_OrderUUIDFormatPerPanel(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query().Get("order_uuid")
		_, _ = w.Write([]byte(`{"status":"OK","data":{"order_id":"1","status":"accept"}}`))
	}))
	defer srv.Close()
	in := fulfillIn()
	in.OrderUUID = "6af9e7b410669ab1b900319f"

	for _, tc := range []struct {
		name string
		uuid bool
	}{{"gift4card", true}, {"jentel", false}, {"speedcard", false}} {
		p, err := NewPanel(PanelConfig{ID: 1, Name: tc.name, BaseURL: srv.URL, Token: "t"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Fulfill(context.Background(), in); err != nil {
			t.Fatal(err)
		}
		if tc.uuid && !uuidV4.MatchString(got) {
			t.Errorf("%s: order_uuid %q, want a UUID v4", tc.name, got)
		}
		if !tc.uuid && got != in.OrderUUID {
			t.Errorf("%s: order_uuid %q, want the raw key", tc.name, got)
		}
	}
}
