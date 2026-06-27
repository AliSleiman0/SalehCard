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

	"github.com/AliSleiman0/salehcard/api/internal/modules/code"
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
	// Real (derived from users / wallet ledger):
	ActiveUsers  int     `json:"activeUsers"`  // accounts seen in the last 24h
	WalletTopups float64 `json:"walletTopups"` // top-ups credited today
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
}

// RegisterAdminRoutes mounts the dashboard routes onto r (the /api/admin group).
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	h := &Handler{
		products: db.Collection("products"),
		codes:    code.NewService(db),
		orders:   order.NewMongoRepository(db.Collection("orders")),
		users:    user.NewMongoRepository(db),
		wallet:   wallet.NewMongoRepository(db),
	}
	r.Get("/dashboard/stats", h.GetStats)
	r.Get("/dashboard/low-stock", h.GetLowStock)
	r.Get("/dashboard/revenue-chart", h.GetRevenueChart)
	r.Get("/dashboard/fulfillment-breakdown", h.GetFulfillmentBreakdown)
}

// GetStats handles GET /api/admin/dashboard/stats.
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	totalProducts, err := h.products.CountDocuments(ctx, bson.D{})
	if err != nil {
		response.InternalError(w)
		return
	}

	inv, err := h.codes.Inventory(ctx)
	if err != nil {
		response.InternalError(w)
		return
	}
	var available, delivered, lowStock int
	for _, s := range inv {
		available += s.Available
		delivered += s.Delivered
		if s.Available < s.Threshold {
			lowStock++
		}
	}

	now := time.Now().UTC()
	ordersToday, revenueToday, err := h.orders.DayStats(ctx, now)
	if err != nil {
		response.InternalError(w)
		return
	}
	ordersYday, revenueYday, err := h.orders.DayStats(ctx, now.AddDate(0, 0, -1))
	if err != nil {
		response.InternalError(w)
		return
	}
	pendingTransfers, err := h.orders.CountPendingTransfers(ctx)
	if err != nil {
		response.InternalError(w)
		return
	}

	revenueSpark, ordersSpark, err := h.orders.KpiSparkSeries(ctx)
	if err != nil {
		response.InternalError(w)
		return
	}

	activeUsers, err := h.users.CountActiveSince(ctx, now.Add(-24*time.Hour))
	if err != nil {
		response.InternalError(w)
		return
	}

	walletTopups, err := h.wallet.TopUpsForDay(ctx, now)
	if err != nil {
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
		ActiveUsers:      int(activeUsers),
		WalletTopups:     walletTopups,
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
