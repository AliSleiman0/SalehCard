package chatbot

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// maxKnowledgeChars caps the two knowledge texts together. All of it is sent to
// the model with every question (a stable prompt prefix the provider caches),
// so this is the per-question cost lever: ~24k characters ≈ 8k tokens.
const maxKnowledgeChars = 24000

// knowledgeID is the _id of the single knowledge document.
const knowledgeID = "default"

// Knowledge is what the AI help assistant may answer from: plain text an admin
// writes in one box, in any language (the assistant answers in the customer's
// language). AndroidOnly is a second box the iOS app never gets (App Review:
// the iOS build is telecom-only — no games, gift cards, crypto or transfers).
// Collection chatbot_knowledge, one document.
type Knowledge struct {
	General     string    `bson:"general"             json:"general"`
	AndroidOnly string    `bson:"androidOnly"         json:"androidOnly"`
	UpdatedBy   string    `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
	UpdatedAt   time.Time `bson:"updatedAt"           json:"updatedAt"`
}

// KnowledgeInput is the admin PUT payload.
type KnowledgeInput struct {
	General     string `json:"general"`
	AndroidOnly string `json:"androidOnly"`
}

// normalize trims both texts, normalizes line endings and checks the size cap.
func (in KnowledgeInput) normalize() (KnowledgeInput, string, error) {
	out := KnowledgeInput{General: cleanText(in.General), AndroidOnly: cleanText(in.AndroidOnly)}
	if n := out.chars(); n > maxKnowledgeChars {
		return out, fmt.Sprintf("the knowledge is too long (%d of %d characters) — shorten it", n, maxKnowledgeChars), errInvalid
	}
	return out, "", nil
}

func (in KnowledgeInput) chars() int {
	return utf8.RuneCountInString(in.General) + utf8.RuneCountInString(in.AndroidOnly)
}

func cleanText(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}

// section is one paragraph of the knowledge, the unit an answer must cite.
type section struct {
	Ref  string // "s1", "s2"… — short ids are cheap and easy for the model to copy
	Text string
}

// blankLines separates paragraphs: two line breaks with only whitespace between.
var blankLines = regexp.MustCompile(`\n\s*\n`)

// sectionsFor splits the knowledge the given platform may see into paragraphs
// (separated by blank lines). The admin just writes text; the paragraphs are
// what the hard rule checks citations against. Android-only text is included
// only for "android" and "web" (the website sells everything the Android app
// does) — an unknown platform is treated as iOS (the conservative side for App
// Review).
func sectionsFor(k *Knowledge, platform string) []section {
	if k == nil {
		return nil
	}
	texts := []string{k.General}
	if fullCatalog(platform) {
		texts = append(texts, k.AndroidOnly)
	}
	var out []section
	for _, t := range texts {
		for _, p := range blankLines.Split(cleanText(t), -1) {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, section{Ref: fmt.Sprintf("s%d", len(out)+1), Text: p})
			}
		}
	}
	return out
}

// snippet is how a cited paragraph is shown in the admin conversation log.
func snippet(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 90 {
		return string(r[:89]) + "…"
	}
	return s
}

// KnowledgeRepository persists the single knowledge document.
type KnowledgeRepository interface {
	// Get returns the knowledge, or nil when none was ever saved.
	Get(ctx context.Context) (*Knowledge, error)
	Save(ctx context.Context, k *Knowledge) error
	// InsertIfMissing writes k only when no knowledge document exists yet,
	// reporting whether it did (the idempotent seed path).
	InsertIfMissing(ctx context.Context, k *Knowledge) (bool, error)
}

// MongoKnowledgeRepository is the MongoDB-backed KnowledgeRepository.
type MongoKnowledgeRepository struct{ col *mongo.Collection }

// NewMongoKnowledgeRepository builds the repository over chatbot_knowledge.
func NewMongoKnowledgeRepository(db *mongo.Database) *MongoKnowledgeRepository {
	return &MongoKnowledgeRepository{col: db.Collection("chatbot_knowledge")}
}

func knowledgeFilter() bson.D { return bson.D{{Key: "_id", Value: knowledgeID}} }

// Get implements KnowledgeRepository.
func (r *MongoKnowledgeRepository) Get(ctx context.Context) (*Knowledge, error) {
	var k Knowledge
	if err := r.col.FindOne(ctx, knowledgeFilter()).Decode(&k); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &k, nil
}

// Save implements KnowledgeRepository (a document-atomic upsert).
func (r *MongoKnowledgeRepository) Save(ctx context.Context, k *Knowledge) error {
	_, err := r.col.ReplaceOne(ctx, knowledgeFilter(), k, options.Replace().SetUpsert(true))
	return err
}

// InsertIfMissing implements KnowledgeRepository with $setOnInsert, so an
// existing — possibly admin-edited — document is never touched.
func (r *MongoKnowledgeRepository) InsertIfMissing(ctx context.Context, k *Knowledge) (bool, error) {
	res, err := r.col.UpdateOne(ctx, knowledgeFilter(),
		bson.D{{Key: "$setOnInsert", Value: k}}, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return false, err
	}
	return res.UpsertedCount > 0, nil
}
