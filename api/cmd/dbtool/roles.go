package main

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/role"
)

// dbtoolActor labels audit rows written by this CLI (there is no signed-in admin).
const dbtoolActor = "dbtool (ops CLI)"

// reorderFlags moves "--flag" arguments to the front, because the standard flag
// package stops parsing at the first positional argument ("role-grant x y z --apply").
func reorderFlags(args []string) []string {
	var flags, rest []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
		} else {
			rest = append(rest, a)
		}
	}
	return append(flags, rest...)
}

// grantPerms returns the permissions to add for an area at a level: view alone,
// or manage plus view (manage always implies view, as the API normalizes).
func grantPerms(area, level string) ([]string, error) {
	view, manage := area+".view", area+".manage"
	if !role.ValidPermission(view) {
		return nil, fmt.Errorf("%q is not an RBAC area (see modules/role/permissions.go)", area)
	}
	switch level {
	case "view":
		return []string{view}, nil
	case "manage":
		return []string{view, manage}, nil
	}
	return nil, fmt.Errorf("level must be view or manage, not %q", level)
}

// missingPerms is the subset of want that have does not contain, in want's order.
func missingPerms(have, want []string) []string {
	var out []string
	for _, p := range want {
		if !slices.Contains(have, p) {
			out = append(out, p)
		}
	}
	return out
}

func loadRoles(ctx context.Context, db *mongo.Database) []role.Role {
	cur, err := db.Collection("roles").Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		log.Fatalf("roles: %v", err)
	}
	var out []role.Role
	if err := cur.All(ctx, &out); err != nil {
		log.Fatalf("roles: %v", err)
	}
	return out
}

func holders(ctx context.Context, db *mongo.Database, id bson.ObjectID) int64 {
	n, err := db.Collection("users").CountDocuments(ctx, bson.D{{Key: "adminRoleId", Value: id}})
	if err != nil {
		return -1
	}
	return n
}

// cmdRoles lists custom roles. Super admins (admins with no custom role) hold
// every permission implicitly and are not listed.
func cmdRoles(ctx context.Context, db *mongo.Database) {
	roles := loadRoles(ctx, db)
	if len(roles) == 0 {
		fmt.Println("No custom roles: every admin is a super admin and already has every permission.")
		return
	}
	for _, r := range roles {
		perms := slices.Clone(r.Permissions)
		slices.Sort(perms)
		fmt.Printf("%s  %-28s holders=%d  %s\n", r.ID.Hex(), r.Name, holders(ctx, db, r.ID), strings.Join(perms, ", "))
	}
}

// cmdRoleGrant adds area.view (+ area.manage) to one role or all custom roles.
// $addToSet keeps it idempotent; each changed role gets an admin_audit_log row
// (role.update, actor "dbtool (ops CLI)") like an edit from admin → Roles.
func cmdRoleGrant(ctx context.Context, db *mongo.Database, area, level, target string, apply bool) {
	want, err := grantPerms(area, level)
	if err != nil {
		log.Fatal(err)
	}
	roles := loadRoles(ctx, db)
	if target != "all" {
		id, err := bson.ObjectIDFromHex(target)
		if err != nil {
			log.Fatalf("roleId must be a role id from `dbtool roles` or \"all\", not %q", target)
		}
		roles = slices.DeleteFunc(roles, func(r role.Role) bool { return r.ID != id })
		if len(roles) == 0 {
			log.Fatalf("no custom role with id %s", target)
		}
	}
	if len(roles) == 0 {
		fmt.Println("No custom roles: every admin is a super admin and already has every permission. Nothing to do.")
		return
	}

	changed := 0
	for _, r := range roles {
		add := missingPerms(r.Permissions, want)
		if len(add) == 0 {
			fmt.Printf("  = %-28s already has %s\n", r.Name, strings.Join(want, " + "))
			continue
		}
		changed++
		if !apply {
			fmt.Printf("  + %-28s would add %s (holders=%d)\n", r.Name, strings.Join(add, ", "), holders(ctx, db, r.ID))
			continue
		}
		now := time.Now().UTC()
		var after role.Role
		err := db.Collection("roles").FindOneAndUpdate(ctx,
			bson.D{{Key: "_id", Value: r.ID}},
			bson.D{
				{Key: "$addToSet", Value: bson.D{{Key: "permissions", Value: bson.D{{Key: "$each", Value: add}}}}},
				{Key: "$set", Value: bson.D{{Key: "updatedAt", Value: now}}},
			},
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		).Decode(&after)
		if err != nil {
			log.Fatalf("update %s: %v", r.Name, err)
		}
		if _, err := db.Collection("admin_audit_log").InsertOne(ctx, audit.Entry{
			ActorEmail: dbtoolActor, Action: audit.ActionRoleUpdate, TargetType: "role", TargetID: r.ID.Hex(),
			Summary: map[string]any{"name": after.Name, "permissions": after.Permissions, "added": add},
			At:      now,
		}); err != nil {
			log.Printf("warning: %s updated but the audit row failed: %v", r.Name, err)
		}
		perms := slices.Clone(after.Permissions)
		slices.Sort(perms)
		fmt.Printf("  OK %-27s now holds %s\n", after.Name, strings.Join(perms, ", "))
	}

	switch {
	case changed == 0:
		fmt.Println("Nothing to change.")
	case !apply:
		fmt.Printf("DRY RUN: %d role(s) would change. Re-run with --apply to write.\n", changed)
	default:
		fmt.Println("Done. Holders get the new access on their next token refresh or login.")
	}
}
