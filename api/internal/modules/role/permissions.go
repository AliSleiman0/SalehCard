package role

import "strings"

// Domain is one admin console area that RBAC gates. Every domain yields two
// permissions: "<key>.view" (GET/HEAD) and "<key>.manage" (mutations). The
// catalog is the single source of truth — server middleware, role validation,
// and the admin console's role editor all derive from it.
type Domain struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Domains is the full permission catalog, one entry per admin module / nav
// area. Role management itself is deliberately absent: it is super-admin-only
// and never grantable to a custom role.
var Domains = []Domain{
	{Key: "dashboard", Label: "Dashboard"},
	{Key: "products", Label: "Products"},
	{Key: "categories", Label: "Categories"},
	{Key: "inventory", Label: "Inventory"},
	{Key: "offers", Label: "Offers"},
	{Key: "orders", Label: "Orders"},
	{Key: "bridge", Label: "Bridge"},
	{Key: "suppliers", Label: "Suppliers"},
	{Key: "users", Label: "Users"},
	{Key: "resellers", Label: "Resellers"},
	{Key: "kyc", Label: "KYC"},
	{Key: "finance", Label: "Finance"},
	{Key: "topups", Label: "Top-ups"},
	{Key: "payments", Label: "Payments"},
	{Key: "promos", Label: "Promos"},
	{Key: "expenses", Label: "Expenses"},
	{Key: "reviews", Label: "Reviews"},
	{Key: "audit", Label: "Audit log"},
	{Key: "settings", Label: "Settings"},
}

// validPermissions is the set of grantable permission strings.
var validPermissions = func() map[string]bool {
	m := make(map[string]bool, len(Domains)*2)
	for _, d := range Domains {
		m[d.Key+".view"] = true
		m[d.Key+".manage"] = true
	}
	return m
}()

// ValidPermission reports whether p is a grantable permission. The super-admin
// wildcard "*" is NOT grantable — super admin exists only implicitly (an admin
// with no custom role assigned).
func ValidPermission(p string) bool { return validPermissions[p] }

// NormalizePermissions validates and dedupes perms, preserving order. It also
// grants "<domain>.view" implicitly whenever "<domain>.manage" is present, so a
// role can never mutate an area it cannot read. The bool is false when any
// entry is not a grantable permission.
func NormalizePermissions(perms []string) ([]string, bool) {
	seen := make(map[string]bool, len(perms))
	out := make([]string, 0, len(perms))
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	for _, p := range perms {
		if !ValidPermission(p) {
			return nil, false
		}
		add(p)
		if domain, ok := strings.CutSuffix(p, ".manage"); ok {
			add(domain + ".view")
		}
	}
	return out, true
}
