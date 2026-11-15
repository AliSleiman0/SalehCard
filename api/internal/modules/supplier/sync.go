package supplier

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/supplieralert"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
)

// syncActor is the audit actor of every change the sync job makes.
const syncActor = "system:supplier-sync"

// firstPassDelay keeps the first scheduled pass off the boot path.
const firstPassDelay = 30 * time.Second

// SyncConfig tunes the job (env SUPPLIER_SYNC_*).
type SyncConfig struct {
	Interval time.Duration
	// Enabled runs the scheduled pass. The "Sync now" button works either way.
	Enabled bool
	// DryRun: decide and alert, but change nothing a customer or an order
	// can see (no pause, no switch, no hide / re-show). The link snapshots —
	// what each supplier charges, last checked — are still recorded, since
	// they are what the cost alerts compare against.
	DryRun bool
}

// SyncJob is the scheduled supplier sync (price watch, status watch,
// pause-on-loss, alerts). One pass at a time: the scheduled tick and the admin
// "Sync now" button share a lock.
type SyncJob struct {
	svc    *Service
	alerts supplieralert.Raiser
	rec    audit.Recorder
	cfg    SyncConfig
	now    func() time.Time

	mu       sync.Mutex // one pass at a time
	lastMu   sync.Mutex
	last     *SyncReport
	inFlight bool
}

// NewSyncJob builds the job. alerts / rec may be nil (tests).
func NewSyncJob(svc *Service, alerts supplieralert.Raiser, rec audit.Recorder, cfg SyncConfig) *SyncJob {
	if cfg.Interval <= 0 {
		cfg.Interval = 10 * time.Minute
	}
	return &SyncJob{svc: svc, alerts: alerts, rec: rec, cfg: cfg, now: func() time.Time { return time.Now().UTC() }}
}

// SupplierSyncStatus is how one supplier's catalog loaded in a pass.
type SupplierSyncStatus struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Status: "ok" | "stale" (only an old copy: treated as unknown) |
	// "unavailable" | "not_supported" (the adapter has no catalog).
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// SyncReport sums up one pass.
type SyncReport struct {
	DryRun      bool                 `json:"dryRun"`
	Supplier    int                  `json:"supplier,omitempty"` // 0 = every supplier
	StartedAt   time.Time            `json:"startedAt"`
	FinishedAt  time.Time            `json:"finishedAt"`
	Suppliers   []SupplierSyncStatus `json:"suppliers"`
	Checked     int                  `json:"checked"`
	Snapshots   int                  `json:"snapshots"`
	CostChanges int                  `json:"costChanges"`
	Paused      int                  `json:"paused"`
	Recovered   int                  `json:"recovered"`
	Promoted    int                  `json:"promoted"`
	Hidden      int                  `json:"hidden"`
	Shown       int                  `json:"shown"`
	Conflicts   int                  `json:"conflicts"`
	Alerts      int                  `json:"alerts"`
	Errors      int                  `json:"errors"`
}

// SyncStatus is GET /suppliers/sync-status.
type SyncStatus struct {
	Enabled         bool        `json:"enabled"`
	DryRun          bool        `json:"dryRun"`
	IntervalSeconds int         `json:"intervalSeconds"`
	Running         bool        `json:"running"`
	Last            *SyncReport `json:"last,omitempty"`
}

// Status reports the job's config and its last pass (this instance).
func (j *SyncJob) Status() SyncStatus {
	j.lastMu.Lock()
	defer j.lastMu.Unlock()
	return SyncStatus{
		Enabled: j.cfg.Enabled, DryRun: j.cfg.DryRun,
		IntervalSeconds: int(j.cfg.Interval / time.Second),
		Running:         j.inFlight, Last: j.last,
	}
}

// Run is the scheduled loop (returns at once when the schedule is disabled).
func (j *SyncJob) Run(ctx context.Context) {
	if !j.cfg.Enabled {
		slog.Info("supplier sync: scheduled sync disabled (SUPPLIER_SYNC_ENABLED=false)")
		return
	}
	slog.Info("supplier sync: started", "interval", j.cfg.Interval, "dryRun", j.cfg.DryRun)
	wait := firstPassDelay
	for {
		select {
		case <-ctx.Done():
			slog.Info("supplier sync: stopped")
			return
		case <-time.After(wait):
		}
		rep, err := j.Pass(ctx, 0)
		if err != nil {
			slog.Error("supplier sync: pass failed", "error", err)
		} else {
			slog.Info("supplier sync: pass done", "checked", rep.Checked, "hidden", rep.Hidden, "shown", rep.Shown,
				"paused", rep.Paused, "promoted", rep.Promoted, "costChanges", rep.CostChanges, "conflicts", rep.Conflicts, "dryRun", rep.DryRun)
		}
		wait = j.cfg.Interval
	}
}

// ErrSupplierCatalog is returned by a one-supplier pass whose supplier's
// catalog didn't load fresh (acting on an old copy could hide products wrongly).
var ErrSupplierCatalog = errors.New("supplier catalog did not load")

// Pass runs one sync pass over every product linked to supplier only (0 =
// every supplier). The products' other suppliers are read from the shared
// cache; only fresh catalogs count as known.
func (j *SyncJob) Pass(ctx context.Context, only int) (SyncReport, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.setInFlight(true)
	defer j.setInFlight(false)

	rep := SyncReport{DryRun: j.cfg.DryRun, Supplier: only, StartedAt: j.now()}
	env, err := j.loadEnv(ctx, only, &rep)
	if err != nil {
		return rep, err
	}

	filter := bson.D{
		{Key: "deletedAt", Value: nil},
		{Key: "fulfillmentMode", Value: string(product.FulfillmentModeAPI)},
	}
	if only != 0 {
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "supplierLinks.provider", Value: only}},
			bson.D{{Key: "fulfillmentProvider", Value: only}},
		}})
	} else {
		// Plain $in on the configured supplier ids (the same shape as the
		// supplier cards' mapped count). Prod's Cosmos vCore matched nothing
		// for the earlier "supplierLinks.0 $exists / fulfillmentProvider $ne
		// null" form, so every pass checked 0 products.
		ids := bson.A{}
		for _, sc := range j.svc.suppliers {
			ids = append(ids, sc.ID)
		}
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "supplierLinks.provider", Value: bson.D{{Key: "$in", Value: ids}}}},
			bson.D{{Key: "fulfillmentProvider", Value: bson.D{{Key: "$in", Value: ids}}}},
		}})
	}
	cur, err := j.svc.productc.Find(ctx, filter)
	if err != nil {
		return rep, err
	}
	defer cur.Close(ctx)

	fxMissing := map[int]map[string]int{} // provider -> currency -> links
	visibilityChanged := false
	for cur.Next(ctx) {
		var p product.Product
		if err := cur.Decode(&p); err != nil {
			rep.Errors++
			slog.Error("supplier sync: decode product", "error", err)
			continue
		}
		rep.Checked++
		res := evaluateProduct(&p, env)
		for prov, ccy := range res.FxMissing {
			if fxMissing[prov] == nil {
				fxMissing[prov] = map[string]int{}
			}
			fxMissing[prov][ccy]++
		}
		changed, err := j.apply(ctx, &p, res, &rep)
		if err != nil {
			rep.Errors++
			slog.Error("supplier sync: apply", "product", p.ID.Hex(), "error", err)
			continue
		}
		visibilityChanged = visibilityChanged || changed
	}
	if err := cur.Err(); err != nil {
		return rep, err
	}
	if visibilityChanged {
		product.InvalidateListingCache()
	}
	j.fxAlerts(ctx, env, only, fxMissing, &rep)

	rep.FinishedAt = j.now()
	j.lastMu.Lock()
	r := rep
	j.last = &r
	j.lastMu.Unlock()
	return rep, nil
}

func (j *SyncJob) setInFlight(v bool) {
	j.lastMu.Lock()
	j.inFlight = v
	j.lastMu.Unlock()
}

// loadEnv loads every configured supplier's catalog (refreshing only when
// only is 0 or that supplier) and the exchange rates.
func (j *SyncJob) loadEnv(ctx context.Context, only int, rep *SyncReport) (syncEnv, error) {
	env := syncEnv{Catalogs: map[int]catalogView{}, Names: map[int]string{}, Now: j.now(), Preview: j.cfg.DryRun}
	settings, err := j.svc.settings.All(ctx)
	if err != nil {
		return env, err
	}
	ccy := map[int]string{}
	for _, sc := range j.svc.suppliers {
		env.Names[sc.ID] = sc.Name
		ccy[sc.ID] = sc.Currency
		var snap provider.CatalogSnapshot
		if only == 0 || only == sc.ID {
			snap, err = j.svc.probes.Refresh(ctx, sc.ID)
		} else {
			snap, err = j.svc.probes.Catalog(ctx, sc.ID)
		}
		st := SupplierSyncStatus{ID: sc.ID, Name: sc.Name, Status: "ok"}
		switch {
		case errors.Is(err, provider.ErrNotImplemented):
			st.Status = "not_supported"
		case err != nil:
			st.Status, st.Error = "unavailable", err.Error()
		case snap.Stale:
			st.Status = "stale"
		default:
			byID := make(map[string]provider.CatalogProduct, len(snap.Products))
			for _, cp := range snap.Products {
				byID[cp.UpstreamID] = cp
			}
			env.Catalogs[sc.ID] = catalogView{Known: true, Products: byID}
		}
		rep.Suppliers = append(rep.Suppliers, st)
		if only == sc.ID && st.Status != "ok" {
			return env, fmt.Errorf("%w: supplier #%d is %s", ErrSupplierCatalog, sc.ID, st.Status)
		}
	}
	env.FX = func(prov int, currency string) (float64, bool) {
		c := strings.ToUpper(strings.TrimSpace(currency))
		if c == "" {
			c = strings.ToUpper(ccy[prov])
		}
		if c == "" || c == "USD" {
			return 1, true
		}
		if r := settings[prov].FxToUSD; r > 0 {
			return r, true
		}
		return 0, false
	}
	return env, nil
}

// revFilter matches a product still at links rev (an unlinked product has
// no rev field: 0 matches both).
func revFilter(rev int) bson.E {
	if rev == 0 {
		return bson.E{Key: "supplierLinksRev", Value: bson.D{{Key: "$in", Value: bson.A{0, nil}}}}
	}
	return bson.E{Key: "supplierLinksRev", Value: rev}
}

// apply writes one product's decision. Every write is guarded on the links
// rev (and the active link) it was decided from, so an admin save in between
// wins and the product is simply looked at again next pass. It reports
// whether the product's visibility changed.
func (j *SyncJob) apply(ctx context.Context, p *product.Product, res evalResult, rep *SyncReport) (bool, error) {
	pc := j.svc.productc
	rev := p.SupplierLinksRev
	dry := j.cfg.DryRun
	base := bson.D{
		{Key: "_id", Value: p.ID},
		{Key: "deletedAt", Value: nil},
		revFilter(rev),
		{Key: "activeSupplierLinkId", Value: p.ActiveSupplierLinkID},
	}
	now := j.now()

	// 1. The links table: whole, under the rev guard. A structural change
	// (pause / unpause / switch) bumps the rev; a snapshot-only refresh
	// doesn't, so an editor open in the admin isn't made stale by it.
	structural := res.Structural && !dry
	if structural || len(res.Snapshots) > 0 {
		set := bson.D{{Key: "supplierLinks", Value: res.Links}}
		if structural {
			active := linkByID(res.Links, res.ActiveID)
			if active == nil {
				return false, fmt.Errorf("active link %q missing", res.ActiveID)
			}
			rev++
			set = append(set,
				bson.E{Key: "activeSupplierLinkId", Value: res.ActiveID},
				bson.E{Key: "supplierLinksRev", Value: rev},
				// The mirror always follows the active link, same write.
				bson.E{Key: "fulfillmentProvider", Value: active.Provider},
				bson.E{Key: "upstreamProductId", Value: active.UpstreamID},
				bson.E{Key: "updatedAt", Value: now},
			)
		} else if dry && res.Structural {
			// A dry run stores snapshots but never the decided pauses.
			set = bson.D{{Key: "supplierLinks", Value: withoutPauseChanges(p, res.Links)}}
		}
		ur, err := pc.UpdateOne(ctx, base, bson.D{{Key: "$set", Value: set}})
		if err != nil {
			return false, err
		}
		if ur.MatchedCount == 0 {
			rep.Conflicts++
			return false, nil // an admin changed it meanwhile: next pass
		}
		rep.Snapshots += len(res.Snapshots)
	}

	// 2. Visibility, guarded on the same (possibly bumped) rev and on the
	// state it was decided from.
	vis := false
	var visErr error
	if !dry && (res.Hide || res.Show) {
		f := bson.D{{Key: "_id", Value: p.ID}, {Key: "deletedAt", Value: nil}, revFilter(rev)}
		var set bson.D
		if res.Hide {
			f = append(f, bson.E{Key: "available", Value: true})
			set = bson.D{{Key: "available", Value: false}, {Key: "hiddenBySyncAt", Value: now}, {Key: "updatedAt", Value: now}}
		} else {
			f = append(f, bson.E{Key: "available", Value: false}, bson.E{Key: "hiddenBySyncAt", Value: bson.D{{Key: "$ne", Value: nil}}})
			set = bson.D{{Key: "available", Value: true}, {Key: "hiddenBySyncAt", Value: nil}, {Key: "updatedAt", Value: now}}
		}
		ur, err := pc.UpdateOne(ctx, f, bson.D{{Key: "$set", Value: set}})
		switch {
		case err != nil:
			visErr = err
		case ur.MatchedCount == 0:
			rep.Conflicts++
		default:
			vis = true
		}
	}

	// 3. Report, alerts, audit — only for what actually happened (a dry run
	// reports what it would have done).
	var done []syncChange
	for _, ch := range res.Changes {
		switch ch.Kind {
		case changeHidden, changeShown:
			if !dry && !vis {
				continue
			}
		}
		done = append(done, ch)
		switch ch.Kind {
		case changeCost:
			rep.CostChanges++
		case changePaused:
			rep.Paused++
		case changeRecovered:
			rep.Recovered++
		case changePromoted:
			rep.Promoted++
		case changeHidden:
			rep.Hidden++
		case changeShown:
			rep.Shown++
		}
	}
	for _, a := range res.Alerts {
		if (a.Kind == supplieralert.KindProductHidden || a.Kind == supplieralert.KindProductShown) && !dry && !vis {
			continue
		}
		j.raise(ctx, a, rep)
	}
	if j.alerts != nil {
		for _, k := range res.Resolve {
			if strings.HasPrefix(k, "hidden:") || strings.HasPrefix(k, "shown:") {
				if !dry && !vis {
					continue
				}
			}
			if err := j.alerts.Resolve(ctx, k, syncActor); err != nil {
				slog.Warn("supplier sync: resolve alert", "key", k, "error", err)
			}
		}
	}
	if !dry && j.rec != nil {
		var audited []syncChange
		for _, ch := range done {
			if ch.Kind != changeSnapshot {
				audited = append(audited, ch)
			}
		}
		if len(audited) > 0 {
			j.rec.Record(ctx, audit.Entry{
				ActorID: syncActor, ActorEmail: syncActor,
				Action: audit.ActionSupplierSyncChange, TargetType: "product", TargetID: p.ID.Hex(),
				Summary: map[string]any{"product": strings.TrimSpace(p.Title.En), "changes": audited},
			})
		}
	}
	return vis, visErr
}

// withoutPauseChanges keeps the stored pause state of every link (dry run):
// only the snapshots of the new table are taken.
func withoutPauseChanges(p *product.Product, next []product.SupplierLink) []product.SupplierLink {
	out := make([]product.SupplierLink, len(next))
	copy(out, next)
	for i := range out {
		for _, old := range p.SupplierLinks {
			if old.ID == out[i].ID {
				out[i].PausedReason, out[i].PausedAt = old.PausedReason, old.PausedAt
			}
		}
	}
	return out
}

func (j *SyncJob) raise(ctx context.Context, a supplieralert.Alert, rep *SyncReport) {
	if j.alerts == nil {
		return
	}
	// Warning-only mode: the message is already worded as "would …"
	// (syncEnv.Preview); the flag drives the "Warning only" badge.
	a.DryRun = j.cfg.DryRun
	if err := j.alerts.Raise(ctx, a); err != nil {
		slog.Warn("supplier sync: raise alert", "key", a.Key, "error", err)
		return
	}
	rep.Alerts++
}

// fxAlerts raises one fx_missing alert per supplier whose prices couldn't be
// converted to USD, and closes it for a checked supplier that no longer needs
// one.
func (j *SyncJob) fxAlerts(ctx context.Context, env syncEnv, only int, missing map[int]map[string]int, rep *SyncReport) {
	if j.alerts == nil {
		return
	}
	for _, sc := range j.svc.suppliers {
		if only != 0 && sc.ID != only {
			continue
		}
		key := fmt.Sprintf("fx:%d", sc.ID)
		m := missing[sc.ID]
		if len(m) == 0 {
			if env.Catalogs[sc.ID].Known {
				_ = j.alerts.Resolve(ctx, key, syncActor)
			}
			continue
		}
		n := 0
		var ccys []string
		for c, k := range m {
			n += k
			ccys = append(ccys, c)
		}
		j.raise(ctx, supplieralert.Alert{
			Kind: supplieralert.KindFxMissing, Key: key, Provider: sc.ID,
			Message: fmt.Sprintf("%s prices are in %s but no USD exchange rate is set, so the loss check is skipped for %d linked product(s). Set the rate in Suppliers → Settings.",
				supplierName(env, sc.ID), strings.Join(ccys, "/"), n),
		}, rep)
	}
}
