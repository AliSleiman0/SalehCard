// Package chatbot is the Help Center's AI assistant: customers type a question
// and Azure OpenAI answers it ONLY from the knowledge text admins write in the
// console (admin → Help bot), never from general knowledge (see enforce). It is
// off unless the admin Settings toggle is on and a real LLM provider is
// configured.
//
// The predefined quick questions (collection chatbot_entries) were retired on
// 2026-11-15: admins no longer edit them, and the current apps never show them.
// GET /api/v1/chatbot still returns the existing ones read-only so app builds
// released before the AI assistant keep their help bot — drop them (and the
// collection) once those builds are gone.
package chatbot

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var errInvalid = errors.New("invalid")

// Text is a bilingual string (legacy quick questions).
type Text struct {
	En string `bson:"en" json:"en"`
	Ar string `bson:"ar" json:"ar"`
}

// Entry is one legacy quick question and its answer (read-only). A nil
// ParentID is a top-level question; otherwise a follow-up.
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
