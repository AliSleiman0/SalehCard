package chatbot

import (
	"context"
	"regexp"
	"slices"
	"testing"
	"time"
)

// TestStarterKB keeps the starter knowledge base valid, within budget and
// iOS-safe (the same forbidden list as the starter Q&A).
// notOnIOS: what the telecom-only iOS build doesn't sell. Only docs tagged
// android-only may mention it.
var notOnIOS = regexp.MustCompile(`(?i)\bgames?\b|player|gift ?cards?|voucher|pubg|free fire|لعب|اللاعب|بطاقات الهدايا|القسائم`)

func TestStarterKB(t *testing.T) {
	seen := map[string]bool{}
	androidOnly := 0
	var docs []KBDoc
	for _, s := range StarterKB {
		if seen[s.Slug] {
			t.Errorf("duplicate slug %q", s.Slug)
		}
		seen[s.Slug] = true
		in, msg, err := KBInput{Slug: s.Slug, Title: s.Title, Body: s.Body, Tags: s.Tags, Enabled: true}.normalize()
		if err != nil {
			t.Errorf("%s: %s", s.Slug, msg)
		}
		if in.Slug == "" || s.Title.En == "" || s.Title.Ar == "" || s.Body.En == "" || s.Body.Ar == "" {
			t.Errorf("%s: every starter doc needs a slug and both languages", s.Slug)
		}
		docs = append(docs, KBDoc{Title: s.Title, Body: s.Body, Enabled: true})
		if slices.Contains(s.Tags, TagAndroidOnly) {
			androidOnly++
			continue // never shown on iOS
		}
		for _, txt := range []string{s.Title.En, s.Title.Ar, s.Body.En, s.Body.Ar} {
			if forbidden.MatchString(txt) {
				t.Errorf("%s mentions something the iOS build must not: %q", s.Slug, forbidden.FindString(txt))
			}
			if notOnIOS.MatchString(txt) {
				t.Errorf("%s reaches the telecom-only iOS build but mentions %q: tag it %q or reword it",
					s.Slug, notOnIOS.FindString(txt), TagAndroidOnly)
			}
		}
	}
	if androidOnly == 0 || androidOnly == len(StarterKB) {
		t.Errorf("expected some (not all) starter docs to be android-only, got %d", androidOnly)
	}
	if used := kbUsedChars(docs, nil); used > maxKBChars*3/4 {
		t.Errorf("starter KB uses %d of %d chars — leave admins room to add docs", used, maxKBChars)
	}
}

type memKB struct{ fakeKB }

func (m *memKB) InsertIfSlugMissing(_ context.Context, d *KBDoc) (bool, error) {
	for _, e := range m.docs {
		if e.Slug == d.Slug {
			return false, nil
		}
	}
	m.docs = append(m.docs, *d)
	return true, nil
}

func TestSeedKB_IdempotentAndDryRun(t *testing.T) {
	repo := &memKB{}
	now := time.Now().UTC()
	dry, err := SeedKB(context.Background(), repo, StarterKB, false, now)
	if err != nil || len(dry.Created) != len(StarterKB) || len(repo.docs) != 0 {
		t.Fatalf("dry run must write nothing: %+v %v (%d docs)", dry, err, len(repo.docs))
	}
	if _, err := SeedKB(context.Background(), repo, StarterKB, true, now); err != nil {
		t.Fatal(err)
	}
	if len(repo.docs) != len(StarterKB) {
		t.Fatalf("want %d docs, got %d", len(StarterKB), len(repo.docs))
	}
	repo.docs[0].Title.En = "Edited by admin"
	again, err := SeedKB(context.Background(), repo, StarterKB, true, now)
	if err != nil || len(again.Created) != 0 || len(again.Skipped) != len(StarterKB) {
		t.Fatalf("re-run must skip everything: %+v %v", again, err)
	}
	if repo.docs[0].Title.En != "Edited by admin" {
		t.Fatal("re-run must not overwrite an edited doc")
	}
}
