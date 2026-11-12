// Package todo is the admin console's shared team to-do list (client feedback
// 2026-11-12 #18): short tasks any admin with todos.view can see, and admins
// with todos.manage can add, tick off, edit or delete.
package todo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

const (
	maxTitleLen = 200
	maxNoteLen  = 1000
	// listLimit caps one fetch; a team to-do list stays small, and done items
	// older than doneRetention drop off the list.
	listLimit     = 300
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

type handler struct{ col *mongo.Collection }

// RegisterAdminRoutes mounts the to-do routes on the /api/admin group (wrapped
// by the server in the "todos" RBAC domain: GET is todos.view, else manage).
func RegisterAdminRoutes(r chi.Router, db *mongo.Database) {
	col := db.Collection("admin_todos")
	_, _ = col.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{{Key: "done", Value: 1}, {Key: "createdAt", Value: -1}},
	})
	h := &handler{col: col}
	r.Get("/todos", h.list)
	r.Post("/todos", h.create)
	r.Put("/todos/{id}", h.update)
	r.Delete("/todos/{id}", h.delete)
}

func actor(r *http.Request) string {
	if c, ok := auth.ClaimsFromContext(r.Context()); ok && c != nil {
		if c.Email != "" {
			return c.Email
		}
		return c.Phone
	}
	return ""
}

// list returns open tasks (soonest due first, then newest) followed by tasks
// completed in the last 30 days.
func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	cutoff := time.Now().UTC().Add(-doneRetention)
	filter := bson.D{{Key: "$or", Value: bson.A{
		bson.D{{Key: "done", Value: false}},
		bson.D{{Key: "doneAt", Value: bson.D{{Key: "$gte", Value: cutoff}}}},
	}}}
	cur, err := h.col.Find(r.Context(), filter, options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(listLimit))
	if err != nil {
		response.InternalError(w)
		return
	}
	out := []Todo{}
	if err := cur.All(r.Context(), &out); err != nil {
		response.InternalError(w)
		return
	}
	sortTodos(out)
	response.OK(w, out)
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

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var in Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if msg, err := in.validate(true); err != nil {
		response.BadRequest(w, msg)
		return
	}
	now := time.Now().UTC()
	t := Todo{ID: bson.NewObjectID(), Title: *in.Title, DueAt: in.DueAt, CreatedBy: actor(r), CreatedAt: now, UpdatedAt: now}
	if in.Note != nil {
		t.Note = *in.Note
	}
	if _, err := h.col.InsertOne(r.Context(), t); err != nil {
		response.InternalError(w)
		return
	}
	response.JSON(w, http.StatusCreated, response.Response{Success: true, Data: t})
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

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}
	var in Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if msg, err := in.validate(false); err != nil {
		response.BadRequest(w, msg)
		return
	}
	var out Todo
	err = h.col.FindOneAndUpdate(r.Context(), bson.D{{Key: "_id", Value: id}},
		buildUpdate(in, time.Now().UTC(), actor(r)),
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		response.NotFound(w)
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, out)
}

func (h *handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}
	res, err := h.col.DeleteOne(r.Context(), bson.D{{Key: "_id", Value: id}})
	if err != nil {
		response.InternalError(w)
		return
	}
	if res.DeletedCount == 0 {
		response.NotFound(w)
		return
	}
	response.OK(w, map[string]bool{"deleted": true})
}
