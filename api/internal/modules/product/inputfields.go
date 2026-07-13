package product

import (
	"fmt"
	"strings"
)

// sanitizeInputFields normalizes legacy-tolerable input-field shapes and
// validates the rest before a create/update persists them. Normalization is
// repair-only (a save must never be blocked by pre-existing imported data):
// keys are trimmed, an empty type defaults to text, constraints that don't
// belong to the field's type are stripped, the legacy-corrupt {min:0,max:0}
// bound is removed, and select options are trimmed/deduped. Rejections are
// reserved for unambiguous admin mistakes fixable in the product editor
// (empty/duplicate keys, unknown types, negative bounds, min>max).
// A nil/empty input is valid — the product collects nothing at checkout.
// The caller's slice is never mutated; a normalized copy is returned.
func sanitizeInputFields(fields []InputField) ([]InputField, error) {
	if len(fields) == 0 {
		return fields, nil
	}
	out := make([]InputField, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for i, f := range fields {
		f.Key = strings.TrimSpace(f.Key)
		if f.Key == "" {
			return nil, badRequest(fmt.Sprintf("input field #%d needs a key", i+1))
		}
		if _, dup := seen[f.Key]; dup {
			return nil, badRequest(fmt.Sprintf("input field keys must be unique: %q appears more than once", f.Key))
		}
		seen[f.Key] = struct{}{}

		if f.Type == "" {
			// Pre-type legacy documents round-trip through the editor typeless.
			f.Type = InputFieldText
		}
		switch f.Type {
		case InputFieldText:
			f.Constraints = nil
		case InputFieldAmount, InputFieldQuantity:
			c, err := sanitizeNumericConstraints(f.Key, f.Constraints)
			if err != nil {
				return nil, err
			}
			f.Constraints = c
		case InputFieldSelect:
			f.Constraints = sanitizeSelectConstraints(f.Constraints)
		default:
			return nil, badRequest(fmt.Sprintf("input field %q has unknown type %q", f.Key, f.Type))
		}
		out[i] = f
	}
	return out, nil
}

// sanitizeNumericConstraints validates the min/max bounds on an amount or
// quantity field. The legacy-corrupt {min:0,max:0} import shape is silently
// stripped (it means "unbounded", and order-time enforcement skips it anyway);
// one-sided bounds are legal. Options never belong on a numeric field.
func sanitizeNumericConstraints(key string, c *InputFieldConstraints) (*InputFieldConstraints, error) {
	if c == nil {
		return nil, nil
	}
	lo, hi := c.Min, c.Max
	if lo != nil && hi != nil && *lo == 0 && *hi == 0 {
		lo, hi = nil, nil
	}
	if (lo != nil && *lo < 0) || (hi != nil && *hi < 0) {
		return nil, badRequest(fmt.Sprintf("input field %q: min/max cannot be negative", key))
	}
	if lo != nil && hi != nil && *lo > *hi {
		return nil, badRequest(fmt.Sprintf("input field %q: min cannot exceed max", key))
	}
	if lo == nil && hi == nil {
		return nil, nil
	}
	return &InputFieldConstraints{Min: lo, Max: hi}, nil
}

// sanitizeSelectConstraints keeps only the options list on a select field:
// each option is trimmed, blanks are dropped, and duplicates keep their first
// occurrence. An empty result is tolerated (legacy imports carry optionless
// selects; order-time enforcement treats them as free-entry) — the admin
// editor blocks creating new ones client-side.
func sanitizeSelectConstraints(c *InputFieldConstraints) *InputFieldConstraints {
	if c == nil {
		return nil
	}
	opts := make([]string, 0, len(c.Options))
	seen := make(map[string]struct{}, len(c.Options))
	for _, o := range c.Options {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if _, dup := seen[o]; dup {
			continue
		}
		seen[o] = struct{}{}
		opts = append(opts, o)
	}
	if len(opts) == 0 {
		return nil
	}
	return &InputFieldConstraints{Options: opts}
}
