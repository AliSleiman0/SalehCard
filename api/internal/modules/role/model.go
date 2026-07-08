package role

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Role is a custom admin role: a named set of RBAC permissions. Only custom
// roles live in the collection — the built-in Super Admin is implicit (an
// admin user with no adminRoleId carries the "*" wildcard) and can never be
// edited or deleted. Permissions are always stored normalized (valid, deduped,
// manage implies view; see NormalizePermissions).
type Role struct {
	ID          bson.ObjectID `bson:"_id,omitempty"          json:"id"`
	Name        string        `bson:"name"                   json:"name"`
	Description string        `bson:"description,omitempty"  json:"description,omitempty"`
	Permissions []string      `bson:"permissions"            json:"permissions"`
	CreatedAt   time.Time     `bson:"createdAt"              json:"createdAt"`
	UpdatedAt   time.Time     `bson:"updatedAt"              json:"updatedAt"`
}
