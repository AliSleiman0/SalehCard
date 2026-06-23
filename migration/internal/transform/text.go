package transform

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/AliSleiman0/salehcard/migration/internal/schema"
)

var (
	tagRe       = regexp.MustCompile(`<[^>]*>`)
	nonSlugRe   = regexp.MustCompile(`[^a-z0-9]+`)
	multiDashRe = regexp.MustCompile(`-+`)
	wsRe        = regexp.MustCompile(`\s+`)
)

// slugify lowercases and reduces a name to ASCII [a-z0-9-]. Non-ASCII (e.g.
// Arabic) characters drop out; when nothing usable remains the caller's fallback
// is used (see slugWithFallback).
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonSlugRe.ReplaceAllString(s, "-")
	s = multiDashRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// slugWithFallback returns a slug for name, or "<prefix>-<id>" when name yields
// no ASCII slug (Arabic-only names).
func slugWithFallback(name, prefix string, id int) string {
	if s := slugify(name); s != "" {
		return s
	}
	return fmt.Sprintf("%s-%d", prefix, id)
}

// stripHTML removes tags and unescapes entities, returning the plain text and
// whether the input actually carried markup (for the markup_stripped flag).
func stripHTML(s string) (text string, hadMarkup bool) {
	if s == "" {
		return "", false
	}
	hadMarkup = tagRe.MatchString(s) || strings.Contains(s, "&")
	out := tagRe.ReplaceAllString(s, " ")
	out = html.UnescapeString(out)
	out = wsRe.ReplaceAllString(out, " ")
	return strings.TrimSpace(out), hadMarkup
}

// ptrOrNil returns a pointer to a trimmed string, or nil when it's empty (so the
// emitted JSON is null rather than "").
func ptrOrNil(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// i18n builds an en/ar/tr value; tr is always null pending translation.
func i18n(en, ar string) schema.I18n {
	return schema.I18n{En: ptrOrNil(en), Ar: ptrOrNil(ar), Tr: nil}
}

// label2 builds an ar/en input-field label.
func label2(en, ar string) schema.I18nLabel {
	return schema.I18nLabel{Ar: ptrOrNil(ar), En: ptrOrNil(en)}
}

// needsTranslation reports whether any required locale (en or tr) is missing.
func needsTranslation(v schema.I18n) bool {
	return v.En == nil || v.Tr == nil
}
