package provider

import (
	"context"
	"errors"
	"testing"
)

func TestReferenceAdapter_Fulfills(t *testing.T) {
	a := NewReference(7)
	if a.ID() != 7 {
		t.Fatalf("ID = %d, want 7", a.ID())
	}
	res, err := a.Fulfill(context.Background(), FulfillInput{ProductID: "p1", Qty: 2})
	if err != nil {
		t.Fatalf("fulfill error: %v", err)
	}
	if errors.Is(err, ErrNotImplemented) {
		t.Fatal("reference adapter must complete, not park (ErrNotImplemented)")
	}
	if res.Reference == "" {
		t.Fatal("expected a non-empty upstream reference")
	}
}

func TestRegistry_ResolvesReferenceByID(t *testing.T) {
	reg := NewRegistry(NewReference(3))
	id := 3
	if _, err := reg.Resolve(&id).Fulfill(context.Background(), FulfillInput{ProductID: "p"}); err != nil {
		t.Fatalf("registered adapter should fulfill: %v", err)
	}
	// An unregistered id resolves to the stub, which parks.
	other := 99
	if _, err := reg.Resolve(&other).Fulfill(context.Background(), FulfillInput{ProductID: "p"}); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("unknown id should resolve to the parking stub, got %v", err)
	}
}
