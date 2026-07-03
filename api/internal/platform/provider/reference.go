package provider

import (
	"context"
	"fmt"
)

// ReferenceAdapter is a mock upstream fulfillment provider: it "fulfills" any
// api-mode order by returning a synthetic reference, so the api-mode order path
// completes rather than parking — a working example before a real upstream is
// integrated. A real adapter replaces this: implement [Provider], register it in
// the order routes under its numeric ID, and set matching products'
// FulfillmentProvider to that ID. Sensitive inputs (PlayerID) are never logged.
type ReferenceAdapter struct {
	id int
}

// NewReference constructs a ReferenceAdapter that registers under id.
func NewReference(id int) *ReferenceAdapter { return &ReferenceAdapter{id: id} }

// ID is the numeric provider id this adapter registers under.
func (r *ReferenceAdapter) ID() int { return r.id }

// Fulfill returns a synthetic upstream reference for the order line.
func (r *ReferenceAdapter) Fulfill(_ context.Context, in FulfillInput) (Result, error) {
	return Result{Reference: fmt.Sprintf("ref_%s_x%d", in.ProductID, in.Qty)}, nil
}

// Verify returns a placeholder account for the check-name flow.
func (r *ReferenceAdapter) Verify(_ context.Context, _ FulfillInput) (AccountInfo, error) {
	return AccountInfo{Username: "reference-account"}, nil
}
