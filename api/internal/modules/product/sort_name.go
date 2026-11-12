package product

import (
	"sort"
	"strings"
	"unicode"
)

// SortByName is the ListFilter.Sort value for a natural A→Z listing (client
// feedback 2026-11-12 #68/#104: "sort everything from A to Z").
const SortByName = "name"

// nameSortCap bounds how many matches the in-memory A→Z path loads. Mongo's
// plain string sort is byte-wise ("Zoo" before "apple", "1000" before "150")
// and collation support varies on Cosmos vCore, so the name sort is done in Go
// over the whole match set, then paged. Catalog subsets are far below this.
const nameSortCap = 5000

// sortKey is the name a product sorts by: its English title, else Arabic.
func sortKey(p *Product) string {
	if s := strings.TrimSpace(p.Title.En); s != "" {
		return s
	}
	return strings.TrimSpace(p.Title.Ar)
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

// sortProductsByName orders products A→Z in place (stable).
func sortProductsByName(ps []Product) {
	sort.SliceStable(ps, func(i, j int) bool { return naturalLess(sortKey(&ps[i]), sortKey(&ps[j])) })
}
