package product

import (
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// SortByName is the ListFilter.Sort value for the customer catalog order
// (?sort=name — every app/web listing sends it): products an admin arranged
// in their category (Position > 0) come first in that order, then the rest
// natural A→Z (client feedback 2026-11-12 #68/#104: "sort everything from A
// to Z"; 2026-11-14: "let us set the order of products inside a category").
// The name is kept for wire compatibility with shipped app builds.
const SortByName = "name"

// nameSortCap bounds how many matches the in-memory A→Z path loads. Mongo's
// plain string sort is byte-wise ("Zoo" before "apple", "1000" before "150")
// and collation support varies on Cosmos vCore, so the name sort is done in Go
// over the whole match set, then paged. Catalog subsets are far below this.
const nameSortCap = 5000

// sortKey is the name a product sorts by: its English title, else Arabic,
// with leading punctuation dropped so "(Sigma Plus…" files under S.
func sortKey(p *Product) string {
	s := strings.TrimSpace(p.Title.En)
	if s == "" {
		s = strings.TrimSpace(p.Title.Ar)
	}
	return strings.TrimLeftFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// naturalLess compares two names case-insensitively, treating runs of digits
// as numbers ("JAWAKER 150" < "JAWAKER 1000") and ignoring thousands commas
// inside numbers ("1,000" == "1000").
func naturalLess(a, b string) bool {
	ra, rb := []rune(strings.ToLower(a)), []rune(strings.ToLower(b))
	i, j := 0, 0
	for i < len(ra) && j < len(rb) {
		if unicode.IsDigit(ra[i]) && unicode.IsDigit(rb[j]) {
			na, ni := readNumber(ra, i)
			nb, nj := readNumber(rb, j)
			if na != nb {
				// compare as numbers: shorter (after trimming zeros) is smaller
				if len(na) != len(nb) {
					return len(na) < len(nb)
				}
				return na < nb
			}
			i, j = ni, nj
			continue
		}
		if ra[i] != rb[j] {
			return ra[i] < rb[j]
		}
		i++
		j++
	}
	return len(ra)-i < len(rb)-j
}

// readNumber reads the digit run starting at i (skipping "," between digits)
// and returns it without leading zeros, plus the index just past it.
func readNumber(r []rune, i int) (string, int) {
	var b strings.Builder
	for i < len(r) {
		if unicode.IsDigit(r[i]) {
			b.WriteRune(r[i])
			i++
			continue
		}
		if r[i] == ',' && i+1 < len(r) && unicode.IsDigit(r[i+1]) && b.Len() > 0 {
			i++
			continue
		}
		break
	}
	s := strings.TrimLeft(b.String(), "0")
	if s == "" {
		s = "0"
	}
	return s, i
}

// sortProductsByName orders products in catalog order in place (stable):
// arranged products (Position > 0) first by Position, then the unarranged ones
// A→Z, untitled last. In a listing that spans several nodes (a collection's
// subtree, a root domain, a search) positions from different nodes tie and
// fall back to A→Z, so each node's top picks lead the list.
func sortProductsByName(ps []Product) {
	sort.SliceStable(ps, func(i, j int) bool {
		pi, pj := ps[i].Position, ps[j].Position
		if (pi > 0) != (pj > 0) {
			return pi > 0 // arranged before unarranged
		}
		if pi != pj {
			return pi < pj
		}
		a, b := sortKey(&ps[i]), sortKey(&ps[j])
		if (a == "") != (b == "") {
			return b == "" // untitled products go last
		}
		return naturalLess(a, b)
	})
}

// --- A→Z result cache --------------------------------------------------------
//
// The name sort loads and sorts the whole match set, and the app pages through
// it (infinite scroll) on a public endpoint, so each sorted set is cached
// briefly per filter. Package-level because several MongoRepository instances
// exist per process; every product write through any of them clears it, and
// writes elsewhere (stock counts, supplier sync) age out within the TTL.

const (
	nameCacheTTL = 30 * time.Second
	nameCacheMax = 64
)

type nameCacheEntry struct {
	at    time.Time
	items []Product // sorted; never handed out directly — see cloneProducts
}

var nameCache = struct {
	sync.Mutex
	m map[string]nameCacheEntry
}{m: map[string]nameCacheEntry{}}

// nameCacheGet returns the cached sorted set for key if still fresh.
func nameCacheGet(key string, now time.Time) ([]Product, bool) {
	nameCache.Lock()
	defer nameCache.Unlock()
	e, ok := nameCache.m[key]
	if !ok || now.Sub(e.at) > nameCacheTTL {
		return nil, false
	}
	return e.items, true
}

// nameCachePut stores a sorted set, evicting the oldest entry when full.
func nameCachePut(key string, items []Product, now time.Time) {
	nameCache.Lock()
	defer nameCache.Unlock()
	if _, exists := nameCache.m[key]; !exists && len(nameCache.m) >= nameCacheMax {
		oldestKey, oldest := "", now
		for k, e := range nameCache.m {
			if e.at.Before(oldest) || oldestKey == "" {
				oldestKey, oldest = k, e.at
			}
		}
		delete(nameCache.m, oldestKey)
	}
	nameCache.m[key] = nameCacheEntry{at: now, items: items}
}

// invalidateNameCache drops every cached sorted set (called on product writes).
func invalidateNameCache() {
	nameCache.Lock()
	nameCache.m = map[string]nameCacheEntry{}
	nameCache.Unlock()
}

// cloneProducts copies a page out of the cache. The service enriches listed
// products in place (offer/reseller prices on Variants), so each product is a
// struct copy with its own Variants slice — never the cached backing array.
func cloneProducts(src []Product) []Product {
	out := make([]Product, len(src))
	for i := range src {
		out[i] = src[i]
		out[i].Variants = append([]Variant(nil), src[i].Variants...)
	}
	return out
}
