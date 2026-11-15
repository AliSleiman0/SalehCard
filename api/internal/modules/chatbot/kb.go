package chatbot

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Knowledge-base limits. The whole enabled knowledge base is sent to the model
// with every AI question (it is small, and a stable prompt prefix is cached by
// the provider), so maxKBChars is the per-question cost lever: ~24k characters
// ≈ 8k tokens.
const (
	maxKBTitleLen = 120
	maxKBBodyLen  = 4000
	maxKBDocs     = 40
	maxKBChars    = 24000
	maxKBTags     = 10
	maxKBTagLen   = 30
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,59}$`)

// KBDoc is one knowledge-base article the AI help assistant answers from
// (collection kb_docs). Admins maintain them in the console; only enabled docs
// reach the model. Slug is an optional stable key (used by cmd/kbseed so a
// re-run never duplicates or overwrites an admin-edited doc).
type KBDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"       json:"id"`
	Slug      string        `bson:"slug,omitempty"      json:"slug,omitempty"`
	Title     Text          `bson:"title"               json:"title"`
	Body      Text          `bson:"body"                json:"body"`
	Tags      []string      `bson:"tags,omitempty"      json:"tags,omitempty"`
	SortOrder int           `bson:"sortOrder"           json:"sortOrder"`
	Enabled   bool          `bson:"enabled"             json:"enabled"`
	CreatedBy string        `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	UpdatedBy string        `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
	CreatedAt time.Time     `bson:"createdAt"           json:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"           json:"updatedAt"`
}

// chars is the doc's contribution to the knowledge-base budget.
func (d KBDoc) chars() int {
	return utf8.RuneCountInString(d.Title.En) + utf8.RuneCountInString(d.Title.Ar) +
		utf8.RuneCountInString(d.Body.En) + utf8.RuneCountInString(d.Body.Ar)
}

// KBInput is the admin create/update payload (a full replace on update).
type KBInput struct {
	Slug      string   `json:"slug"`
	Title     Text     `json:"title"`
	Body      Text     `json:"body"`
	Tags      []string `json:"tags"`
	SortOrder int      `json:"sortOrder"`
	Enabled   bool     `json:"enabled"`
}

// normalize trims and validates the input, returning a user-facing message on
// failure. Budget and uniqueness checks need the other docs (the handler does).
func (in KBInput) normalize() (KBInput, string, error) {
	out := in
	out.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	out.Title = in.Title.trimmed()
	out.Body = in.Body.trimmed()
	if out.Slug != "" && !slugRe.MatchString(out.Slug) {
		return out, "the key may only use lowercase letters, digits and dashes", errInvalid
	}
	if out.Title.empty() {
		return out, "a title is required (English or Arabic)", errInvalid
	}
	if out.Body.empty() {
		return out, "the article text is required (English or Arabic)", errInvalid
	}
	if utf8.RuneCountInString(out.Title.En) > maxKBTitleLen || utf8.RuneCountInString(out.Title.Ar) > maxKBTitleLen {
		return out, "the title is too long", errInvalid
	}
	if utf8.RuneCountInString(out.Body.En) > maxKBBodyLen || utf8.RuneCountInString(out.Body.Ar) > maxKBBodyLen {
		return out, "the article is too long (max 4000 characters per language)", errInvalid
	}
	tags := make([]string, 0, len(in.Tags))
	seen := map[string]bool{}
	for _, t := range in.Tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || seen[t] {
			continue
		}
		if utf8.RuneCountInString(t) > maxKBTagLen {
			return out, "a tag is too long", errInvalid
		}
		seen[t] = true
		tags = append(tags, t)
	}
	if len(tags) > maxKBTags {
		return out, "too many tags", errInvalid
	}
	out.Tags = tags
	return out, "", nil
}

// kbUsedChars sums the budget used by the enabled docs, skipping skip (the doc
// being replaced, if any).
func kbUsedChars(docs []KBDoc, skip *bson.ObjectID) int {
	n := 0
	for _, d := range docs {
		if !d.Enabled || (skip != nil && d.ID == *skip) {
			continue
		}
		n += d.chars()
	}
	return n
}
