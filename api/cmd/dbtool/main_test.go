package main

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMask(t *testing.T) {
	cases := map[string]string{
		"ABCD1234-XYZ-6991": "…6991",
		"6991":              "****",
		"":                  "****",
		"ab":                "****",
	}
	for in, want := range cases {
		if got := mask(in); got != want {
			t.Errorf("mask(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsProdURI(t *testing.T) {
	prod := []string{
		"mongodb+srv://u:p@docdb-cluster.mongocluster.cosmos.azure.com/?tls=true",
		"mongodb://x.cosmos.azure.com:10255",
	}
	for _, u := range prod {
		if !isProdURI(u) {
			t.Errorf("isProdURI(%q) = false, want true", u)
		}
	}
	if isProdURI("mongodb://localhost:27017") {
		t.Errorf("isProdURI(localhost) = true, want false")
	}
}

func TestTallyStatus(t *testing.T) {
	got := tallyStatus([]bson.M{
		{"status": "available"},
		{"status": "delivered"},
		{"status": "available"},
	})
	if got["available"] != 2 || got["delivered"] != 1 {
		t.Errorf("tallyStatus = %v, want available:2 delivered:1", got)
	}
}

func TestGrantPerms(t *testing.T) {
	got, err := grantPerms("complaints", "manage")
	if err != nil || len(got) != 2 || got[0] != "complaints.view" || got[1] != "complaints.manage" {
		t.Fatalf("manage: %v %v", got, err)
	}
	if got, _ := grantPerms("complaints", "view"); len(got) != 1 || got[0] != "complaints.view" {
		t.Fatalf("view: %v", got)
	}
	if _, err := grantPerms("nope", "view"); err == nil {
		t.Error("unknown area must be refused")
	}
	if _, err := grantPerms("complaints", "admin"); err == nil {
		t.Error("unknown level must be refused")
	}
}

func TestMissingPermsAndReorder(t *testing.T) {
	m := missingPerms([]string{"orders.view", "complaints.view"}, []string{"complaints.view", "complaints.manage"})
	if len(m) != 1 || m[0] != "complaints.manage" {
		t.Fatalf("missing: %v", m)
	}
	r := reorderFlags([]string{"complaints", "manage", "all", "--apply"})
	if r[0] != "--apply" || r[3] != "all" {
		t.Fatalf("reorder: %v", r)
	}
}
