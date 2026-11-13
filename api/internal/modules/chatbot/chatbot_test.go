package chatbot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestInputNormalize(t *testing.T) {
	in := Input{
		Question: Text{En: "  How do I top up?  "},
		Answer:   Text{Ar: " من صفحة المحفظة "},
		Keywords: []string{" Wallet ", "wallet", "", "TOPUP"},
	}
	out, parent, msg, err := in.normalize()
	if err != nil {
		t.Fatalf("valid input rejected: %s", msg)
	}
	if out.Question.En != "How do I top up?" || out.Answer.Ar != "من صفحة المحفظة" {
		t.Errorf("not trimmed: %+v", out)
	}
	if strings.Join(out.Keywords, ",") != "wallet,topup" {
		t.Errorf("keywords not lower-cased/deduped: %v", out.Keywords)
	}
	if parent != nil {
		t.Error("no parent expected")
	}

	bad := []Input{
		{Answer: Text{En: "a"}},   // no question
		{Question: Text{En: "q"}}, // no answer
		{Question: Text{En: "q"}, Answer: Text{En: "a"}, ParentID: "nope"},
		{Question: Text{En: strings.Repeat("x", maxQuestionLen+1)}, Answer: Text{En: "a"}},
		{Question: Text{En: "q"}, Answer: Text{En: strings.Repeat("x", maxAnswerLen+1)}},
	}
	for i, b := range bad {
		if _, _, _, err := b.normalize(); !errors.Is(err, errInvalid) {
			t.Errorf("case %d: want errInvalid, got %v", i, err)
		}
	}
}

func TestPublicEntries_HidesOrphanedFollowUps(t *testing.T) {
	top := bson.NewObjectID()
	disabledTop := bson.NewObjectID() // not in the enabled-only list
	all := []Entry{
		{ID: top, Question: Text{En: "Top"}, CreatedBy: "admin@x"},
		{ID: bson.NewObjectID(), ParentID: &top, Question: Text{En: "Follow-up"}},
		{ID: bson.NewObjectID(), ParentID: &disabledTop, Question: Text{En: "Orphan"}},
	}
	got := publicEntries(all)
	if len(got) != 2 {
		t.Fatalf("want top + its follow-up, got %d", len(got))
	}
	for _, e := range got {
		if e.CreatedBy != "" {
			t.Error("createdBy must not reach customers")
		}
		if e.Question.En == "Orphan" {
			t.Error("a follow-up under a disabled parent must be hidden")
		}
	}
}

// fakeRepo is an in-memory Repository for the tree-rule checks.
type fakeRepo struct{ byID map[bson.ObjectID]*Entry }

func (f *fakeRepo) List(context.Context, bool) ([]Entry, error) { return nil, nil }
func (f *fakeRepo) Get(_ context.Context, id bson.ObjectID) (*Entry, error) {
	if e, ok := f.byID[id]; ok {
		return e, nil
	}
	return nil, errNotFound
}
func (f *fakeRepo) Count(context.Context) (int64, error)  { return int64(len(f.byID)), nil }
func (f *fakeRepo) Create(context.Context, *Entry) error  { return nil }
func (f *fakeRepo) Replace(context.Context, *Entry) error { return nil }
func (f *fakeRepo) DeleteWithChildren(context.Context, bson.ObjectID) (*Entry, int64, error) {
	return nil, 0, nil
}
func (f *fakeRepo) HasChildren(_ context.Context, id bson.ObjectID) (bool, error) {
	for _, e := range f.byID {
		if e.ParentID != nil && *e.ParentID == id {
			return true, nil
		}
	}
	return false, nil
}

func TestCheckParent_OneLevelTree(t *testing.T) {
	top := bson.NewObjectID()
	child := bson.NewObjectID()
	lone := bson.NewObjectID()
	repo := &fakeRepo{byID: map[bson.ObjectID]*Entry{
		top:   {ID: top},
		child: {ID: child, ParentID: &top},
		lone:  {ID: lone},
	}}
	h := &adminHandler{repo: repo}
	ctx := context.Background()
	missing := bson.NewObjectID()

	ok := func(self, parent *bson.ObjectID) bool {
		_, err := h.checkParent(ctx, self, parent)
		return err == nil
	}
	if !ok(nil, nil) || !ok(nil, &top) || !ok(&lone, &top) {
		t.Error("top-level and follow-up under a top-level question must be allowed")
	}
	if ok(nil, &child) {
		t.Error("no follow-up under a follow-up (one level deep)")
	}
	if ok(nil, &missing) {
		t.Error("unknown parent must be rejected")
	}
	if ok(&top, &top) {
		t.Error("an entry cannot follow up on itself")
	}
	if ok(&top, &lone) {
		t.Error("a question with follow-ups must stay top-level")
	}
}
