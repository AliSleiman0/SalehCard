package todo

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func sp(s string) *string { return &s }

func TestValidate(t *testing.T) {
	in := Input{Title: sp("  call supplier  ")}
	_, err := in.validate(true)
	require.NoError(t, err)
	assert.Equal(t, "call supplier", *in.Title)

	_, err = (&Input{}).validate(true)
	assert.Error(t, err, "create needs a title")
	_, err = (&Input{}).validate(false)
	assert.NoError(t, err, "update may omit the title")
	_, err = (&Input{Title: sp("   ")}).validate(false)
	assert.Error(t, err)
	_, err = (&Input{Title: sp(strings.Repeat("x", 201))}).validate(true)
	assert.Error(t, err)
	_, err = (&Input{Title: sp("ok"), Note: sp(strings.Repeat("n", 1001))}).validate(true)
	assert.Error(t, err)
}

func TestSortTodos(t *testing.T) {
	base := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	at := func(h int) *time.Time { v := base.Add(time.Duration(h) * time.Hour); return &v }
	ts := []Todo{
		{Title: "done-old", Done: true, DoneAt: at(-5)},
		{Title: "undated-old", CreatedAt: base.Add(-2 * time.Hour)},
		{Title: "due-later", DueAt: at(48)},
		{Title: "done-new", Done: true, DoneAt: at(-1)},
		{Title: "undated-new", CreatedAt: base},
		{Title: "due-soon", DueAt: at(2)},
	}
	sortTodos(ts)
	got := make([]string, len(ts))
	for i, x := range ts {
		got[i] = x.Title
	}
	assert.Equal(t, []string{"due-soon", "due-later", "undated-new", "undated-old", "done-new", "done-old"}, got)
}

func TestBuildUpdate(t *testing.T) {
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	yes, no := true, false

	upd := buildUpdate(Input{Done: &yes}, now, "a@b.c")
	set := upd[0].Value.(bson.D)
	assert.Contains(t, set, bson.E{Key: "done", Value: true})
	assert.Contains(t, set, bson.E{Key: "doneBy", Value: "a@b.c"})
	assert.Len(t, upd, 1, "ticking off adds no $unset")

	upd = buildUpdate(Input{Done: &no, ClearDue: true}, now, "a@b.c")
	require.Len(t, upd, 2)
	unset := upd[1].Value.(bson.D)
	keys := []string{}
	for _, e := range unset {
		keys = append(keys, e.Key)
	}
	assert.ElementsMatch(t, []string{"dueAt", "doneAt", "doneBy"}, keys)
}
