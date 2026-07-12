package main

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// TestInputFieldRoundTripPreservesMetadata proves that decoding a full input
// field into our inputField struct, editing label.en, and re-encoding preserves
// every other property (type, sensitive, constraints, legacyName) — the exact
// operation the -apply path performs on each product's inputFields array.
func TestInputFieldRoundTripPreservesMetadata(t *testing.T) {
	// A representative field as stored in Mongo: a masked password field with a
	// legacyName and a select constraint, Arabic sitting in label.en.
	original := bson.M{
		"key":        "field_2",
		"label":      bson.M{"en": "ادخل كلمة السر", "ar": "ادخل كلمة السر"},
		"type":       "text",
		"sensitive":  true,
		"legacyName": "password",
		"constraints": bson.M{
			"options": bson.A{"a", "b"},
		},
	}
	raw, err := bson.Marshal(original)
	if err != nil {
		t.Fatalf("marshal original: %v", err)
	}

	var f inputField
	if err := bson.Unmarshal(raw, &f); err != nil {
		t.Fatalf("unmarshal into inputField: %v", err)
	}

	// Sanity: the fields we act on decoded correctly.
	if f.Key != "field_2" || f.Label.En != "ادخل كلمة السر" {
		t.Fatalf("key/label did not decode: key=%q en=%q", f.Key, f.Label.En)
	}

	// The edit -apply makes.
	f.Label.En = "Enter password"

	out, err := bson.Marshal(f)
	if err != nil {
		t.Fatalf("marshal edited: %v", err)
	}
	var got bson.M
	if err := bson.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if got["type"] != "text" {
		t.Errorf("type dropped/changed: %v", got["type"])
	}
	if got["sensitive"] != true {
		t.Errorf("sensitive flag lost — a password field would become visible: %v", got["sensitive"])
	}
	if got["legacyName"] != "password" {
		t.Errorf("legacyName lost: %v", got["legacyName"])
	}
	// Nested docs decode as bson.D through the inline map — values are intact,
	// only the Go container type differs (D vs M).
	c, ok := got["constraints"].(bson.D)
	if !ok {
		t.Fatalf("constraints lost or wrong type: %#v", got["constraints"])
	}
	if lookup(c, "options") == nil {
		t.Errorf("constraints.options lost: %#v", c)
	}

	// The label edit took, ar untouched.
	lbl, _ := got["label"].(bson.D)
	if lookup(lbl, "en") != "Enter password" {
		t.Errorf("label.en not updated: %v", lookup(lbl, "en"))
	}
	if lookup(lbl, "ar") != "ادخل كلمة السر" {
		t.Errorf("label.ar changed: %v", lookup(lbl, "ar"))
	}
}

func lookup(d bson.D, key string) any {
	for _, e := range d {
		if e.Key == key {
			return e.Value
		}
	}
	return nil
}
