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
	for _, k := range []string{"category", "status", "inputFields", "pricing", "images"} {
		if v := valueFor(set, k); v != nil && !isLiteral(v) {
			t.Fatalf("%s not $literal-wrapped: %#v", k, v)
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
