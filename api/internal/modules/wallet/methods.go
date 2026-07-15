package wallet

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// MethodFieldType is the kind of customer input a top-up method collects. It
// mirrors product.InputFieldType in spirit but is deliberately its own, smaller
// set (top-up methods are admin free-text config, not localized catalog specs).
type MethodFieldType string

const (
	MethodFieldText   MethodFieldType = "text"   // free text (e.g. a reference number)
	MethodFieldNumber MethodFieldType = "number" // numeric text
	MethodFieldSelect MethodFieldType = "select" // one of Options
	MethodFieldFile   MethodFieldType = "file"   // an uploaded document URL (transfer proof)
)

// validMethodFieldType reports whether t is a known input kind.
func validMethodFieldType(t MethodFieldType) bool {
	switch t {
	case MethodFieldText, MethodFieldNumber, MethodFieldSelect, MethodFieldFile:
		return true
	}
	return false
}

// Method field / name / instructions length caps — sanity bounds on admin input.
const (
	maxMethodNameLen         = 80
	maxMethodInstructionsLen = 2000
	maxMethodFields          = 12
	maxMethodFieldLabelLen   = 120
	maxMethodFieldKeyLen     = 60
	maxMethodOptions         = 40
)

// MethodField is one input a customer fills when paying via a manual top-up
// method (Account name, Reference #, an uploaded transfer screenshot, …). Label
// is admin-typed and snapshotted onto the request so the admin queue renders it.
type MethodField struct {
	Key         string          `bson:"key"                   json:"key"`
	Label       string          `bson:"label"                 json:"label"`
	Type        MethodFieldType `bson:"type"                  json:"type"`
	Required    bool            `bson:"required"              json:"required"`
	Options     []string        `bson:"options,omitempty"     json:"options,omitempty"`
	Placeholder string          `bson:"placeholder,omitempty" json:"placeholder,omitempty"`
}

// TopUpMethod is an admin-defined manual funding channel shown to customers on
// the wallet top-up screen. It is informational + collect-only: it never moves
// money — the customer submits the collected fields and an admin manually
// approves the resulting request to credit the wallet. Instructions carry the
// "how to pay" text (e.g. the bank account to transfer to).
type TopUpMethod struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string        `bson:"name"          json:"name"`
	Instructions string        `bson:"instructions"  json:"instructions"`
	Enabled      bool          `bson:"enabled"       json:"enabled"`
	SortOrder    int           `bson:"sortOrder"     json:"sortOrder"`
	Fields       []MethodField `bson:"fields"        json:"fields"`
	CreatedAt    time.Time     `bson:"createdAt"     json:"createdAt"`
	UpdatedAt    time.Time     `bson:"updatedAt"     json:"updatedAt"`
}

// MethodInput is the admin create/update payload for a top-up method.
type MethodInput struct {
	Name         string        `json:"name"`
	Instructions string        `json:"instructions"`
	Enabled      bool          `json:"enabled"`
	SortOrder    int           `json:"sortOrder"`
	Fields       []MethodField `json:"fields"`
}

// normalizeAndValidate trims and validates admin input, returning the cleaned
// method (fields normalized) or a 400-classified error. Field keys must be
// unique and non-empty; select fields need at least one option.
func (in MethodInput) normalizeAndValidate() (*TopUpMethod, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, badMethod("a method name is required")
	}
	if len(name) > maxMethodNameLen {
		return nil, badMethod("method name is too long")
	}
	instructions := strings.TrimSpace(in.Instructions)
	if len(instructions) > maxMethodInstructionsLen {
		return nil, badMethod("instructions are too long")
	}
	if len(in.Fields) > maxMethodFields {
		return nil, badMethod("too many input fields")
	}

	seen := map[string]bool{}
	fields := make([]MethodField, 0, len(in.Fields))
	for _, f := range in.Fields {
		key := strings.TrimSpace(f.Key)
		label := strings.TrimSpace(f.Label)
		if key == "" {
			return nil, badMethod("every input field needs a key")
		}
		if len(key) > maxMethodFieldKeyLen {
			return nil, badMethod("input field key is too long")
		}
		if seen[key] {
			return nil, badMethod("input field keys must be unique: " + key)
		}
		seen[key] = true
		if label == "" {
			return nil, badMethod("every input field needs a label")
		}
		if len(label) > maxMethodFieldLabelLen {
			return nil, badMethod("input field label is too long")
		}
		if !validMethodFieldType(f.Type) {
			return nil, badMethod("unknown input field type: " + string(f.Type))
		}
		var opts []string
		if f.Type == MethodFieldSelect {
			for _, o := range f.Options {
				o = strings.TrimSpace(o)
				if o != "" {
					opts = append(opts, o)
				}
			}
			if len(opts) == 0 {
				return nil, badMethod("a select field needs at least one option")
			}
			if len(opts) > maxMethodOptions {
				return nil, badMethod("too many options for a select field")
			}
		}
		fields = append(fields, MethodField{
			Key:         key,
			Label:       label,
			Type:        f.Type,
			Required:    f.Required,
			Options:     opts,
			Placeholder: strings.TrimSpace(f.Placeholder),
		})
	}

	return &TopUpMethod{
		Name:         name,
		Instructions: instructions,
		Enabled:      in.Enabled,
		SortOrder:    in.SortOrder,
		Fields:       fields,
	}, nil
}

// badMethod builds a 400-classified validation error.
func badMethod(msg string) error {
	return &apperrors.AppError{Code: "BAD_REQUEST", Message: msg, Err: apperrors.ErrBadRequest}
}

// MethodStore defines persistence for admin-managed top-up methods.
type MethodStore interface {
	Create(ctx context.Context, m *TopUpMethod) error
	Update(ctx context.Context, id bson.ObjectID, in MethodInput) (*TopUpMethod, error)
	Delete(ctx context.Context, id bson.ObjectID) error
	List(ctx context.Context) ([]*TopUpMethod, error)        // admin: all, sorted
	ListEnabled(ctx context.Context) ([]*TopUpMethod, error) // customer: enabled only
	GetByID(ctx context.Context, id bson.ObjectID) (*TopUpMethod, error)
}

// MethodRepo is the MongoDB-backed MethodStore over topup_methods.
type MethodRepo struct {
	col *mongo.Collection
}

// NewMethodRepo constructs a MethodRepo over topup_methods.
func NewMethodRepo(db *mongo.Database) *MethodRepo {
	return &MethodRepo{col: db.Collection("topup_methods")}
}

// EnsureMethodIndexes creates the display-order index (enabled methods are
// listed sorted by sortOrder then name).
func EnsureMethodIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("topup_methods").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "sortOrder", Value: 1}, {Key: "name", Value: 1}},
	})
	return err
}

// Create inserts a method, stamping ID/timestamps.
func (r *MethodRepo) Create(ctx context.Context, m *TopUpMethod) error {
	if m.ID.IsZero() {
		m.ID = bson.NewObjectID()
	}
	now := time.Now().UTC()
	m.CreatedAt = now
	m.UpdatedAt = now
	if m.Fields == nil {
		m.Fields = []MethodField{}
	}
	_, err := r.col.InsertOne(ctx, m)
	return err
}

// Update replaces the editable fields of a method, returning the post-update doc.
func (r *MethodRepo) Update(ctx context.Context, id bson.ObjectID, in MethodInput) (*TopUpMethod, error) {
	clean, err := in.normalizeAndValidate()
	if err != nil {
		return nil, err
	}
	if clean.Fields == nil {
		clean.Fields = []MethodField{}
	}
	var out TopUpMethod
	err = r.col.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "name", Value: clean.Name},
			{Key: "instructions", Value: clean.Instructions},
			{Key: "enabled", Value: clean.Enabled},
			{Key: "sortOrder", Value: clean.SortOrder},
			{Key: "fields", Value: clean.Fields},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&out)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

// Delete removes a method (existing requests keep their snapshotted method name).
func (r *MethodRepo) Delete(ctx context.Context, id bson.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// List returns every method, display order first (admin view).
func (r *MethodRepo) List(ctx context.Context) ([]*TopUpMethod, error) {
	return r.find(ctx, bson.D{})
}

// ListEnabled returns only enabled methods, display order first (customer view).
func (r *MethodRepo) ListEnabled(ctx context.Context) ([]*TopUpMethod, error) {
	return r.find(ctx, bson.D{{Key: "enabled", Value: true}})
}

func (r *MethodRepo) find(ctx context.Context, filter bson.D) ([]*TopUpMethod, error) {
	opts := options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}, {Key: "name", Value: 1}})
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*TopUpMethod{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetByID returns a single method (ErrNotFound when missing).
func (r *MethodRepo) GetByID(ctx context.Context, id bson.ObjectID) (*TopUpMethod, error) {
	var m TopUpMethod
	err := r.col.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&m)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}
