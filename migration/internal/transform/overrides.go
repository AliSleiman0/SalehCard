package transform

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"strconv"

	"github.com/AliSleiman0/salehcard/migration/internal/schema"
)

// Override is a hand-authored correction for one product, keyed by legacyId in
// overrides.json. It forces fulfillment/pricing fields and can clear review
// flags. Applied last so human decisions survive re-runs.
type Override struct {
	Fulfillment *struct {
		Type     *string `json:"type"`
		Mode     *string `json:"mode"`
		Provider *int    `json:"provider"`
	} `json:"fulfillment"`
	Pricing *struct {
		Mode *string `json:"mode"`
	} `json:"pricing"`
	ClearFlags []string `json:"clearFlags"` // flag names to remove, or ["*"] to clear all
}

// Overrides maps a legacyId (as string) to its Override.
type Overrides map[string]Override

// LoadOverrides reads overrides.json. A missing file yields an empty set (not an
// error) so the importer runs cleanly before any corrections exist.
func LoadOverrides(path string) (Overrides, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Overrides{}, nil
		}
		return nil, err
	}
	var ov Overrides
	if err := json.Unmarshal(b, &ov); err != nil {
		return nil, err
	}
	return ov, nil
}

// ApplyOverrides mutates products in place per the overrides, returning the
// number of products touched.
func ApplyOverrides(products []schema.Product, ov Overrides) int {
	if len(ov) == 0 {
		return 0
	}
	touched := 0
	for i := range products {
		o, ok := ov[strconv.Itoa(products[i].LegacyID)]
		if !ok {
			continue
		}
		applyOne(&products[i], o)
		touched++
	}
	return touched
}

func applyOne(p *schema.Product, o Override) {
	if o.Fulfillment != nil {
		if o.Fulfillment.Type != nil {
			p.Fulfillment.Type = *o.Fulfillment.Type
		}
		if o.Fulfillment.Mode != nil {
			p.Fulfillment.Mode = *o.Fulfillment.Mode
		}
		if o.Fulfillment.Provider != nil {
			p.Fulfillment.Provider = o.Fulfillment.Provider
		}
	}
	if o.Pricing != nil && o.Pricing.Mode != nil {
		p.Pricing.Mode = *o.Pricing.Mode
	}
	if len(o.ClearFlags) > 0 {
		p.Flags = clearFlags(p.Flags, o.ClearFlags)
	}
}

// clearFlags removes the named flags (or all when ["*"]).
func clearFlags(flags, clear []string) []string {
	for _, c := range clear {
		if c == "*" {
			return nil
		}
	}
	remove := map[string]bool{}
	for _, c := range clear {
		remove[c] = true
	}
	out := flags[:0:0]
	for _, f := range flags {
		if !remove[f] {
			out = append(out, f)
		}
	}
	return out
}
