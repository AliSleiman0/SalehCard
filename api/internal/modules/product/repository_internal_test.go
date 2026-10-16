package product

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// A "$"-prefixed value (e.g. a product titled "$ 15 Roblox USA") must be stored
// literally, not evaluated as an aggregation field path. buildUpsertSet wraps
// plain data fields in $literal to guarantee that; `variants` and the insert-only
// $ifNull fields must stay real expressions.
func TestBuildUpsertSet_LiteralWrapsDollarValues(t *testing.T) {
	in := UpsertProductInput{
		LegacyID: 741,
		Title:    I18nString{En: "$ 15 Roblox USA", Ar: "$ 15 Roblox USA"},
		Status:   "active",
		Variants: []Variant{{Denomination: "$Default", Price: 15}},
	}
	set := buildUpsertSet(in, time.Now().UTC(), bson.NewObjectID())

	// title is wrapped and preserves the "$"-prefixed value verbatim.
	title := valueFor(set, "title")
	if !isLiteral(title) {
		t.Fatalf("title not $literal-wrapped: %#v", title)
	}
	if got := literalValue(title); got != in.Title {
		t.Fatalf("title literal = %#v, want %#v", got, in.Title)
	}

	// A representative sample of the other data fields is wrapped too.
	for _, k := range []string{"category", "status", "images"} {
		if v := valueFor(set, k); v != nil && !isLiteral(v) {
			t.Fatalf("%s not $literal-wrapped: %#v", k, v)
		}
	}
	// Unit-pricing-guarded fields wrap their refreshed (else) value.
	for _, k := range []string{"inputFields", "pricing", "fulfillmentType", "fulfillmentMode"} {
		if v := condElse(t, set, k); !isLiteral(v) {
			t.Fatalf("%s refreshed value not $literal-wrapped: %#v", k, v)
		}
	}

	// variants stays a real expression ($cond), NOT wrapped — otherwise the
	// existing variant _id / resellerPrice merge would break.
	if isLiteral(valueFor(set, "variants")) {
		t.Fatalf("variants must remain an expression, got $literal")
	}
	// insert-only createdAt stays an $ifNull expression (preserves existing value).
	if isLiteral(valueFor(set, "createdAt")) {
		t.Fatalf("createdAt must stay $ifNull, got $literal")
	}
}

// The category taxonomy filter must match a single node exactly when only the
// expanded-list is empty (the "directOnly" / no-resolver path), and use $in when
// the service has expanded a node to its descendants (the tree-aware path).
func TestBuildFilter_CategoryDirectVsTree(t *testing.T) {
	oid := bson.NewObjectID()

	// Direct: only CategoryID set → exact single-id equality, no $in.
	direct := buildFilter(ListFilter{CategoryID: oid.Hex()})
	v := valueFor(direct, "categoryId")
	if got, ok := v.(bson.ObjectID); !ok || got != oid {
		t.Fatalf("direct categoryId = %#v, want exact %v", v, oid)
	}

	// Tree: CategoryIDs set → $in over the expanded list.
	tree := buildFilter(ListFilter{CategoryID: oid.Hex(), CategoryIDs: []bson.ObjectID{oid}})
	tv := valueFor(tree, "categoryId")
	d, ok := tv.(bson.D)
	if !ok || len(d) != 1 || d[0].Key != "$in" {
		t.Fatalf("tree categoryId = %#v, want $in expression", tv)
	}
}

// A loadseed re-import must not touch what a per-unit product's pricing
// depends on: for a stored doc with unitPricing set, variants, inputFields,
// fulfillmentType/Mode and pricing keep their stored values ($cond on
// unitPricing ≠ null → "$field"); catalog fields like title still refresh.
func TestBuildUpsertSet_PreservesUnitPricedProduct(t *testing.T) {
	in := UpsertProductInput{
		LegacyID:        42,
		Title:           I18nString{En: "Up Live"},
		FulfillmentType: FulfillmentCode,
		InputFields:     []InputField{{Key: "id", Type: InputFieldText}},
		Pricing:         &Pricing{Mode: PricingModeFixedPackage},
		Variants:        []Variant{{Denomination: "10", Price: 100}},
	}
	set := buildUpsertSet(in, time.Now().UTC(), bson.NewObjectID())
	for _, k := range []string{"variants", "inputFields", "fulfillmentType", "fulfillmentMode", "pricing"} {
		cond := condOf(t, set, k)
		ifExpr := cond[0].Value.(bson.D)[0]
		if ifExpr.Key != "$ne" {
			t.Fatalf("%s: if = %#v, want $ne on unitPricing", k, ifExpr)
		}
		ne := ifExpr.Value.(bson.A)
		ifNull := ne[0].(bson.D)[0]
		if ifNull.Key != "$ifNull" || ifNull.Value.(bson.A)[0] != "$unitPricing" || ne[1] != nil {
			t.Fatalf("%s: guard = %#v, want unitPricing ≠ null", k, ne)
		}
		if then := cond[1]; then.Key != "then" || then.Value != "$"+k {
			t.Fatalf("%s: then = %#v, want stored $%s", k, then, k)
		}
	}
	if !isLiteral(valueFor(set, "title")) {
		t.Fatal("title must still refresh unconditionally")
	}
}

// condOf returns the {if, then, else} body of set[key]'s $cond.
func condOf(t *testing.T, set bson.D, key string) bson.D {
	t.Helper()
	d, ok := valueFor(set, key).(bson.D)
	if !ok || len(d) != 1 || d[0].Key != "$cond" {
		t.Fatalf("%s is not a $cond: %#v", key, valueFor(set, key))
	}
	return d[0].Value.(bson.D)
}

// condElse returns the else branch of set[key]'s $cond.
func condElse(t *testing.T, set bson.D, key string) any {
	t.Helper()
	return condOf(t, set, key)[2].Value
}

func valueFor(d bson.D, key string) any {
	for _, e := range d {
		if e.Key == key {
			return e.Value
		}
	}
	return nil
}

func isLiteral(v any) bool {
	d, ok := v.(bson.D)
	return ok && len(d) == 1 && d[0].Key == "$literal"
}

func literalValue(v any) any {
	return v.(bson.D)[0].Value
}
