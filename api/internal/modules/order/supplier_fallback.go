package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/supplieralert"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
)

// Automatic supplier fallback (owner decision 2026-11-15): when the supplier an
// order is sent to DEFINITELY refuses it (provider.Refused: no upstream order
// was created — its balance is empty, it throttled us, it doesn't sell that id,
// …), the order moves on by itself to the product's next supplier link, in the
// admin's order. It never falls back when the first supplier may have taken the
// order: a transport/decode failure (the attempt is then marked ambiguous for
// good), a live or "search:" ref, an attempt an admin chose, or a link this
// order already tried. Every hop is a guarded claim with a fresh idempotency
// key; the old attempt is archived (Auto + Reason) and the give-up clock keeps
// counting from placement.

const (
	// fallbackActor is the audit actor and archive "rerouted by" of every
	// automatic hop.
	fallbackActor = "system:supplier-fallback"
	// maxPlacementHops bounds the synchronous hops while the customer waits at
	// checkout; past it the order parks and the settler continues (one hop per
	// re-dispatch).
	maxPlacementHops = 2
	// maxFallbackReason caps the archived refusal reason.
	maxFallbackReason = 200
)

// SetSupplierFallback wires what the automatic fallback needs: the shared
// supplier catalog cache (to skip a link its supplier lists as unavailable or
// can't sell this quantity — nil skips those checks) and the audit recorder
// (nil records nothing).
func (s *OrderService) SetSupplierFallback(probes *provider.Probes, rec audit.Recorder) {
	s.fbProbes = probes
	s.fbRec = rec
}

// SetSupplierAlerts wires the supplier alerts store: an order whose supplier
// refused it with no other link left to try raises a fallback_exhausted
// alert (nil raises nothing).
func (s *OrderService) SetSupplierAlerts(a supplieralert.Raiser) { s.fbAlerts = a }

// adminRouted reports whether the order's current attempt is one an admin chose
// (a reroute, a manual order sent to a supplier, a bulk re-send) rather than
// placement or an automatic hop. An admin-routed attempt never falls back and
// its definitive rejection waits for the admin instead of refunding.
func adminRouted(o *Order) bool {
	return o.Fulfillment.SupplierAttempt > 0 && !o.Fulfillment.SupplierAttemptAuto
}

// apiLineIndex is the index of the order's supplier line (the one
// supplierFulfillInput dispatches), or -1.
func apiLineIndex(o *Order) int {
	for i, it := range o.Items {
		if it.FulfillmentMode == string(product.FulfillmentModeAPI) {
			return i
		}
	}
	return -1
}

// supplierKey identifies one supplier product.
type supplierKey struct {
	provider int
	upstream string
}

// nextLink picks the link an order falls back to after a definite refusal: the
// first link, in the admin's order, that is enabled, has a supplier product and
// a configured supplier (has), was neither tried on this order nor is the
// current attempt, and passes usable (the live catalog check; nil = no check).
// It returns nothing when the product switched fallback off, the current
// attempt was admin-routed or is ambiguous, or the order holds an upstream ref.
func nextLink(o *Order, p *product.Product, has func(int) bool, usable func(product.SupplierLink) bool) (product.SupplierLink, bool) {
	if p == nil || p.SupplierFallbackOff {
		return product.SupplierLink{}, false
	}
	f := o.Fulfillment
	if adminRouted(o) || f.SupplierAmbiguous || f.ProviderRef != "" {
		return product.SupplierLink{}, false
	}
	idx := apiLineIndex(o)
	if idx < 0 {
		return product.SupplierLink{}, false
	}
	tried := map[supplierKey]bool{}
	if it := o.Items[idx]; it.FulfillmentProvider != nil {
		tried[supplierKey{*it.FulfillmentProvider, strings.TrimSpace(it.UpstreamProductID)}] = true
	}
	for _, a := range f.SupplierAttempts {
		if a.Provider != nil {
			tried[supplierKey{*a.Provider, strings.TrimSpace(a.UpstreamProductID)}] = true
		}
	}
	links, _ := product.EffectiveLinks(p)
	for _, l := range links {
		up := strings.TrimSpace(l.UpstreamID)
		// A link the supplier sync paused (it sells at a loss) is skipped.
		if !l.Enabled || l.Paused() || up == "" || tried[supplierKey{l.Provider, up}] || !has(l.Provider) {
			continue
		}
		if usable != nil && !usable(l) {
			continue
		}
		return l, true
	}
	return product.SupplierLink{}, false
}

// linkUsable is the live-catalog part of nextLink: skip a link whose supplier
// doesn't list that product, marks it unavailable, or doesn't sell this
// quantity. A catalog we can't load (or an adapter without one) doesn't skip
// the link — the worst case is one more refusal.
func (s *OrderService) linkUsable(ctx context.Context, qty int) func(product.SupplierLink) bool {
	if s.fbProbes == nil {
		return nil
	}
	return func(l product.SupplierLink) bool {
		cp, found, _, err := s.fbProbes.Lookup(ctx, l.Provider, l.UpstreamID)
		if err != nil {
			return true
		}
		return found && cp.Available && provider.AcceptsQty(cp, qty)
	}
}

// tryFallback moves o to its product's next supplier link after the definite
// refusal cause, returning the claimed order (nil = no fallback: none
// eligible, or the claim lost a race — the caller then parks as before).
// placing marks the checkout-time path (the order may still be pending).
func (s *OrderService) tryFallback(ctx context.Context, o *Order, cause error, placing bool) *Order {
	if !provider.Refused(cause) {
		return nil
	}
	idx := apiLineIndex(o)
	if idx < 0 || o.Items[idx].ProductID.IsZero() {
		return nil
	}
	qty, ok := supplierQty(o.Items[idx])
	if !ok {
		return nil
	}
	p, err := s.products.FindByID(ctx, o.Items[idx].ProductID.Hex())
	if err != nil {
		return nil
	}
	l, ok := nextLink(o, p, s.providers.Has, s.linkUsable(ctx, qty))
	if !ok {
		s.alertFallbackExhausted(ctx, o, p, cause)
		return nil
	}
	claimed, err := s.claimFallback(ctx, o, idx, l, cause, placing)
	if err != nil {
		slog.Warn("order: supplier fallback claim lost", "order", o.ID.Hex(), "error", err)
		return nil
	}
	return claimed
}

// alertFallbackExhausted raises one open alert per order whose supplier
// definitely refused it with no other link left to try (the order parks and
// keeps retrying). Not for a product with fallback off or an attempt that
// could never fall back (admin-routed, ambiguous, holding a ref).
func (s *OrderService) alertFallbackExhausted(ctx context.Context, o *Order, p *product.Product, cause error) {
	f := o.Fulfillment
	if s.fbAlerts == nil || p == nil || p.SupplierFallbackOff || adminRouted(o) || f.SupplierAmbiguous || f.ProviderRef != "" {
		return
	}
	idx := apiLineIndex(o)
	name := strings.TrimSpace(p.Title.En)
	from := "(none)"
	if idx >= 0 {
		from = supplierLabel(o.Items[idx].FulfillmentProvider)
	}
	err := s.fbAlerts.Raise(ctx, supplieralert.Alert{
		Kind:        supplieralert.KindFallbackExhausted,
		Key:         "fallback:" + o.ID.Hex(),
		OrderID:     o.ID.Hex(),
		ProductID:   p.ID.Hex(),
		ProductName: name,
		Message: fmt.Sprintf("Order %s (%s): supplier %s refused it and no other linked supplier can take it (%s). It keeps retrying — send it to another supplier or fulfil it by hand.",
			strings.ToUpper(o.ID.Hex()[len(o.ID.Hex())-8:]), name, from, fallbackReason(cause)),
		Args: map[string]string{
			"order": strings.ToUpper(o.ID.Hex()[len(o.ID.Hex())-8:]), "product": name,
			"from": from, "reason": fallbackReason(cause),
		},
	})
	if err != nil {
		slog.Warn("order: fallback-exhausted alert", "order", o.ID.Hex(), "error", err)
	}
}

// claimFallback claims the automatic attempt at link l (it does not dispatch).
func (s *OrderService) claimFallback(ctx context.Context, o *Order, idx int, l product.SupplierLink, cause error, placing bool) (*Order, error) {
	old := o.Items[idx]
	now := time.Now().UTC()
	reason := fallbackReason(cause)
	claimed, err := s.repo.ClaimSupplierReroute(ctx, RerouteClaim{
		OrderID:         o.ID,
		ItemIndex:       idx,
		ExpectedAttempt: o.Fulfillment.SupplierAttempt,
		ExpectedRef:     "",
		NewProvider:     l.Provider,
		NewUpstreamID:   l.UpstreamID,
		NewOrderUUID:    bson.NewObjectID().Hex(),
		Archived: &SupplierAttempt{
			Provider:          old.FulfillmentProvider,
			UpstreamProductID: old.UpstreamProductID,
			ProviderRef:       o.Fulfillment.RejectedProviderRef,
			OrderUUID:         supplierOrderUUID(o),
			StuckAt:           o.Fulfillment.SupplierStuckAt,
			ReroutedAt:        now,
			ReroutedBy:        fallbackActor,
			Auto:              true,
			Reason:            reason,
		},
		Event: TimelineEvent{
			Status: "processing",
			Note:   fmt.Sprintf("supplier %s couldn't take the order — trying supplier #%d automatically", supplierLabel(old.FulfillmentProvider), l.Provider),
			At:     now,
		},
		Auto:         true,
		AllowPending: placing,
		KeepClock:    true,
	})
	if err != nil {
		return nil, err
	}
	if s.fbRec != nil {
		where := "settler"
		if placing {
			where = "placement"
		}
		s.fbRec.Record(ctx, audit.Entry{
			ActorID:    fallbackActor,
			ActorEmail: fallbackActor,
			Action:     audit.ActionOrderSupplierFallback,
			TargetType: "order",
			TargetID:   o.ID.Hex(),
			Summary: map[string]any{
				"fromProvider": providerLabel(old.FulfillmentProvider), "fromUpstream": old.UpstreamProductID,
				"toProvider": l.Provider, "toUpstream": l.UpstreamID, "linkId": l.ID,
				"reason": reason, "attempt": claimed.Fulfillment.SupplierAttempt, "at": where,
			},
		})
	}
	return claimed, nil
}

// fallbackReason is the archived, admin-facing reason for a hop.
func fallbackReason(err error) string {
	msg := strings.TrimSuffix(err.Error(), ": "+provider.ErrRefused.Error())
	if len(msg) > maxFallbackReason {
		msg = msg[:maxFallbackReason]
	}
	return msg
}

// supplierLabel renders a supplier id for a timeline note.
func supplierLabel(id *int) string {
	if id == nil {
		return "(none)"
	}
	return fmt.Sprintf("#%d", *id)
}

// retryAmbiguous reports whether a parked dispatch error leaves the attempt
// ambiguous (the supplier may have taken the order): everything except a
// definite refusal and an unwired supplier (stub — nothing was sent).
func retryAmbiguous(err error) bool {
	return !provider.Refused(err) && !errors.Is(err, provider.ErrNotImplemented)
}
