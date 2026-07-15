package supplier

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// balanceCacheTTL bounds how stale a supplier balance probe may be. Probes hit a
// billed/rate-limited upstream, so the /suppliers list and dashboard strip share
// this short cache instead of probing on every request.
const balanceCacheTTL = 60 * time.Second

// probeTimeout caps a single balance/catalog probe so one slow supplier never
// stalls the whole list.
const probeTimeout = 8 * time.Second

// Service is the admin supplier surface. It resolves each configured supplier id
// to its provider.Cataloger (through the shared registry) for live probes, and
// owns the supplier_settings persistence plus catalog import/sync over the
// products collection.
type Service struct {
	registry  *provider.Registry
	suppliers []config.SupplierConfig
	settings  *settingsRepo
	products  product.Service
	productc  *mongo.Collection
	orderc    *mongo.Collection

	mu    sync.Mutex
	cache map[int]cachedProbe
}

type cachedProbe struct {
	acc provider.Account
	err error
	at  time.Time
}

// NewService builds the supplier service.
func NewService(db *mongo.Database, registry *provider.Registry, suppliers []config.SupplierConfig, products product.Service) *Service {
	return &Service{
		registry:  registry,
		suppliers: suppliers,
		settings:  newSettingsRepo(db),
		products:  products,
		productc:  db.Collection("products"),
		orderc:    db.Collection("orders"),
		cache:     map[int]cachedProbe{},
	}
}

// supplierByID finds a configured supplier by its provider id.
func (s *Service) supplierByID(id int) (config.SupplierConfig, bool) {
	for _, sc := range s.suppliers {
		if sc.ID == id {
			return sc, true
		}
	}
	return config.SupplierConfig{}, false
}

// cataloger resolves a supplier id to its Cataloger, or (nil,false) when the
// adapter doesn't support catalog/balance probes (stub / non-Cataloger).
func (s *Service) cataloger(id int) (provider.Cataloger, bool) {
	cat, ok := s.registry.Resolve(&id).(provider.Cataloger)
	return cat, ok
}

// probe returns a supplier's account balance, cached for balanceCacheTTL. A
// supplier whose adapter is not a Cataloger returns provider.ErrNotImplemented
// (health "not_probed").
func (s *Service) probe(ctx context.Context, id int) (provider.Account, error) {
	s.mu.Lock()
	if c, ok := s.cache[id]; ok && time.Since(c.at) < balanceCacheTTL {
		s.mu.Unlock()
		return c.acc, c.err
	}
	s.mu.Unlock()

	cat, ok := s.cataloger(id)
	if !ok {
		return provider.Account{}, provider.ErrNotImplemented
	}
	cctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	acc, err := cat.Profile(cctx)

	s.mu.Lock()
	s.cache[id] = cachedProbe{acc: acc, err: err, at: time.Now()}
	s.mu.Unlock()
	return acc, err
}

// List returns every configured supplier with a live (cached) balance/health
// probe and its mapped-product count. Suppliers are probed concurrently.
func (s *Service) List(ctx context.Context) ([]SupplierView, error) {
	settingsMap, err := s.settings.All(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]SupplierView, len(s.suppliers))
	var wg sync.WaitGroup
	for i, sc := range s.suppliers {
		wg.Add(1)
		go func(i int, sc config.SupplierConfig) {
			defer wg.Done()
			set := settingsMap[sc.ID] // zero value when unset
			acc, perr := s.probe(ctx, sc.ID)
			mapped, _ := s.productc.CountDocuments(ctx, bson.D{{Key: "fulfillmentProvider", Value: sc.ID}})

			health := provider.HealthFromError(perr)
			var balance *float64
			balanceText := "—"
			if perr == nil {
				b := acc.Balance
				balance = &b
				balanceText = formatMoney(b, sc.Currency)
				if set.LowBalanceThreshold > 0 && b < set.LowBalanceThreshold {
					health = "low_balance"
				}
			}
			views[i] = SupplierView{
				ID:                  sc.ID,
				Name:                sc.Name,
				Kind:                sc.Kind,
				Currency:            sc.Currency,
				BaseURL:             sc.BaseURL,
				Health:              health,
				Balance:             balance,
				BalanceText:         balanceText,
				MappedProducts:      mapped,
				LowBalanceThreshold: set.LowBalanceThreshold,
				MarkupPercent:       set.MarkupPercent,
			}
		}(i, sc)
	}
	wg.Wait()
	return views, nil
}

// mappedUpstreamIDs returns the set of upstream ids already imported for a
// supplier (products with this provider id and a non-empty upstreamProductId).
func (s *Service) mappedUpstreamIDs(ctx context.Context, id int) (map[string]bool, error) {
	cur, err := s.productc.Find(ctx,
		bson.D{{Key: "fulfillmentProvider", Value: id}},
		options.Find().SetProjection(bson.D{{Key: "upstreamProductId", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := map[string]bool{}
	for cur.Next(ctx) {
		var row struct {
			UpstreamProductID string `bson:"upstreamProductId"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		if row.UpstreamProductID != "" {
			out[row.UpstreamProductID] = true
		}
	}
	return out, cur.Err()
}

// Catalog lists a supplier's upstream products, flagging which are already
// mapped to a SalehCard product.
func (s *Service) Catalog(ctx context.Context, id int) ([]CatalogProductView, error) {
	cat, ok := s.cataloger(id)
	if !ok {
		return nil, provider.ErrNotImplemented
	}
	cctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	products, err := cat.ListProducts(cctx)
	if err != nil {
		return nil, err
	}
	mapped, err := s.mappedUpstreamIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]CatalogProductView, 0, len(products))
	for _, p := range products {
		out = append(out, CatalogProductView{
			UpstreamID:  p.UpstreamID,
			Name:        p.Name,
			Category:    p.Category,
			ParentID:    p.ParentID,
			Price:       p.Price,
			BasePrice:   p.BasePrice,
			Currency:    p.Currency,
			Available:   p.Available,
			ProductType: p.ProductType,
			Params:      p.Params,
			QtyMin:      p.QtyMin,
			QtyMax:      p.QtyMax,
			QtyValues:   p.QtyValues,
			Mapped:      mapped[p.UpstreamID],
		})
	}
	return out, nil
}

// mappedProduct is the projection used by Sync over the products collection.
type mappedProduct struct {
	ID                bson.ObjectID      `bson:"_id"`
	Title             product.I18nString `bson:"title"`
	UpstreamProductID string             `bson:"upstreamProductId"`
	Available         bool               `bson:"available"`
	Variants          []struct {
		Price float64 `bson:"price"`
	} `bson:"variants"`
}

// Sync reconciles the already-mapped products against the live catalog: it
// updates each product's availability to match the upstream (products that
// vanished upstream are marked unavailable) and REPORTS price drift without ever
// rewriting the customer-facing sell price (that stays an explicit admin
// decision — an automated FX-free price rewrite is exactly the footgun we avoid).
func (s *Service) Sync(ctx context.Context, id int) (SyncResult, error) {
	cat, ok := s.cataloger(id)
	if !ok {
		return SyncResult{}, provider.ErrNotImplemented
	}
	cctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	catalog, err := cat.ListProducts(cctx)
	if err != nil {
		return SyncResult{}, err
	}
	byID := make(map[string]provider.CatalogProduct, len(catalog))
	for _, p := range catalog {
		byID[p.UpstreamID] = p
	}

	cur, err := s.productc.Find(ctx, bson.D{{Key: "fulfillmentProvider", Value: id}})
	if err != nil {
		return SyncResult{}, err
	}
	defer cur.Close(ctx)

	var res SyncResult
	for cur.Next(ctx) {
		var mp mappedProduct
		if err := cur.Decode(&mp); err != nil {
			return SyncResult{}, err
		}
		if mp.UpstreamProductID == "" {
			continue
		}
		res.Checked++
		up, found := byID[mp.UpstreamProductID]
		if !found || !up.Available {
			// Withdrawn or unavailable upstream — pull it OFF sale (never delete;
			// orders may reference it). Sync ONLY ever hides (true→false): it never
			// auto-publishes (false→true), so an imported-hidden product stays
			// hidden until an admin reviews it — publishing is an admin decision.
			res.Unavailable++
			if mp.Available {
				_ = s.setAvailable(ctx, mp.ID, false)
				res.Updated++
			}
			if !found {
				continue
			}
		}
		if len(mp.Variants) > 0 {
			old := mp.Variants[0].Price
			if up.BasePrice > 0 && !floatEq(old, up.BasePrice) {
				res.Drift = append(res.Drift, PriceDrift{
					UpstreamID: mp.UpstreamProductID,
					Name:       mp.Title.En,
					OldPrice:   old,
					NewPrice:   up.BasePrice,
				})
			}
		}
	}
	return res, cur.Err()
}

// setAvailable flips a product's availability directly (a targeted, safe write —
// Sync never touches anything else on the product).
func (s *Service) setAvailable(ctx context.Context, id bson.ObjectID, available bool) error {
	_, err := s.productc.UpdateByID(ctx, id, bson.D{{Key: "$set", Value: bson.D{
		{Key: "available", Value: available},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}}})
	return err
}

// Import creates SalehCard products from selected upstream products. Created
// products are api-mode, mapped to this supplier, priced at base×markup, and
// HIDDEN (available=false) pending admin review. Already-mapped items are skipped.
func (s *Service) Import(ctx context.Context, id int, in ImportInput) (ImportResult, error) {
	sc, ok := s.supplierByID(id)
	if !ok {
		return ImportResult{}, fmt.Errorf("unknown supplier %d", id)
	}
	cat, ok := s.cataloger(id)
	if !ok {
		return ImportResult{}, provider.ErrNotImplemented
	}
	cctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	catalog, err := cat.ListProducts(cctx)
	if err != nil {
		return ImportResult{}, err
	}
	byID := make(map[string]provider.CatalogProduct, len(catalog))
	for _, p := range catalog {
		byID[p.UpstreamID] = p
	}
	mapped, err := s.mappedUpstreamIDs(ctx, id)
	if err != nil {
		return ImportResult{}, err
	}
	defMarkup := 0.0
	if set, err := s.settings.Get(ctx, id); err == nil {
		defMarkup = set.MarkupPercent
	}

	var res ImportResult
	for _, item := range in.Items {
		up, found := byID[item.UpstreamID]
		if !found {
			res.Skipped = append(res.Skipped, ImportSkipped{UpstreamID: item.UpstreamID, Reason: "not in catalog"})
			continue
		}
		if mapped[item.UpstreamID] {
			res.Skipped = append(res.Skipped, ImportSkipped{UpstreamID: item.UpstreamID, Reason: "already mapped"})
			continue
		}
		markup := defMarkup
		if item.MarkupPercent != nil {
			markup = *item.MarkupPercent
		}
		input := s.buildProductInput(sc, up, item.CategoryID, markup)
		if _, err := s.products.Create(ctx, input); err != nil {
			res.Skipped = append(res.Skipped, ImportSkipped{UpstreamID: item.UpstreamID, Reason: "create failed: " + err.Error()})
			continue
		}
		mapped[item.UpstreamID] = true // guard against dup ids within one request
		res.Created++
	}
	return res, nil
}

// buildProductInput maps one upstream catalog entry to a CreateProductInput:
// api-mode, mapped to the supplier, one priced variant, input fields from the
// upstream params + qty constraints, created hidden for review.
func (s *Service) buildProductInput(sc config.SupplierConfig, up provider.CatalogProduct, categoryID *string, markup float64) product.CreateProductInput {
	sell := up.BasePrice
	if sell <= 0 {
		sell = up.Price
	}
	if markup != 0 {
		sell = sell * (1 + markup/100)
	}
	title := product.I18nString{En: up.Name, Ar: up.Name, Tr: up.Name}
	return product.CreateProductInput{
		Title:               title,
		Category:            up.Category,
		CategoryID:          categoryID,
		Variants:            []product.Variant{{ID: bson.NewObjectID(), Denomination: up.Name, Price: round2(sell)}},
		FulfillmentType:     fulfillmentTypeFor(sc.Kind, up.ProductType),
		FulfillmentMode:     product.FulfillmentModeAPI,
		FulfillmentProvider: intPtr(sc.ID),
		UpstreamProductID:   up.UpstreamID,
		InputFields:         buildInputFields(up),
		Images:              []string{},
		Stock:               0,
		Available:           false, // hidden — admin reviews before it goes live
	}
}

// RecentOrders returns this supplier's recent api-mode orders (ops trace).
func (s *Service) RecentOrders(ctx context.Context, id int, p pagination.Params) ([]RecentOrderView, int64, error) {
	filter := bson.D{{Key: "items.fulfillmentProvider", Value: id}}
	total, err := s.orderc.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	cur, err := s.orderc.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: -1}}).
			SetSkip(pagination.Skip(p)).
			SetLimit(int64(p.Limit)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var out []RecentOrderView
	for cur.Next(ctx) {
		var row struct {
			ID          bson.ObjectID `bson:"_id"`
			Status      string        `bson:"status"`
			Total       float64       `bson:"total"`
			Currency    string        `bson:"currency"`
			CreatedAt   time.Time     `bson:"createdAt"`
			Fulfillment struct {
				ProviderRef     string     `bson:"providerRef"`
				DeliveredCode   string     `bson:"deliveredCode"`
				SupplierStuckAt *time.Time `bson:"supplierStuckAt"`
			} `bson:"fulfillment"`
			Items []struct {
				PlayerID string `bson:"playerId"`
			} `bson:"items"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, 0, err
		}
		playerID := ""
		if len(row.Items) > 0 {
			playerID = row.Items[0].PlayerID
		}
		out = append(out, RecentOrderView{
			ID:            row.ID.Hex(),
			Status:        row.Status,
			Total:         row.Total,
			Currency:      row.Currency,
			CreatedAt:     row.CreatedAt,
			UpstreamRef:   row.Fulfillment.ProviderRef,
			DeliveredCode: row.Fulfillment.DeliveredCode,
			PlayerID:      playerID,
			StuckAt:       row.Fulfillment.SupplierStuckAt,
		})
	}
	return out, total, cur.Err()
}

// UpdateSettings persists a supplier's operational settings.
func (s *Service) UpdateSettings(ctx context.Context, id int, in SettingsInput, actor string) (Settings, error) {
	return s.settings.Update(ctx, id, in, actor)
}

// --- pure helpers ---

func intPtr(n int) *int { return &n }

func floatEq(a, b float64) bool {
	d := a - b
	return d < 0.0001 && d > -0.0001
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

// fulfillmentTypeFor picks the customer-facing fulfillment type for an imported
// product: panels deliver codes; umanage vouchers are codes, its bundles/credit/
// recharge are account credits.
func fulfillmentTypeFor(kind, productType string) product.FulfillmentType {
	if kind == "telecom" {
		if productType == "voucher" {
			return product.FulfillmentCode
		}
		return product.FulfillmentCredit
	}
	return product.FulfillmentCode
}

// buildInputFields maps upstream params + qty constraints to product InputFields.
func buildInputFields(up provider.CatalogProduct) []product.InputField {
	var fields []product.InputField
	for i, param := range up.Params {
		key := slugify(param)
		if key == "" {
			key = fmt.Sprintf("field%d", i+1)
		}
		fields = append(fields, product.InputField{
			Key:   key,
			Label: product.I18nLabel{En: param},
			Type:  product.InputFieldText,
		})
	}
	switch {
	case len(up.QtyValues) > 0:
		fields = append(fields, product.InputField{
			Key:         "quantity",
			Label:       product.I18nLabel{En: "Quantity"},
			Type:        product.InputFieldSelect,
			Constraints: &product.InputFieldConstraints{Options: up.QtyValues},
		})
	case up.QtyMin != nil || up.QtyMax != nil:
		c := &product.InputFieldConstraints{}
		if up.QtyMin != nil {
			m := float64(*up.QtyMin)
			c.Min = &m
		}
		if up.QtyMax != nil {
			m := float64(*up.QtyMax)
			c.Max = &m
		}
		fields = append(fields, product.InputField{
			Key:         "quantity",
			Label:       product.I18nLabel{En: "Quantity"},
			Type:        product.InputFieldQuantity,
			Constraints: c,
		})
	}
	return fields
}

// slugify lowercases and underscores an ASCII key; returns "" for a param with
// no ASCII alphanumerics (e.g. a purely Arabic prompt) so the caller falls back.
func slugify(s string) string {
	var b strings.Builder
	prevUnderscore := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevUnderscore = false
		case !prevUnderscore && b.Len() > 0:
			b.WriteByte('_')
			prevUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

// formatMoney renders a balance in its supplier currency for the admin card.
func formatMoney(v float64, currency string) string {
	switch currency {
	case "USD":
		return "$" + humanizeFloat(v)
	case "LBP":
		return humanizeInt(int64(v)) + " LBP"
	default:
		return fmt.Sprintf("%.2f %s", v, currency)
	}
}

// humanizeFloat formats a USD amount with thousands separators and 2 decimals.
func humanizeFloat(v float64) string {
	whole := int64(v)
	frac := int64((v-float64(whole))*100 + 0.5)
	if frac >= 100 {
		whole++
		frac -= 100
	}
	return fmt.Sprintf("%s.%02d", humanizeInt(whole), frac)
}

// humanizeInt inserts thousands separators into an integer.
func humanizeInt(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	out := strings.Join(parts, ",")
	if neg {
		out = "-" + out
	}
	return out
}
