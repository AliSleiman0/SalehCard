// Package todo is the admin console's shared team to-do list (client feedback
// 2026-11-12 #18): short tasks every admin can see, add, tick off, edit or
// delete. No RBAC domain — it sits behind AdminOnly only. Deletes are audited.
package todo

import (
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	maxTitleLen = 200
	maxNoteLen  = 1000
	// openLimit / doneLimit bound one fetch. Open tasks are fetched as a whole
	// set (then sorted soonest-due first in Go), so an old overdue task can
	// never be cut off by a newest-first limit; done tasks from the last
	// doneRetention are listed after, newest first.
	openLimit     = 1000
	doneLimit     = 200
	doneRetention = 30 * 24 * time.Hour
)

// Todo is one task on the shared list.
type Todo struct {
	ID        bson.ObjectID `bson:"_id,omitempty"       json:"id"`
	Title     string        `bson:"title"               json:"title"`
	Note      string        `bson:"note,omitempty"      json:"note,omitempty"`
	DueAt     *time.Time    `bson:"dueAt,omitempty"     json:"dueAt,omitempty"`
	Done      bool          `bson:"done"                json:"done"`
	DoneAt    *time.Time    `bson:"doneAt,omitempty"    json:"doneAt,omitempty"`
	DoneBy    string        `bson:"doneBy,omitempty"    json:"doneBy,omitempty"`
	CreatedBy string        `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	CreatedAt time.Time     `bson:"createdAt"           json:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"           json:"updatedAt"`
}

// Input is the create/update payload. On update, nil fields are left as-is.
type Input struct {
	Title    *string    `json:"title"`
	Note     *string    `json:"note"`
	DueAt    *time.Time `json:"dueAt"`
	ClearDue bool       `json:"clearDue"`
	Done     *bool      `json:"done"`
}

var errInvalid = errors.New("invalid")

// validate trims and checks the text fields; creating requires a title.
func (in *Input) validate(creating bool) (string, error) {
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		in.Title = &t
		if t == "" {
			return "title is required", errInvalid
		}
		if utf8.RuneCountInString(t) > maxTitleLen {
			return "title must be at most 200 characters", errInvalid
		}
	} else if creating {
		return "title is required", errInvalid
	}
	if in.Note != nil {
		n := strings.TrimSpace(*in.Note)
		in.Note = &n
		if utf8.RuneCountInString(n) > maxNoteLen {
			return "note must be at most 1000 characters", errInvalid
		}
	}
	return "", nil
}

// sortTodos puts open tasks first (dated ones soonest-first, then undated
// newest-first) and done tasks after (most recently completed first).
func sortTodos(ts []Todo) {
	sort.SliceStable(ts, func(i, j int) bool {
		a, b := ts[i], ts[j]
		if a.Done != b.Done {
			return !a.Done
		}
		if a.Done {
			return a.DoneAt != nil && (b.DoneAt == nil || a.DoneAt.After(*b.DoneAt))
		}
		switch {
		case a.DueAt != nil && b.DueAt != nil:
			return a.DueAt.Before(*b.DueAt)
		case a.DueAt != nil:
			return true
		case b.DueAt != nil:
			return false
		default:
			return a.CreatedAt.After(b.CreatedAt)
		}
	})
}

// buildUpdate turns a validated Input into a Mongo update document.
func buildUpdate(in Input, now time.Time, who string) bson.D {
	set := bson.D{{Key: "updatedAt", Value: now}}
	unset := bson.D{}
	if in.Title != nil {
		set = append(set, bson.E{Key: "title", Value: *in.Title})
	}
	if in.Note != nil {
		set = append(set, bson.E{Key: "note", Value: *in.Note})
	}
	if in.ClearDue {
		unset = append(unset, bson.E{Key: "dueAt", Value: ""})
	} else if in.DueAt != nil {
		set = append(set, bson.E{Key: "dueAt", Value: in.DueAt.UTC()})
	}
	if in.Done != nil {
		set = append(set, bson.E{Key: "done", Value: *in.Done})
		if *in.Done {
			set = append(set, bson.E{Key: "doneAt", Value: now}, bson.E{Key: "doneBy", Value: who})
		} else {
			unset = append(unset, bson.E{Key: "doneAt", Value: ""}, bson.E{Key: "doneBy", Value: ""})
		}
	}
	upd := bson.D{{Key: "$set", Value: set}}
	if len(unset) > 0 {
		upd = append(upd, bson.E{Key: "$unset", Value: unset})
	}
	return upd
}
