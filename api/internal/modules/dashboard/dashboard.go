// Package dashboard serves the admin overview endpoints. Every figure is derived
// from real data: products/codes (inventory), orders (revenue, counts,
// fulfillment, sparklines), users (active-in-24h via lastSeen), and the wallet
// ledger (today's top-ups).
package dashboard

import (
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/sync/errgroup"

	"github.com/AliSleiman0/salehcard/api/internal/modules/code"
	"github.com/AliSleiman0/salehcard/api/internal/modules/kyc"
	"github.com/AliSleiman0/salehcard/api/internal/modules/order"
	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Stats is the payload for GET /api/admin/dashboard/stats.
type Stats struct {
	// Real (derived from products + codes):
	TotalProducts  int64 `json:"totalProducts"`
	LowStockCount  int   `json:"lowStockCount"`
	CodesAvailable int   `json:"codesAvailable"`
	CodesDelivered int   `json:"codesDelivered"`
	// Real (derived from orders):
	RevenueToday     float64   `json:"revenueToday"`
	RevenueDeltaPct  float64   `json:"revenueDeltaPct"`
	OrdersToday      int       `json:"ordersToday"`
	OrdersDeltaPct   float64   `json:"ordersDeltaPct"`
	PendingTransfers int       `json:"pendingTransfers"`
	RevenueSpark     []float64 `json:"revenueSpark"`
	OrdersSpark      []int     `json:"ordersSpark"`
	// Real (all-time order counts per status):
	OrdersFailed    int `json:"ordersFailed"`
	OrdersPending   int `json:"ordersPending"`
	OrdersCompleted int `json:"ordersCompleted"`
	OrdersRefunded  int `json:"ordersRefunded"`
	// Real (derived from users / wallet ledger):
	ActiveUsers  int     `json:"activeUsers"`  // accounts seen in the last 24h
	WalletTopups float64 `json:"walletTopups"` // top-ups credited today
	// Work-queue signals (money is blocked on these):
	PendingTopups int `json:"pendingTopups"` // open top-up requests awaiting a decision
	PendingKyc    int `json:"pendingKyc"`    // KYC submissions awaiting review
}

// ffSlice is one segment of the fulfillment-breakdown donut.
type ffSlice struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value int    `json:"value"`
}

// Handler serves the dashboard endpoints.
type Handler struct {
	products *mongo.Collection
	codes    code.Service
	orders   order.Repository
	users    user.Repository
	wallet   wallet.Repository
	topups   wallet.TopUpStore
	kyc      kyc.Repository
}

// RegisterAdminRoutes mounts the dashboard routes onto r (the /api/admin group).
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	h := &Handler{
		products: db.Collection("products"),
		codes:    code.NewService(db),
		orders:   order.NewMongoRepository(db.Collection("orders")),
		users:    user.NewMongoRepository(db),
		wallet:   wallet.NewMongoRepository(db),
		topups:   wallet.NewTopUpRepo(db),
		kyc:      kyc.NewMongoRepository(db.Collection("kyc_submissions"), db.Collection("users")),
	}
	r.Get("/dashboard/stats", h.GetStats)
	r.Get("/dashboard/low-stock", h.GetLowStock)
	r.Get("/dashboard/revenue-chart", h.GetRevenueChart)
	r.Get("/dashboard/fulfillment-breakdown", h.GetFulfillmentBreakdown)
}

// GetStats handles GET /api/admin/dashboard/stats. The figures come from ~10
// independent reads with no data dependencies between them, so they run
// concurrently via errgroup — the response latency is the slowest single query,
// not their sum (the whole panel used to run serially against a remote DB).
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	g, ctx := errgroup.WithContext(r.Context())

	var (
		totalProducts             int64
		available, delivered      int
		lowStock                  int
		ordersToday, ordersYday   int
		revenueToday, revenueYday float64
		pendingTransfers          int64
		revenueSpark              []float64
		ordersSpark               []int
		activeUsers               int64
		walletTopups              float64
		pendingTopups, pendingKyc int64

		ordersFailed, ordersPending     int64
		ordersCompleted, ordersRefunded int64
	)

	g.Go(func() error {
		var err error
		totalProducts, err = h.products.CountDocuments(ctx, bson.D{})
		return err
	})
	g.Go(func() error {
		inv, err := h.codes.Inventory(ctx)
		if err != nil {
			return err
		}
		for _, s := range inv {
			available += s.Available
			delivered += s.Delivered
			if s.Available < s.Threshold {
				lowStock++
			}
		}
		return nil
	})
	g.Go(func() error {
		var err error
		ordersToday, revenueToday, err = h.orders.DayStats(ctx, now)
		return err
	})
	g.Go(func() error {
		var err error
		ordersYday, revenueYday, err = h.orders.DayStats(ctx, now.AddDate(0, 0, -1))
		return err
	})
	g.Go(func() error {
		var err error
		pendingTransfers, err = h.orders.CountPendingTransfers(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		ordersFailed, err = h.orders.CountByStatus(ctx, order.OrderStatusFailed)
		return err
	})
	g.Go(func() error {
		var err error
		ordersPending, err = h.orders.CountByStatus(ctx, order.OrderStatusPending)
		return err
	})
	g.Go(func() error {
		var err error
		ordersCompleted, err = h.orders.CountByStatus(ctx, order.OrderStatusCompleted)
		return err
	})
	g.Go(func() error {
		var err error
		ordersRefunded, err = h.orders.CountByStatus(ctx, order.OrderStatusRefunded)
		return err
	})
	g.Go(func() error {
		var err error
		revenueSpark, ordersSpark, err = h.orders.KpiSparkSeries(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		activeUsers, err = h.users.CountActiveSince(ctx, now.Add(-24*time.Hour))
		return err
	})
	g.Go(func() error {
		var err error
		walletTopups, err = h.wallet.TopUpsForDay(ctx, now)
		return err
	})
	g.Go(func() error {
		var err error
		pendingTopups, err = h.topups.CountPending(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		pendingKyc, err = h.kyc.CountPending(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		response.InternalError(w)
		return
	}

	stats := Stats{
		TotalProducts:    totalProducts,
		LowStockCount:    lowStock,
		CodesAvailable:   available,
		CodesDelivered:   delivered,
		RevenueToday:     revenueToday,
		RevenueDeltaPct:  deltaPct(revenueToday, revenueYday),
		OrdersToday:      ordersToday,
		OrdersDeltaPct:   deltaPct(float64(ordersToday), float64(ordersYday)),
		PendingTransfers: int(pendingTransfers),
		RevenueSpark:     revenueSpark,
		OrdersSpark:      ordersSpark,
		OrdersFailed:     int(ordersFailed),
		OrdersPending:    int(ordersPending),
		OrdersCompleted:  int(ordersCompleted),
		OrdersRefunded:   int(ordersRefunded),
		ActiveUsers:      int(activeUsers),
		WalletTopups:     walletTopups,
		PendingTopups:    int(pendingTopups),
		PendingKyc:       int(pendingKyc),
	}
	response.OK(w, stats)
}

// deltaPct is the percentage change from prev to cur, rounded to one decimal and
// guarded against a zero baseline.
func deltaPct(cur, prev float64) float64 {
	if prev == 0 {
		if cur == 0 {
			return 0
		}
		return 100
	}
	return math.Round((cur-prev)/prev*1000) / 10
}

// GetLowStock handles GET /api/admin/dashboard/low-stock.
func (h *Handler) GetLowStock(w http.ResponseWriter, r *http.Request) {
	low, err := h.codes.LowStock(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, low)
}

// GetRevenueChart handles GET /api/admin/dashboard/revenue-chart, returning a
// completed-order revenue series for the requested range (daily/weekly/monthly).
func (h *Handler) GetRevenueChart(w http.ResponseWriter, r *http.Request) {
	rng := r.URL.Query().Get("range")
	labels, series, err := h.orders.RevenueSeries(r.Context(), rng)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]any{"range": rng, "labels": labels, "series": series})
}

// GetFulfillmentBreakdown handles GET /api/admin/dashboard/fulfillment-breakdown,
// returning order line-item counts by fulfillment type as donut-ready slices.
func (h *Handler) GetFulfillmentBreakdown(w http.ResponseWriter, r *http.Request) {
	counts, err := h.orders.FulfillmentBreakdown(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	out := []ffSlice{
		{Key: "code", Label: "Code / PIN", Value: counts["code"]},
		{Key: "credit", Label: "Account credit", Value: counts["account_credit"]},
		{Key: "transfer", Label: "Money transfer", Value: counts["transfer"]},
	}
	response.OK(w, out)
}
