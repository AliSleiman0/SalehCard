// Package chatbot is the storefront help bot (client feedback 2026-11-12 #74):
// PREDEFINED questions and answers that admins maintain in the console. There
// is no AI and no free-text understanding: customers tap a question, get its
// answer, then its follow-up questions (one level deep). Admin CRUD sits behind
// the "chatbot" RBAC domain and every edit is audited.
package chatbot

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	maxQuestionLen = 200
	maxAnswerLen   = 2000
	maxKeywords    = 20
	maxKeywordLen  = 40
	maxEntries     = 500
)

// Text is a bilingual string (the storefront speaks English and Arabic).
type Text struct {
	En string `bson:"en" json:"en"`
	Ar string `bson:"ar" json:"ar"`
}

func (t Text) trimmed() Text { return Text{En: strings.TrimSpace(t.En), Ar: strings.TrimSpace(t.Ar)} }
func (t Text) empty() bool   { return t.En == "" && t.Ar == "" }

// Entry is one question and its answer. A nil ParentID is a top-level
// question; otherwise it is a follow-up shown after its parent's answer.
type Entry struct {
	ID        bson.ObjectID  `bson:"_id,omitempty"       json:"id"`
	ParentID  *bson.ObjectID `bson:"parentId,omitempty"  json:"parentId,omitempty"`
	Question  Text           `bson:"question"            json:"question"`
	Answer    Text           `bson:"answer"              json:"answer"`
	Keywords  []string       `bson:"keywords,omitempty"  json:"keywords,omitempty"`
	SortOrder int            `bson:"sortOrder"           json:"sortOrder"`
	Enabled   bool           `bson:"enabled"             json:"enabled"`
	CreatedBy string         `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	CreatedAt time.Time      `bson:"createdAt"           json:"createdAt"`
	UpdatedAt time.Time      `bson:"updatedAt"           json:"updatedAt"`
}

// Input is the admin create/update payload (a full replace on update).
type Input struct {
	ParentID  string   `json:"parentId"`
	Question  Text     `json:"question"`
	Answer    Text     `json:"answer"`
	Keywords  []string `json:"keywords"`
	SortOrder int      `json:"sortOrder"`
	Enabled   bool     `json:"enabled"`
}

var errInvalid = errors.New("invalid")

// normalize trims and validates the input, returning a user-facing message on
// failure. It does not check that the parent exists (the handler does).
func (in Input) normalize() (Input, *bson.ObjectID, string, error) {
	out := in
	out.Question = in.Question.trimmed()
	out.Answer = in.Answer.trimmed()
	if out.Question.empty() {
		return out, nil, "a question is required (English or Arabic)", errInvalid
	}
	if out.Answer.empty() {
		return out, nil, "an answer is required (English or Arabic)", errInvalid
	}
	if utf8.RuneCountInString(out.Question.En) > maxQuestionLen || utf8.RuneCountInString(out.Question.Ar) > maxQuestionLen {
		return out, nil, "the question is too long", errInvalid
	}
	if utf8.RuneCountInString(out.Answer.En) > maxAnswerLen || utf8.RuneCountInString(out.Answer.Ar) > maxAnswerLen {
		return out, nil, "the answer is too long", errInvalid
	}
	kws := make([]string, 0, len(in.Keywords))
	seen := map[string]bool{}
	for _, k := range in.Keywords {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" || seen[k] {
			continue
		}
		if utf8.RuneCountInString(k) > maxKeywordLen {
			return out, nil, "a keyword is too long", errInvalid
		}
		seen[k] = true
		kws = append(kws, k)
	}
	if len(kws) > maxKeywords {
		return out, nil, "too many keywords", errInvalid
	}
	out.Keywords = kws

	var parent *bson.ObjectID
	if p := strings.TrimSpace(in.ParentID); p != "" {
		oid, err := bson.ObjectIDFromHex(p)
		if err != nil {
			return out, nil, "invalid parent question", errInvalid
		}
		parent = &oid
	}
	return out, parent, "", nil
}
