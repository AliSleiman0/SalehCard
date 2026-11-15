package supplier

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/supplieralert"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
)

// The pure half of the supplier sync job: given one product, every supplier's
// catalog as loaded this pass, and the exchange rates, decide what changes
// (link snapshots, loss pauses, a new active link, hide / re-show) and which
// alerts to raise. No I/O — sync.go applies the result with guarded writes.
//
// Rules (HANDOFF-2026-11-15-SUPPLIER-LINKS.md §7):
//   - never edit a selling price;
//   - a supplier whose catalog didn't load (or only a stale copy) is UNKNOWN
//     and never causes a hide;
//   - hide only when every watched link was checked and none has the
//     product; re-show only what the sync hid (HiddenBySyncAt);
//   - pause a link whose USD cost is at or above our unit selling price; if
//     it was the active link, promote the next eligible one (else hide);
//     clear the pause when the cost recovers, never switching back.

// catalogView is one supplier's catalog as the pass sees it.
type catalogView struct {
	// Known: the catalog loaded fresh this pass (not a stale fallback, not an
	// adapter without a catalog).
	Known    bool
	Products map[string]provider.CatalogProduct
}

// syncEnv is everything evaluateProduct reads besides the product.
type syncEnv struct {
	Catalogs map[int]catalogView
	// FX returns the USD rate for one unit of currency at supplier id
	// (ok=false when none is set). USD is always 1.
	FX    func(provider int, currency string) (float64, bool)
	Names map[int]string // supplier display names
	Now   time.Time
	// Preview = warning-only mode (SUPPLIER_SYNC_DRYRUN): nothing is applied,
	// so alerts say what WOULD happen, in plain words for a non-technical admin.
	Preview bool
}

// Sync change kinds (report + audit).
const (
	changeSnapshot  = "snapshot"
	changePaused    = "loss_paused"
	changeRecovered = "loss_recovered"
	changeUnpaused  = "pause_cleared" // price sync switched off on a paused link
	changePromoted  = "promoted"
	changeHidden    = "hidden"
	changeShown     = "shown"
	changeCost      = "cost_changed"
)

// syncChange is one decided change, for the audit log and the pass report.
type syncChange struct {
	Kind     string         `json:"kind"`
	LinkID   string         `json:"linkId,omitempty"`
	Provider int            `json:"provider,omitempty"`
	Detail   map[string]any `json:"detail,omitempty"`
}

// evalResult is the decision for one product.
type evalResult struct {
	// Links is the links table to store (snapshots and pauses applied);
	// ActiveID the active link after any promotion.
	Links    []product.SupplierLink
	ActiveID string
	// Structural: a pause, unpause or promotion changed the links table, so
	// it is written whole under the rev guard (and the rev bumps). Snapshot-
	// only changes are written per link without touching the rev, so an
	// editor open in the admin is never made stale by a routine check.
	Structural bool
	// Snapshots: link ids whose Sync snapshot changed.
	Snapshots []string
	Hide      bool
	Show      bool
	// HideReason: "loss" | "unavailable" (only when Hide).
	HideReason string
	Alerts     []supplieralert.Alert
	Resolve    []string // open alert keys to close
	Changes    []syncChange
	FxMissing  map[int]string // provider -> currency without a rate
}

// costMoveRel / costMoveAbsUSD: a cost move smaller than both is noise.
const (
	costMoveRel    = 0.005
	costMoveAbsUSD = 0.01
	// lossEpsilon keeps float noise from pausing a link priced exactly at cost.
	lossEpsilon = 1e-9
)

// unitSellPrice is our selling price for ONE upstream unit, the figure a
// supplier's per-unit cost is compared with: a per-unit product sells one
// variant per BlockUnits units; a package product sells one upstream unit per
// variant, so its cheapest variant is the conservative bound.
func unitSellPrice(p *product.Product) (float64, bool) {
	if p.UnitPricing != nil && p.UnitPricing.BlockUnits > 0 {
		if len(p.Variants) == 0 || p.Variants[0].Price <= 0 {
			return 0, false
		}
		return p.Variants[0].Price / p.UnitPricing.BlockUnits, true
	}
	best := math.Inf(1)
	for _, v := range p.Variants {
		if v.Price > 0 && v.Price < best {
			best = v.Price
		}
	}
	if math.IsInf(best, 1) {
		return 0, false
	}
	return best, true
}

// blockUnits is how many upstream units one sale block holds (alerts quote
// costs per block: "per 10,000", not a 0.0001 unit price).
func blockUnits(p *product.Product) float64 {
	if p.UnitPricing != nil && p.UnitPricing.BlockUnits > 0 {
		return p.UnitPricing.BlockUnits
	}
	return 1
}

// evaluateProduct decides one product's sync changes.
func evaluateProduct(p *product.Product, env syncEnv) evalResult {
	links, activeID := product.EffectiveLinks(p)
	legacy := len(p.SupplierLinks) == 0 // virtual link: nothing per-link is stored
	res := evalResult{ActiveID: activeID, FxMissing: map[int]string{}}
	res.Links = make([]product.SupplierLink, len(links))
	copy(res.Links, links)

	pid := p.ID.Hex()
	pname := strings.TrimSpace(p.Title.En)
	sell, sellOK := unitSellPrice(p)
	block := blockUnits(p)

	// 1. Snapshots + cost moves + loss pauses, link by link.
	for i := range res.Links {
		l := &res.Links[i]
		cat := env.Catalogs[l.Provider]
		if !cat.Known || strings.TrimSpace(l.UpstreamID) == "" {
			continue // unknown: keep the last snapshot, decide nothing
		}
		cp, found := cat.Products[strings.TrimSpace(l.UpstreamID)]
		snap := &product.LinkSync{CheckedAt: env.Now, Found: found}
		if found {
			snap.Available, snap.Name = cp.Available, cp.Name
			snap.Cost, snap.Currency = cp.Price, cp.Currency
			snap.QtyMin, snap.QtyMax = cp.QtyMin, cp.QtyMax
		}
		var rate float64
		rateOK := false
		if found && cp.Price > 0 {
			rate, rateOK = env.FX(l.Provider, cp.Currency)
			if rateOK {
				usd := cp.Price * rate
				snap.CostUSD = &usd
			} else if !l.PriceSyncOff {
				res.FxMissing[l.Provider] = cp.Currency
			}
		}
		prev := l.Sync
		if prev != nil {
			snap.CostChangedAt, snap.PrevCost = prev.CostChangedAt, prev.PrevCost
		}

		// Cost move (compared per sale block, in USD when a rate is known).
		if !l.PriceSyncOff && prev != nil && prev.Found && found && cp.Price > 0 && prev.Cost > 0 &&
			prev.Currency == cp.Currency {
			oldB, newB := prev.Cost*block, cp.Price*block
			thr := costMoveRel * oldB
			if rateOK {
				oldB, newB = oldB*rate, newB*rate
				thr = math.Max(costMoveRel*oldB, costMoveAbsUSD)
			}
			if math.Abs(newB-oldB) > thr {
				at := env.Now
				was := prev.Cost
				snap.CostChangedAt, snap.PrevCost = &at, &was
				pct := (newB - oldB) / oldB * 100
				msg := fmt.Sprintf("%s: %s cost %s moved %s → %s (%+.1f%%). Selling price unchanged — review it.",
					pname, supplierName(env, l.Provider), perBlock(block), money(oldB, rateOK, cp.Currency), money(newB, rateOK, cp.Currency), pct)
				res.Alerts = append(res.Alerts, supplieralert.Alert{
					Kind: supplieralert.KindCostChanged, Key: "cost:" + pid + ":" + l.ID, Message: msg,
					Provider: l.Provider, ProductID: pid, ProductName: pname, LinkID: l.ID,
					Data: map[string]any{"oldCost": prev.Cost, "newCost": cp.Price, "currency": cp.Currency, "upstreamId": l.UpstreamID},
					Args: map[string]string{
						"product": pname, "supplier": supplierName(env, l.Provider), "per": blockArg(block),
						"old": money(oldB, rateOK, cp.Currency), "new": money(newB, rateOK, cp.Currency),
						"pct": fmt.Sprintf("%+.1f", pct),
					},
				})
				res.Changes = append(res.Changes, syncChange{Kind: changeCost, LinkID: l.ID, Provider: l.Provider,
					Detail: map[string]any{"oldCost": prev.Cost, "newCost": cp.Price, "currency": cp.Currency}})
			}
		}
		if !legacy && snapshotChanged(prev, snap) {
			res.Snapshots = append(res.Snapshots, l.ID)
		}
		// Always in memory (the decisions below read it); stored only when
		// it changed, or with a structural write.
		l.Sync = snap

		if legacy || !l.Enabled {
			continue
		}
		// Loss pause / recovery.
		switch {
		case l.PriceSyncOff && l.PausedReason == product.PausedLoss:
			l.PausedReason, l.PausedAt = "", nil
			res.Structural = true
			res.Resolve = append(res.Resolve, "loss:"+pid+":"+l.ID)
			res.Changes = append(res.Changes, syncChange{Kind: changeUnpaused, LinkID: l.ID, Provider: l.Provider})
		case l.PriceSyncOff || !sellOK || snap.CostUSD == nil:
			// can't judge: leave any pause as it is
		case *snap.CostUSD+lossEpsilon >= sell && !l.Paused():
			at := env.Now
			l.PausedReason, l.PausedAt = product.PausedLoss, &at
			res.Structural = true
			res.Changes = append(res.Changes, syncChange{Kind: changePaused, LinkID: l.ID, Provider: l.Provider,
				Detail: map[string]any{"costUsd": *snap.CostUSD, "unitSell": sell}})
		case *snap.CostUSD+lossEpsilon < sell && l.PausedReason == product.PausedLoss:
			l.PausedReason, l.PausedAt = "", nil
			res.Structural = true
			res.Resolve = append(res.Resolve, "loss:"+pid+":"+l.ID)
			res.Alerts = append(res.Alerts, supplieralert.Alert{
				Kind: supplieralert.KindLossRecovered, Key: "recovered:" + pid + ":" + l.ID,
				Message: fmt.Sprintf("%s: %s is profitable again (cost %s, we sell at %s %s) — %s",
					pname, supplierName(env, l.Provider), usd(*snap.CostUSD*block), usd(sell*block), perBlock(block),
					pick(env.Preview, "it would be unpaused. Nothing was changed.",
						"unpaused. The active supplier was not switched back.")),
				Provider: l.Provider, ProductID: pid, ProductName: pname, LinkID: l.ID,
				Args: map[string]string{
					"product": pname, "supplier": supplierName(env, l.Provider),
					"cost": usd(*snap.CostUSD * block), "sell": usd(sell * block), "per": blockArg(block),
				},
			})
			res.Changes = append(res.Changes, syncChange{Kind: changeRecovered, LinkID: l.ID, Provider: l.Provider,
				Detail: map[string]any{"costUsd": *snap.CostUSD, "unitSell": sell}})
		}
	}

	// 2. A paused active link hands over to the next eligible one.
	active := linkByID(res.Links, res.ActiveID)
	promotedTo := ""
	if !legacy && active != nil && active.Paused() {
		for i := range res.Links {
			c := &res.Links[i]
			if c.ID == active.ID || !eligibleForPromotion(*c, env, sell, sellOK) {
				continue
			}
			res.ActiveID, promotedTo = c.ID, c.ID
			res.Structural = true
			res.Changes = append(res.Changes, syncChange{Kind: changePromoted, LinkID: c.ID, Provider: c.Provider,
				Detail: map[string]any{"from": active.ID, "fromProvider": active.Provider}})
			break
		}
	}
	// Loss alerts, worded with what happened to the product.
	for _, ch := range res.Changes {
		if ch.Kind != changePaused {
			continue
		}
		l := linkByID(res.Links, ch.LinkID)
		what := pick(env.Preview, "This supplier would be paused (it is only a backup, so orders are not affected).",
			"It stays as a backup only while paused.")
		args := map[string]string{"outcome": "backup"}
		if l.ID == activeID {
			if promotedTo != "" {
				nl := linkByID(res.Links, promotedTo)
				what = fmt.Sprintf(pick(env.Preview, "Orders would switch to %s (product %s).", "Orders now go to %s (product %s)."),
					supplierName(env, nl.Provider), nl.UpstreamID)
				args = map[string]string{"outcome": "switched", "to": supplierName(env, nl.Provider), "toProduct": nl.UpstreamID}
			} else {
				args["outcome"] = "offsale"
				what = pick(env.Preview,
					"No other supplier can take it, so the product would be taken off sale until the cost drops or you change the price.",
					"No other supplier can take it — the product is hidden until the cost recovers or you change the price.")
			}
		}
		cu, _ := ch.Detail["costUsd"].(float64)
		format := "%s: %s now costs %s %s but we sell at %s — link paused. %s"
		if env.Preview {
			format = "%s: %s now costs %s %s but we sell at %s — that is a loss. %s Nothing was changed yet."
		}
		res.Alerts = append(res.Alerts, supplieralert.Alert{
			Kind: supplieralert.KindLossPaused, Key: "loss:" + pid + ":" + l.ID,
			Message: fmt.Sprintf(format,
				pname, supplierName(env, l.Provider), usd(cu*block), perBlock(block), usd(sell*block), what),
			Provider: l.Provider, ProductID: pid, ProductName: pname, LinkID: l.ID,
			Data: map[string]any{"costUsd": cu, "unitSell": sell, "upstreamId": l.UpstreamID},
			Args: withArgs(args, map[string]string{
				"product": pname, "supplier": supplierName(env, l.Provider),
				"cost": usd(cu * block), "sell": usd(sell * block), "per": blockArg(block),
			}),
		})
	}

	// 3. Visibility.
	active = linkByID(res.Links, res.ActiveID)
	lossBlocked := false
	if active != nil {
		if legacy {
			// Nothing can be paused on an unmigrated product: a loss on its
			// only link takes it off sale directly.
			if sellOK && active.Sync != nil && active.Sync.CostUSD != nil && *active.Sync.CostUSD+lossEpsilon >= sell {
				lossBlocked = true
			}
		} else {
			lossBlocked = active.Paused()
		}
	}
	// Hide for "unavailable" only when EVERY watched link was checked this
	// pass and none has the product: one supplier we couldn't reach may
	// still have it, and the order fallback copes with a refusal.
	// A link whose stock watch is off counts as available (unless paused
	// for a loss): the admin chose not to follow its stock, so it must never
	// be why the product goes off sale. A link that has stock but is paused
	// for a loss makes the reason "loss", not "nobody has it".
	anyKnown, anyUnknown, anyAvail, anyLossOnly := false, false, false, false
	for _, l := range res.Links {
		if !l.Enabled || strings.TrimSpace(l.UpstreamID) == "" {
			continue
		}
		if l.StatusSyncOff {
			if !l.Paused() {
				anyAvail = true
			}
			continue
		}
		if !env.Catalogs[l.Provider].Known {
			anyUnknown = true
			continue
		}
		anyKnown = true
		if l.Sync != nil && l.Sync.Found && l.Sync.Available {
			if l.Paused() {
				anyLossOnly = true
			} else {
				anyAvail = true
			}
		}
	}
	activeAvail := active != nil && env.Catalogs[active.Provider].Known && active.Sync != nil &&
		active.Sync.Found && active.Sync.Available && !active.Paused()

	switch {
	case lossBlocked:
		res.HideReason = "loss"
	case anyKnown && !anyUnknown && !anyAvail && anyLossOnly:
		res.HideReason = "loss"
	case anyKnown && !anyUnknown && !anyAvail:
		res.HideReason = "unavailable"
	}
	switch {
	case res.HideReason != "" && p.Available:
		res.Hide = true
		why := "no linked supplier has it available"
		if res.HideReason == "loss" {
			why = "every supplier that has it costs at least our selling price"
		}
		res.Alerts = append(res.Alerts, supplieralert.Alert{
			Kind: supplieralert.KindProductHidden, Key: "hidden:" + pid,
			Message: fmt.Sprintf(pick(env.Preview,
				"%s would be taken off sale: %s. Nothing was changed — it is still on sale.",
				"%s was taken off sale: %s. It comes back by itself when that changes."), pname, why),
			ProductID: pid, ProductName: pname, Data: map[string]any{"reason": res.HideReason},
			Args: map[string]string{"product": pname, "reason": res.HideReason},
		})
		res.Resolve = append(res.Resolve, "shown:"+pid)
		res.Changes = append(res.Changes, syncChange{Kind: changeHidden, Detail: map[string]any{"reason": res.HideReason}})
	case res.HideReason == "" && !p.Available && p.HiddenBySyncAt != nil && (anyAvail || activeAvail):
		res.Show = true
		res.Alerts = append(res.Alerts, supplieralert.Alert{
			Kind: supplieralert.KindProductShown, Key: "shown:" + pid,
			Message: fmt.Sprintf(pick(env.Preview,
				"%s would go back on sale: a linked supplier has it again.",
				"%s is back on sale: a linked supplier has it again."), pname),
			ProductID: pid, ProductName: pname,
			Args: map[string]string{"product": pname},
		})
		res.Resolve = append(res.Resolve, "hidden:"+pid)
		res.Changes = append(res.Changes, syncChange{Kind: changeShown})
	}
	return res
}

// eligibleForPromotion: an enabled, unpaused link whose supplier's catalog
// loaded this pass, lists the product as available, and sells it below our
// unit price (a cost we can't convert to USD is not eligible: the sync never
// moves orders onto an unknown margin).
func eligibleForPromotion(l product.SupplierLink, env syncEnv, sell float64, sellOK bool) bool {
	if !l.Enabled || l.Paused() || strings.TrimSpace(l.UpstreamID) == "" || !env.Catalogs[l.Provider].Known {
		return false
	}
	s := l.Sync
	if s == nil || !s.Found || !s.Available || s.CostUSD == nil || !sellOK {
		return false
	}
	return *s.CostUSD+lossEpsilon < sell
}

// snapshotChanged reports whether a new snapshot differs from the stored one
// in anything but CheckedAt (a re-check that saw the same thing is not worth
// a write — except the very first one, and one a day so "last checked" in the
// editor never gets misleadingly old).
func snapshotChanged(prev, next *product.LinkSync) bool {
	if prev == nil {
		return true
	}
	if next.CheckedAt.Sub(prev.CheckedAt) > 24*time.Hour {
		return true
	}
	return prev.Found != next.Found || prev.Available != next.Available || prev.Name != next.Name ||
		prev.Cost != next.Cost || prev.Currency != next.Currency ||
		!eqFloatPtr(prev.CostUSD, next.CostUSD) || !eqIntPtr(prev.QtyMin, next.QtyMin) || !eqIntPtr(prev.QtyMax, next.QtyMax) ||
		!eqTimePtr(prev.CostChangedAt, next.CostChangedAt)
}

func linkByID(links []product.SupplierLink, id string) *product.SupplierLink {
	for i := range links {
		if links[i].ID == id {
			return &links[i]
		}
	}
	return nil
}

func supplierName(env syncEnv, id int) string {
	if n := env.Names[id]; n != "" {
		return fmt.Sprintf("%s (#%d)", n, id)
	}
	return fmt.Sprintf("supplier #%d", id)
}

// blockArg is the sale block for alert Args: "1" for a unit, else the
// number of units with separators ("1,000").
func blockArg(block float64) string {
	if block <= 1 {
		return "1"
	}
	return humanizeInt(int64(math.Round(block)))
}

// withArgs returns a with every entry of b added.
func withArgs(a, b map[string]string) map[string]string {
	for k, v := range b {
		a[k] = v
	}
	return a
}

func perBlock(block float64) string {
	if block <= 1 {
		return "per unit"
	}
	return "per " + humanizeInt(int64(math.Round(block)))
}

// pick returns preview when the job runs warning-only, else live.
func pick(preview bool, previewText, live string) string {
	if preview {
		return previewText
	}
	return live
}

func usd(v float64) string { return fmt.Sprintf("$%s", trimMoney(v)) }

func money(v float64, isUSD bool, currency string) string {
	if isUSD {
		return usd(v)
	}
	return trimMoney(v) + " " + currency
}

// trimMoney prints 2 decimals for normal amounts and up to 6 significant
// decimals for sub-cent ones (per-unit costs).
func trimMoney(v float64) string {
	if math.Abs(v) >= 0.01 || v == 0 {
		return fmt.Sprintf("%.2f", v)
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", v), "0"), ".")
}

func eqFloatPtr(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func eqIntPtr(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func eqTimePtr(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}
