// Package dashboard serves the admin overview endpoints. Product/code-derived
// figures are real; revenue/orders/users figures are mocked until the orders and
// wallet modules are wired (flagged with TODOs).
package dashboard

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/code"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Stats is the payload for GET /api/admin/dashboard/stats.
type Stats struct {
	// Real (derived from products + codes):
	TotalProducts  int64 `json:"totalProducts"`
	LowStockCount  int   `json:"lowStockCount"`
	CodesAvailable int   `json:"codesAvailable"`
	CodesDelivered int   `json:"codesDelivered"`
	// Mocked until orders/wallet modules land:
	RevenueToday     float64 `json:"revenueToday"`
	RevenueDeltaPct  float64 `json:"revenueDeltaPct"`
	OrdersToday      int     `json:"ordersToday"`
	OrdersDeltaPct   float64 `json:"ordersDeltaPct"`
	ActiveUsers      int     `json:"activeUsers"`
	WalletTopups     float64 `json:"walletTopups"`
	PendingTransfers int     `json:"pendingTransfers"`
}

// Handler serves the dashboard endpoints.
type Handler struct {
	products *mongo.Collection
	codes    code.Service
}

// RegisterAdminRoutes mounts the dashboard routes onto r (the /api/admin group).
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	h := &Handler{
		products: db.Collection("products"),
		codes:    code.NewService(db),
	}
	r.Get("/dashboard/stats", h.GetStats)
	r.Get("/dashboard/low-stock", h.GetLowStock)
	r.Get("/dashboard/revenue-chart", h.GetRevenueChart)
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

	stats := Stats{
		TotalProducts:  totalProducts,
		LowStockCount:  lowStock,
		CodesAvailable: available,
		CodesDelivered: delivered,
		// TODO: replace with real figures once the orders + wallet modules are wired.
		RevenueToday:     12148,
		RevenueDeltaPct:  14.2,
		OrdersToday:      842,
		OrdersDeltaPct:   8.1,
		ActiveUsers:      3610,
		WalletTopups:     48920,
		PendingTransfers: 4,
	}
	response.OK(w, stats)
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

// GetRevenueChart handles GET /api/admin/dashboard/revenue-chart.
// TODO: derive from real order revenue; currently returns a mock series.
func (h *Handler) GetRevenueChart(w http.ResponseWriter, r *http.Request) {
	rng := r.URL.Query().Get("range")
	series := map[string][]float64{
		"daily":   {4.2, 5.1, 4.8, 6.3, 7.1, 6.6, 8.2, 7.4, 9.1, 8.7, 10.2, 9.6, 11.4, 12.1},
		"weekly":  {28, 32, 30, 38, 41, 44, 47, 52, 49, 58, 61, 67},
		"monthly": {98, 112, 121, 134, 128, 156, 162, 178, 171, 198, 214, 240},
	}
	data, ok := series[rng]
	if !ok {
		data = series["daily"]
	}
	response.OK(w, map[string]any{"range": rng, "series": data})
}
