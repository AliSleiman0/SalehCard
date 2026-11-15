package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// SupplierOption is one of the order's product supplier links as seen from
// this order: what it would cost to send this order there, and whether it can
// be sent there at all (the old admin's per-order supplier list).
type SupplierOption struct {
	LinkID       string `json:"linkId"`
	Provider     int    `json:"provider"`
	ProviderName string `json:"providerName,omitempty"`
	UpstreamID   string `json:"upstreamId"`
	Name       string `json:"name,omitempty"`
	Enabled    bool   `json:"enabled"`
	Active     bool   `json:"active"`
	// Current = the order's current attempt went to this supplier product;
	// Tried = an earlier attempt of this order did.
	Current bool `json:"current"`
	Tried   bool `json:"tried"`
	// CatalogStatus: ok | unknown (not in the supplier's list) | unavailable
	// (list couldn't be loaded) | not_checked (supplier can't list products).
	CatalogStatus string   `json:"catalogStatus"`
	Available     bool     `json:"available"`
	Stale         bool     `json:"stale,omitempty"`
	UnitPrice     *float64 `json:"unitPrice,omitempty"`
	Currency      string   `json:"currency,omitempty"`
	// Cost is UnitPrice × the quantity this order sends, in Currency.
	Cost  *float64 `json:"cost,omitempty"`
	QtyOK bool     `json:"qtyOk"`
	// LowBalance: the supplier's known, non-negative balance is below Cost
	// (sending needs an explicit "send anyway"). Balance itself is only
	// returned to admins who may see supplier accounts.
	LowBalance      bool     `json:"lowBalance,omitempty"`
	Balance         *float64 `json:"balance,omitempty"`
	BalanceCurrency string   `json:"balanceCurrency,omitempty"`
	// Blocked explains why this link can't take the order ("" = it can).
	Blocked string `json:"blocked,omitempty"`
}

// SupplierOptions is GET /admin/orders/{id}/supplier-options.
type SupplierOptions struct {
	// CanSend/Reason/Message: whether the order itself may be sent (eligibility).
	CanSend bool   `json:"canSend"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
	// FromManual: a manual order that was never sent to a supplier.
	FromManual bool `json:"fromManual"`
	// NeedsAck: a supplier may still deliver it (AcknowledgeAtSupplier);
	// Ambiguous: its last outcome is unknown (AcknowledgeAmbiguous).
	NeedsAck  bool `json:"needsAck"`
	Ambiguous bool `json:"ambiguous"`
	// Qty is what a supplier is sent; LinePrice what the customer paid for
	// the line (the margin base), in the order currency.
	Qty       int     `json:"qty"`
	LinePrice float64 `json:"linePrice"`
	Currency  string  `json:"currency"`
	// ProductMissing: the line's product no longer exists (no links to list).
	ProductMissing bool             `json:"productMissing,omitempty"`
	Options        []SupplierOption `json:"options"`
}

// optionsLine is the line a supplier would be sent (the api line, else the
// single manual line); -1 when the order has none.
func optionsLine(o *Order) int {
	for i, it := range o.Items {
		if it.FulfillmentMode == string(product.FulfillmentModeAPI) {
			return i
		}
	}
	if len(o.Items) == 1 {
		return 0
	}
	return -1
}

// SupplierOptions lists the order's product supplier links with this order's
// cost at each, availability, quantity fit and balance, plus whether the order
// can be sent and which acknowledgements that needs. showBalance returns the
// balances themselves (suppliers.view).
func (s *SupplierSettler) SupplierOptions(ctx context.Context, id bson.ObjectID, showBalance bool) (*SupplierOptions, error) {
	o, err := s.svc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	out := &SupplierOptions{Currency: o.Currency, Options: []SupplierOption{}}
	ref := o.Fulfillment.ProviderRef
	out.Ambiguous = strings.HasPrefix(ref, "search:")
	out.NeedsAck = !out.Ambiguous && orderAtSupplier(o)
	if _, fromManual, err := rerouteEligible(o); err != nil {
		var ae *apperrors.AppError
		if errors.As(err, &ae) {
			out.Reason, out.Message = ae.Code, ae.Message
		}
	} else {
		out.CanSend, out.FromManual = true, fromManual
	}

	idx := optionsLine(o)
	if idx < 0 {
		return out, nil
	}
	it := o.Items[idx]
	out.LinePrice = it.Price * float64(max(it.Qty, 1))
	qty, qtyOK := supplierQty(it)
	out.Qty = qty
	if !qtyOK && out.CanSend {
		out.CanSend = false
		out.Reason, out.Message = "REROUTE_BAD_QUANTITY", "this order's quantity can't be sent to a supplier — complete or refund it by hand"
	}

	links, active, err := s.productLinks(ctx, it)
	if err != nil {
		return nil, err
	}
	if links == nil && !it.ProductID.IsZero() {
		if _, ferr := s.svc.products.FindByID(ctx, it.ProductID.Hex()); errors.Is(ferr, apperrors.ErrNotFound) {
			out.ProductMissing = true
		}
	}
	var curProv *int
	curUp := ""
	if !out.FromManual && it.FulfillmentMode == string(product.FulfillmentModeAPI) {
		curProv, curUp = it.FulfillmentProvider, it.UpstreamProductID
	}

	out.Options = make([]SupplierOption, len(links))
	var wg sync.WaitGroup
	for i, l := range links {
		opt := SupplierOption{
			LinkID: l.ID, Provider: l.Provider, UpstreamID: l.UpstreamID,
			Enabled: l.Enabled, Active: l.ID == active,
			Current: curProv != nil && *curProv == l.Provider && curUp == l.UpstreamID,
		}
		for _, a := range o.Fulfillment.SupplierAttempts {
			if a.Provider != nil && *a.Provider == l.Provider && a.UpstreamProductID == l.UpstreamID {
				opt.Tried = true
			}
		}
		out.Options[i] = opt
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.quoteOption(ctx, &out.Options[i], qty, qtyOK, showBalance)
		}(i)
	}
	wg.Wait()
	return out, nil
}

// quoteOption fills one option's live supplier data and its Blocked reason
// (the same refusals Reroute would give, worded for the list).
func (s *SupplierSettler) quoteOption(ctx context.Context, opt *SupplierOption, qty int, qtyOK, showBalance bool) {
	opt.CatalogStatus = "not_checked"
	opt.QtyOK = qtyOK
	switch {
	case !s.svc.providers.Has(opt.Provider):
		opt.Blocked = "this supplier isn't connected"
		return
	case strings.TrimSpace(opt.UpstreamID) == "":
		opt.Blocked = "no supplier product linked — link one in the product editor"
		return
	}
	if s.probes == nil {
		opt.Available = true
	} else {
		cp, found, snap, err := s.probes.Lookup(ctx, opt.Provider, opt.UpstreamID)
		switch {
		case errors.Is(err, provider.ErrNotImplemented):
			opt.Available = true
		case err != nil:
			opt.CatalogStatus = "unavailable"
			opt.Blocked = "couldn't load this supplier's product list — try again in a minute"
		case !found:
			opt.CatalogStatus = "unknown"
			opt.Blocked = "not in this supplier's product list any more"
		default:
			opt.CatalogStatus = "ok"
			opt.Stale = snap.Stale
			opt.Name = cp.Name
			opt.Available = cp.Available
			opt.Currency = cp.Currency
			if cp.Price > 0 {
				unit := cp.Price
				cost := unit * float64(qty)
				opt.UnitPrice, opt.Cost = &unit, &cost
			}
			opt.QtyOK = qtyOK && provider.AcceptsQty(cp, qty)
			switch {
			case !cp.Available:
				opt.Blocked = "unavailable at this supplier right now"
			case !opt.QtyOK:
				opt.Blocked = fmt.Sprintf("doesn't accept quantity %s", groupThousands(qty))
			}
		}
		if acc, berr := s.probes.Balance(ctx, opt.Provider); berr == nil {
			if showBalance {
				b := acc.Balance
				opt.Balance, opt.BalanceCurrency = &b, acc.Currency
			}
			if opt.Cost != nil && acc.Balance >= 0 && acc.Balance < *opt.Cost {
				opt.LowBalance = true
			}
		}
	}
	if opt.Blocked == "" && !opt.Enabled {
		opt.Blocked = "switched off in the product's supplier list"
	}
}

// groupThousands formats 45000 as "45,000".
func groupThousands(n int) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// MaxBulkReroute caps one bulk re-send: each order is dispatched to a live,
// billed supplier, so a mistaken selection must stay small.
const MaxBulkReroute = 5

// BulkRerouteInput is POST /admin/orders/reroute-bulk.
type BulkRerouteInput struct {
	IDs []string `json:"ids"`
	// ProviderID (optional) sends every order to its product's link at this
	// supplier; without it each order goes to its first usable link (the
	// active one first, then the backup order), skipping supplier products
	// this order already went to.
	ProviderID            int  `json:"providerId,omitempty"`
	AcknowledgeAtSupplier bool `json:"acknowledgeAtSupplier"`
	AcknowledgeAmbiguous  bool `json:"acknowledgeAmbiguous"`
	AcknowledgeLowBalance bool `json:"acknowledgeLowBalance"`
}

// BulkRerouteResult is one order's outcome.
type BulkRerouteResult struct {
	ID string `json:"id"`
	// Status: completed | sent | refused (the supplier rejected it — stuck
	// again) | needs_ack | no_link | not_eligible | conflict | error.
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	Provider   int    `json:"provider,omitempty"`
	UpstreamID string `json:"upstreamId,omitempty"`
}

// errBulkLimit: more ids than MaxBulkReroute.
var errBulkLimit = rerouteErr("REROUTE_BULK_LIMIT", fmt.Sprintf("re-send at most %d orders at once", MaxBulkReroute), apperrors.ErrBadRequest)

// BulkReroute re-sends up to MaxBulkReroute orders. Claims run one after
// another (each its own guarded FindOneAndUpdate — a failure on one order
// never touches the rest); the claimed orders are then dispatched together,
// each on a detached context, and their outcome read back.
func (s *SupplierSettler) BulkReroute(ctx context.Context, in BulkRerouteInput, actor string) ([]BulkRerouteResult, error) {
	if len(in.IDs) == 0 {
		return nil, rerouteErr("REROUTE_BULK_EMPTY", "no orders selected", apperrors.ErrBadRequest)
	}
	if len(in.IDs) > MaxBulkReroute {
		return nil, errBulkLimit
	}
	if in.ProviderID != 0 && !s.svc.providers.Has(in.ProviderID) {
		return nil, rerouteErr("REROUTE_UNKNOWN_SUPPLIER", "that supplier is not configured", apperrors.ErrBadRequest)
	}
	opts := RerouteInput{
		AcknowledgeAtSupplier: in.AcknowledgeAtSupplier,
		AcknowledgeAmbiguous:  in.AcknowledgeAmbiguous,
		AcknowledgeLowBalance: in.AcknowledgeLowBalance,
	}
	results := make([]BulkRerouteResult, len(in.IDs))
	var claimed []*Order
	claimedAt := map[bson.ObjectID]int{}
	seen := map[bson.ObjectID]bool{}
	for i, raw := range in.IDs {
		results[i] = BulkRerouteResult{ID: raw}
		id, err := bson.ObjectIDFromHex(strings.TrimSpace(raw))
		if err != nil {
			results[i].Status, results[i].Message = "error", "invalid id"
			continue
		}
		if seen[id] {
			results[i].Status, results[i].Message = "conflict", "listed twice"
			continue
		}
		seen[id] = true
		o, t, err := s.bulkClaimOne(ctx, id, in.ProviderID, opts, actor)
		if err != nil {
			results[i].Status, results[i].Message = bulkStatus(err)
			continue
		}
		results[i].Provider, results[i].UpstreamID = t.Provider, t.Upstream
		claimed = append(claimed, o)
		claimedAt[o.ID] = i
	}

	var wg sync.WaitGroup
	for _, o := range claimed {
		wg.Add(1)
		go func(o *Order) {
			defer wg.Done()
			s.dispatchDetached(ctx, o)
		}(o)
	}
	wg.Wait()

	for _, o := range claimed {
		i := claimedAt[o.ID]
		results[i].Status = "sent"
		cur, err := s.svc.repo.FindByID(ctx, o.ID)
		switch {
		case err != nil:
		case cur.Status == OrderStatusCompleted:
			results[i].Status = "completed"
		case cur.Fulfillment.SupplierStuckAt != nil:
			results[i].Status, results[i].Message = "refused", "the supplier refused it — try another supplier or refund"
		}
	}
	return results, nil
}

// bulkClaimOne picks the target for one order of a bulk re-send and claims it.
func (s *SupplierSettler) bulkClaimOne(ctx context.Context, id bson.ObjectID, providerID int, in RerouteInput, actor string) (*Order, rerouteTarget, error) {
	o, err := s.svc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, rerouteTarget{}, err
	}
	prep, err := rerouteCheck(o, in)
	if err != nil {
		return nil, rerouteTarget{}, err
	}
	links, active, err := s.productLinks(ctx, o.Items[prep.idx])
	if err != nil {
		return nil, rerouteTarget{}, err
	}
	cands := bulkCandidates(o, prep, links, active, providerID)
	if len(cands) == 0 {
		return nil, rerouteTarget{}, errNoLink
	}
	var last error
	for _, l := range cands {
		t, err := s.linkTarget(l)
		if err != nil {
			last = err
			continue
		}
		claimed, err := s.claimReroute(ctx, o, prep, t, in, actor, true)
		if err == nil {
			return claimed, t, nil
		}
		if !targetRefusal(err) {
			return nil, rerouteTarget{}, err
		}
		last = err
	}
	return nil, rerouteTarget{}, fmt.Errorf("%w: %w", errNoLink, last)
}

// errNoLink: no product supplier link could take the order.
var errNoLink = errors.New("no linked supplier can take this order")

// bulkCandidates orders the links a bulk re-send may use for o: the active
// link first, then the rest in the admin's backup order; only enabled links,
// only providerID's when given, and (without providerID) never the supplier
// product the current or an earlier attempt already went to.
func bulkCandidates(o *Order, prep reroutePrep, links []product.SupplierLink, active string, providerID int) []product.SupplierLink {
	it := o.Items[prep.idx]
	used := func(l product.SupplierLink) bool {
		if !prep.fromManual && it.FulfillmentProvider != nil && *it.FulfillmentProvider == l.Provider && it.UpstreamProductID == l.UpstreamID {
			return true
		}
		for _, a := range o.Fulfillment.SupplierAttempts {
			if a.Provider != nil && *a.Provider == l.Provider && a.UpstreamProductID == l.UpstreamID {
				return true
			}
		}
		return false
	}
	var first, rest []product.SupplierLink
	for _, l := range links {
		switch {
		case !l.Enabled:
		case providerID != 0 && l.Provider != providerID:
		case providerID == 0 && used(l):
		case l.ID == active:
			first = append(first, l)
		default:
			rest = append(rest, l)
		}
	}
	return append(first, rest...)
}

// targetRefusal: the error is about the chosen supplier product (try the next
// link), not about the order.
func targetRefusal(err error) bool {
	var ae *apperrors.AppError
	if !errors.As(err, &ae) {
		return false
	}
	switch ae.Code {
	case "REROUTE_UPSTREAM_UNKNOWN", "REROUTE_UPSTREAM_UNAVAILABLE", "REROUTE_QTY_REJECTED",
		"REROUTE_CATALOG_UNAVAILABLE", "REROUTE_LOW_BALANCE", "REROUTE_LINK_DISABLED",
		"REROUTE_UPSTREAM_REQUIRED", "REROUTE_UNKNOWN_SUPPLIER":
		return true
	}
	return false
}

// bulkStatus maps a per-order bulk failure to its result status + message.
func bulkStatus(err error) (string, string) {
	var ae *apperrors.AppError
	switch {
	case errors.Is(err, errNoLink):
		msg := "no linked supplier can take it — send it from the order page"
		if errors.As(err, &ae) {
			msg = "no linked supplier can take it: " + ae.Message
		}
		return "no_link", msg
	case errors.As(err, &ae) && (ae.Code == "REROUTE_NEEDS_ACK" || ae.Code == "REROUTE_AMBIGUOUS"):
		return "needs_ack", ae.Message
	case errors.As(err, &ae) && ae.Code == "REROUTE_CONFLICT":
		return "conflict", ae.Message
	case errors.As(err, &ae):
		return "not_eligible", ae.Message
	case errors.Is(err, apperrors.ErrNotFound):
		return "error", "not found"
	default:
		return "error", "internal error"
	}
}
