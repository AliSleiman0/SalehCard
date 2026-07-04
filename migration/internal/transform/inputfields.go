package transform

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/AliSleiman0/salehcard/migration/internal/legacy"
	"github.com/AliSleiman0/salehcard/migration/internal/schema"
)

// reSensitive matches credential-ish field names/questions (ar + en) that must
// be flagged sensitive (spec §3.2 / §4.1).
var reSensitive = regexp.MustCompile(`(?i)password|wallet|iban|address|كلمة السر|كلمة المرور|محفظة|ايبان|آيبان|رمز المحفظة|عنوان الحساب|عنوان المحفظة`)

// reIDLabel matches an account/player "ID" signal — the English word "id"
// (PLAYER ID, zone id, enter id, …) or the Arabic ايدي / أيدي / معرف.
var reIDLabel = regexp.MustCompile(`(?i)\bid\b|ايدي|أيدي|معرف`)

// reNotIDLabel matches fields that are clearly NOT an ID (email/username/phone/
// password/serial/…). It guards reIDLabel so an "Email ID" field stays an email
// and is never collapsed to the generic ID label.
var reNotIDLabel = regexp.MustCompile(`(?i)email|e-mail|بريد|ايميل|ايمل|username|user name|اسم المستخدم|password|كلمة السر|كلمة المرور|phone|mobile|هاتف|serial|token|iban|wallet|محفظة|barcode|بار كود|صندوق|اسم الشركة|اسم الدولة|الرابط`)

// isIDLabel reports whether a text field's legacy label (ar + en name/question)
// denotes an account/player ID that should be normalized to a clean, bilingual
// {ar:"ID", en:"ID"} label — killing the messy legacy Arabic strings the app
// otherwise shows verbatim. Requires an ID signal and no competing-field signal.
func isIDLabel(en, ar legacy.Require) bool {
	text := en.Name + " " + en.Question + " " + ar.Name + " " + ar.Question
	return reIDLabel.MatchString(text) && !reNotIDLabel.MatchString(text)
}

// legacyTypeToTarget maps a legacy requires[].type to the target inputField type.
func legacyTypeToTarget(t string) string {
	switch t {
	case "amount":
		return "amount"
	case "quantity":
		return "quantity"
	case "selectQty":
		return "select"
	default:
		return "text"
	}
}

// BuildInputFields maps a product's requires[] to target inputFields, matching
// the en and ar variants by their stable require id for bilingual labels. It
// returns the fields and whether any field is sensitive.
func BuildInputFields(enReqs, arReqs []legacy.Require) ([]schema.InputField, bool) {
	arByID := map[int]legacy.Require{}
	for _, r := range arReqs {
		arByID[r.ID] = r
	}

	var fields []schema.InputField
	anySensitive := false
	for i, en := range enReqs {
		ar := arByID[en.ID]

		key := slugify(en.Name)
		if key == "" {
			key = slugify(en.Question)
		}
		if key == "" {
			key = fmt.Sprintf("field_%d", i+1)
		}

		sensitive := reSensitive.MatchString(en.Name + " " + en.Question + " " + ar.Name + " " + ar.Question)
		if sensitive {
			anySensitive = true
		}

		targetType := legacyTypeToTarget(en.Type)
		label := label2(en.Question, firstNonEmpty(ar.Question, en.Question))
		// Normalize account/player ID fields to a clean bilingual "ID" — the
		// legacy source ships inconsistent Arabic labels (ايدي, ادخل الايدي, …)
		// in both locales, which the app renders verbatim even in English.
		if targetType == "text" && isIDLabel(en, ar) {
			label = label2("ID", "ID")
		}

		fields = append(fields, schema.InputField{
			Key:         key,
			Label:       label,
			LegacyName:  en.Name,
			Type:        targetType,
			Constraints: parseConstraints(en),
			Sensitive:   sensitive,
		})
	}
	return fields, anySensitive
}

// parseConstraints reads a requires[].type_value into target constraints:
// numeric {min,max} for amount/quantity, {options} for selectQty, nil for text.
func parseConstraints(r legacy.Require) *schema.InputFieldConstraints {
	if len(r.TypeValue) == 0 || string(r.TypeValue) == "null" {
		return nil
	}
	switch r.Type {
	case "amount", "quantity":
		var rng legacy.Range
		if err := json.Unmarshal(r.TypeValue, &rng); err != nil {
			return nil
		}
		min, max := rng.Min.Value, rng.Max.Value
		return &schema.InputFieldConstraints{Min: &min, Max: &max}
	case "selectQty":
		var raw []json.RawMessage
		if err := json.Unmarshal(r.TypeValue, &raw); err != nil {
			return nil
		}
		opts := make([]string, 0, len(raw))
		for _, item := range raw {
			opts = append(opts, rawToString(item))
		}
		return &schema.InputFieldConstraints{Options: opts}
	default:
		return nil
	}
}

// BuildAmountConstraints extracts product-level amount bounds (from legacy
// `amount`) and flags min>=max corruption on either `amount` or `qty`
// (the known qty{0,0} case). Returns nil constraints for non-variable products.
func BuildAmountConstraints(p *legacy.Product) (*schema.AmountConstraints, []string) {
	var flags []string
	// Corruption = an inverted range (min > max) or the known fully-zero case
	// (qty{0,0}). A degenerate-but-valid fixed range like qty{1,1} ("exactly 1")
	// is NOT corruption — flagging it would bury real issues under false
	// positives (most fixed-package products carry min==max==1).
	corrupt := func(r *legacy.Range) bool {
		if r == nil || !r.Min.Set || !r.Max.Set {
			return false
		}
		return r.Min.Value > r.Max.Value || (r.Min.Value == 0 && r.Max.Value == 0)
	}
	if corrupt(p.Amount) || corrupt(p.Qty) {
		flags = append(flags, schema.FlagAmountCorruption)
	}

	if p.Amount == nil || (!p.Amount.Min.Set && !p.Amount.Max.Set) {
		return nil, flags
	}
	return &schema.AmountConstraints{Min: p.Amount.Min.Value, Max: p.Amount.Max.Value}, flags
}

// rawToString renders a JSON scalar (string or number) as a plain string.
func rawToString(b json.RawMessage) string {
	s := strings.TrimSpace(string(b))
	if len(s) >= 2 && s[0] == '"' {
		var str string
		if err := json.Unmarshal(b, &str); err == nil {
			return str
		}
	}
	return s
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
