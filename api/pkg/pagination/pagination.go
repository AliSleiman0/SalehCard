package pagination

import (
	"math"
	"net/http"
	"strconv"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
)

// Params holds the parsed page and limit values from a request.
type Params struct {
	Page  int
	Limit int
}

// Meta holds pagination metadata suitable for inclusion in an API response.
type Meta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
	Pages int   `json:"pages"`
}

// ParseParams reads "page" and "limit" query parameters from r.
// page defaults to 1; limit defaults to 20 and is capped at 100.
// Invalid (non-integer) values fall back to their defaults.
func ParseParams(r *http.Request) Params {
	q := r.URL.Query()

	page := defaultPage
	if raw := q.Get("page"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			page = v
		}
	}

	limit := defaultLimit
	if raw := q.Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	return Params{Page: page, Limit: limit}
}

// CalcMeta builds a Meta value for the given Params and total record count.
// Pages is the ceiling of total / limit.
func CalcMeta(p Params, total int64) Meta {
	pages := 0
	if p.Limit > 0 {
		pages = int(math.Ceil(float64(total) / float64(p.Limit)))
	}
	return Meta{
		Page:  p.Page,
		Limit: p.Limit,
		Total: total,
		Pages: pages,
	}
}

// Skip returns the number of records to skip (offset) for the given Params.
func Skip(p Params) int64 {
	return int64((p.Page - 1) * p.Limit)
}
