package role

import (
	"slices"
	"testing"
)

func TestNormalizePermissions(t *testing.T) {
	tests := []struct {
		name    string
		in      []string
		want    []string
		wantErr bool
	}{
		{name: "manage implies view", in: []string{"orders.manage"}, want: []string{"orders.manage", "orders.view"}},
		{name: "dedup", in: []string{"kyc.view", "kyc.view"}, want: []string{"kyc.view"}},
		{name: "view alone stays", in: []string{"orders.view"}, want: []string{"orders.view"}},
		{name: "unknown domain rejected", in: []string{"nope.view"}, wantErr: true},
		{name: "wildcard not grantable", in: []string{"*"}, wantErr: true},
		{name: "manage alone not valid perm string", in: []string{"orders"}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := NormalizePermissions(tc.in)
			if tc.wantErr {
				if ok {
					t.Fatalf("expected rejection, got %v", got)
				}
				return
			}
			if !ok {
				t.Fatalf("unexpected rejection of %v", tc.in)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestCatalogPermissionsGrantable guards the invariant the server relies on: every
// catalog domain yields two grantable permissions, so a domain() wrapper in
// server.go can never demand a permission NormalizePermissions would reject.
func TestCatalogPermissionsGrantable(t *testing.T) {
	if len(Domains) == 0 {
		t.Fatal("permission catalog is empty")
	}
	for _, d := range Domains {
		if !ValidPermission(d.Key+".view") || !ValidPermission(d.Key+".manage") {
			t.Fatalf("catalog domain %q does not yield grantable view+manage permissions", d.Key)
		}
	}
}
