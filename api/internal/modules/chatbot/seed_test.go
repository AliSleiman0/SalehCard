package chatbot

import (
	"context"
	"regexp"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// memRepo is an in-memory Repository for the seed tests.
type memRepo struct{ entries []Entry }

func (m *memRepo) List(context.Context, bool) ([]Entry, error) { return m.entries, nil }
func (m *memRepo) Get(_ context.Context, id bson.ObjectID) (*Entry, error) {
	for i := range m.entries {
		if m.entries[i].ID == id {
			return &m.entries[i], nil
		}
	}
	return nil, errNotFound
}
func (m *memRepo) Count(context.Context) (int64, error) { return int64(len(m.entries)), nil }
func (m *memRepo) Create(_ context.Context, e *Entry) error {
	m.entries = append(m.entries, *e)
	return nil
}
func (m *memRepo) Replace(context.Context, *Entry) error { return nil }
func (m *memRepo) DeleteWithChildren(context.Context, bson.ObjectID) (*Entry, int64, error) {
	return nil, 0, nil
}
func (m *memRepo) HasChildren(context.Context, bson.ObjectID) (bool, error) { return false, nil }

// The help bot is served to the iOS build too, which must never mention these
// (App Review: individual account, recharge-only scope).
var forbidden = regexp.MustCompile(`(?i)crypto|usdt|usdc|tether|bitcoin|binance|whish|money transfer|كريبتو|عملات رقمية|بيتكوين|تحويل أموال`)

func TestStarterContent(t *testing.T) {
	var walk func([]SeedEntry, int)
	walk = func(items []SeedEntry, depth int) {
		for _, s := range items {
			if depth > 0 && len(s.FollowUps) > 0 {
				t.Errorf("%q: follow-ups are one level deep", s.Question.En)
			}
			if s.Question.En == "" || s.Question.Ar == "" || s.Answer.En == "" || s.Answer.Ar == "" {
				t.Errorf("%q: needs English and Arabic question and answer", s.Question.En)
			}
			if _, _, msg, err := (Input{Question: s.Question, Answer: s.Answer, Keywords: s.Keywords}).normalize(); err != nil {
				t.Errorf("%q: %s", s.Question.En, msg)
			}
			for _, txt := range []string{s.Question.En, s.Question.Ar, s.Answer.En, s.Answer.Ar} {
				if forbidden.MatchString(txt) {
					t.Errorf("%q mentions something the iOS build must not: %q", s.Question.En, forbidden.FindString(txt))
				}
			}
			for _, k := range s.Keywords {
				if forbidden.MatchString(k) {
					t.Errorf("%q keyword %q is forbidden on iOS", s.Question.En, k)
				}
			}
			walk(s.FollowUps, depth+1)
		}
	}
	walk(StarterContent, 0)
}

func TestSeedCreatesTreeAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := &memRepo{}
	now := time.Now()
	total := countSeed(StarterContent)

	dry, err := Seed(ctx, repo, StarterContent, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(dry.Created) != total || len(repo.entries) != 0 {
		t.Fatalf("dry run: want %d planned and 0 written, got %d planned and %d written", total, len(dry.Created), len(repo.entries))
	}

	res, err := Seed(ctx, repo, StarterContent, true, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created) != total || len(repo.entries) != total {
		t.Fatalf("apply: want %d created, got %d (stored %d)", total, len(res.Created), len(repo.entries))
	}
	ids := map[bson.ObjectID]bool{}
	for _, e := range repo.entries {
		if e.ID.IsZero() || ids[e.ID] {
			t.Fatalf("entry %q has a missing or duplicate id", e.Question.En)
		}
		ids[e.ID] = true
		if !e.Enabled || e.CreatedBy != "seed" {
			t.Errorf("entry %q: enabled=%v createdBy=%q", e.Question.En, e.Enabled, e.CreatedBy)
		}
	}
	for _, e := range repo.entries {
		if e.ParentID != nil && !ids[*e.ParentID] {
			t.Errorf("follow-up %q points at a missing parent", e.Question.En)
		}
	}

	again, err := Seed(ctx, repo, StarterContent, true, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Created) != 0 || len(again.Skipped) != total || len(repo.entries) != total {
		t.Fatalf("re-run must not duplicate: created %d, skipped %d, stored %d", len(again.Created), len(again.Skipped), len(repo.entries))
	}
}

func TestSeedAddsMissingFollowUpUnderExistingParent(t *testing.T) {
	ctx := context.Background()
	parent := Entry{ID: bson.NewObjectID(), Question: Text{En: "How do I buy a product?"}, Answer: Text{En: "admin's own answer"}, Enabled: true}
	repo := &memRepo{entries: []Entry{parent}}

	res, err := Seed(ctx, repo, StarterContent[:1], true, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) != 1 || len(res.Created) != len(StarterContent[0].FollowUps) {
		t.Fatalf("want parent skipped and follow-ups created, got %+v", res)
	}
	if repo.entries[0].Answer.En != "admin's own answer" {
		t.Error("an existing entry must never be overwritten")
	}
	for _, e := range repo.entries[1:] {
		if e.ParentID == nil || *e.ParentID != parent.ID {
			t.Errorf("follow-up %q should attach to the existing parent", e.Question.En)
		}
	}
}
