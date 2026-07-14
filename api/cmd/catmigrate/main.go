// Command catmigrate backfills the managed category taxonomy so the tree
// drill-down (Collections → Categories → Subcategories → Products) works on
// data imported before the feature existed. It is a DRY RUN unless you pass
// --apply, and is idempotent (safe to re-run).
//
// Two passes:
//
//	categories  Set each node's Mongo-native parentId (mapped from parentLegacyId)
//	            and its materialized ancestors chain + depth. Uniform tree key.
//	products    Set each product's categoryId by matching its categorySlug +
//	            rootDomain (fallbacks: category string, slug-only) to a node.
//
// Run (prod): add a Cosmos firewall rule for your egress IP, then feed MONGO_URI
// from the gitignored DEPLOY-CREDS.local.md, e.g.
//
//	MONGO_URI=... go run ./cmd/catmigrate            # dry run (both passes)
//	MONGO_URI=... go run ./cmd/catmigrate --apply     # write
//	MONGO_URI=... go run ./cmd/catmigrate categories  # one pass only
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

const maxDepth = 16 // cycle guard when walking parent pointers

// catNode is the subset of a category document catmigrate reads/writes.
type catNode struct {
	ID             bson.ObjectID   `bson:"_id"`
	LegacyID       *int            `bson:"legacyId"`
	ParentLegacyID *int            `bson:"parentLegacyId"`
	ParentID       *bson.ObjectID  `bson:"parentId"`
	Ancestors      []bson.ObjectID `bson:"ancestors"`
	Slug           string          `bson:"slug"`
	RootDomain     string          `bson:"rootDomain"`
	Depth          int             `bson:"depth"`
}

func main() {
	fs := flag.NewFlagSet("catmigrate", flag.ExitOnError)
	apply := fs.Bool("apply", false, "write changes (default is a dry run)")
	_ = fs.Parse(os.Args[1:])

	pass := "all"
	if fs.NArg() > 0 {
		pass = fs.Arg(0)
	}
	if pass != "all" && pass != "categories" && pass != "products" {
		log.Fatalf("unknown pass %q (want: categories | products | all)", pass)
	}

	_ = godotenv.Load()
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client, db, err := mongoplatform.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer mongoplatform.Disconnect(context.Background(), client)

	env := "local"
	if strings.Contains(cfg.MongoURI, "cosmos.azure.com") || strings.Contains(cfg.MongoURI, "mongocluster") {
		env = "PROD"
	}
	mode := "DRY RUN"
	if *apply {
		mode = "APPLY"
	}
	log.Printf("catmigrate: db=%s (%s) pass=%s mode=%s", cfg.DBName, env, pass, mode)

	if pass == "all" || pass == "categories" {
		migrateCategories(ctx, db, *apply)
	}
	if pass == "all" || pass == "products" {
		migrateProducts(ctx, db, *apply)
	}
	if !*apply {
		fmt.Printf("\nDRY RUN — re-run with --apply to write.\n")
	}
}

// loadCategories reads every category node.
func loadCategories(ctx context.Context, db *mongo.Database) []catNode {
	cur, err := db.Collection("categories").Find(ctx, bson.D{})
	if err != nil {
		log.Fatalf("categories find: %v", err)
	}
	var cats []catNode
	if err := cur.All(ctx, &cats); err != nil {
		log.Fatalf("categories decode: %v", err)
	}
	return cats
}

// migrateCategories sets parentId (from parentLegacyId) + ancestors + depth on
// every node, so all tree queries can key on parentId/ancestors uniformly.
func migrateCategories(ctx context.Context, db *mongo.Database, apply bool) {
	cats := loadCategories(ctx, db)
	byLegacy := map[int]bson.ObjectID{}
	byID := map[bson.ObjectID]*catNode{}
	for i := range cats {
		c := &cats[i]
		byID[c.ID] = c
		if c.LegacyID != nil {
			byLegacy[*c.LegacyID] = c.ID
		}
	}

	// Pass 1: resolve parentId from parentLegacyId where not already set.
	for i := range cats {
		c := &cats[i]
		if c.ParentID == nil && c.ParentLegacyID != nil {
			if pid, ok := byLegacy[*c.ParentLegacyID]; ok {
				c.ParentID = &pid
			} else {
				log.Printf("  WARN category %s (legacy %v): parentLegacyId %d has no matching node — treating as root",
					c.ID.Hex(), c.LegacyID, *c.ParentLegacyID)
			}
		}
	}

	// Pass 2: compute ancestors + depth by walking parentId chains.
	updated := 0
	for i := range cats {
		c := &cats[i]
		anc := ancestorsOf(c, byID)
		newDepth := len(anc)
		set := bson.D{}
		if !sameAncestors(c.Ancestors, anc) {
			set = append(set, bson.E{Key: "ancestors", Value: anc})
		}
		if c.Depth != newDepth {
			set = append(set, bson.E{Key: "depth", Value: newDepth})
		}
		if c.ParentID != nil {
			set = append(set, bson.E{Key: "parentId", Value: *c.ParentID})
		}
		if len(set) == 0 {
			continue
		}
		updated++
		if apply {
			set = append(set, bson.E{Key: "updatedAt", Value: time.Now().UTC()})
			if _, err := db.Collection("categories").UpdateByID(ctx, c.ID, bson.D{{Key: "$set", Value: set}}); err != nil {
				log.Fatalf("category %s update: %v", c.ID.Hex(), err)
			}
		}
	}
	fmt.Printf("categories: %d node(s), %d would be updated (parentId/ancestors/depth).\n", len(cats), updated)
}

// ancestorsOf walks c's parent chain to the root, returning root→…→parent.
func ancestorsOf(c *catNode, byID map[bson.ObjectID]*catNode) []bson.ObjectID {
	var chain []bson.ObjectID
	cur := c
	for range maxDepth {
		if cur.ParentID == nil {
			break
		}
		parent, ok := byID[*cur.ParentID]
		if !ok {
			break
		}
		chain = append([]bson.ObjectID{parent.ID}, chain...) // prepend (root-first)
		cur = parent
	}
	return chain
}

// migrateProducts sets categoryId on every product by matching its slug/domain
// to a taxonomy node.
func migrateProducts(ctx context.Context, db *mongo.Database, apply bool) {
	cats := loadCategories(ctx, db)
	// Match keys: rootDomain+"/"+slug (preferred, disambiguates shared slugs) and
	// slug alone (fallback). Prefer the deepest node when several share a key.
	byDomainSlug := map[string]catNode{}
	bySlug := map[string]catNode{}
	for _, c := range cats {
		dk := c.RootDomain + "/" + c.Slug
		if ex, ok := byDomainSlug[dk]; !ok || c.Depth > ex.Depth {
			byDomainSlug[dk] = c
		}
		if ex, ok := bySlug[c.Slug]; !ok || c.Depth > ex.Depth {
			bySlug[c.Slug] = c
		}
	}

	cur, err := db.Collection("products").Find(ctx, bson.D{})
	if err != nil {
		log.Fatalf("products find: %v", err)
	}
	var prods []struct {
		ID           bson.ObjectID  `bson:"_id"`
		CategoryID   *bson.ObjectID `bson:"categoryId"`
		Category     string         `bson:"category"`
		CategorySlug string         `bson:"categorySlug"`
		RootDomain   string         `bson:"rootDomain"`
	}
	if err := cur.All(ctx, &prods); err != nil {
		log.Fatalf("products decode: %v", err)
	}

	matched, already, unmatched := 0, 0, 0
	for _, p := range prods {
		if p.CategoryID != nil {
			already++
			continue
		}
		node, ok := matchNode(p.CategorySlug, p.Category, p.RootDomain, byDomainSlug, bySlug)
		if !ok {
			unmatched++
			log.Printf("  no category match: product %s (slug=%q category=%q rootDomain=%q)",
				p.ID.Hex(), p.CategorySlug, p.Category, p.RootDomain)
			continue
		}
		matched++
		if apply {
			set := bson.D{{Key: "categoryId", Value: node.ID}, {Key: "updatedAt", Value: time.Now().UTC()}}
			if _, err := db.Collection("products").UpdateByID(ctx, p.ID, bson.D{{Key: "$set", Value: set}}); err != nil {
				log.Fatalf("product %s update: %v", p.ID.Hex(), err)
			}
		}
	}
	fmt.Printf("products: %d total, %d already assigned, %d would be assigned, %d unmatched.\n",
		len(prods), already, matched, unmatched)
}

// matchNode resolves the best taxonomy node for a product: prefer categorySlug
// then the flat category string, each disambiguated by rootDomain, falling back
// to a slug-only match.
func matchNode(categorySlug, category, rootDomain string, byDomainSlug, bySlug map[string]catNode) (catNode, bool) {
	for _, slug := range []string{categorySlug, category} {
		if slug == "" {
			continue
		}
		if n, ok := byDomainSlug[rootDomain+"/"+slug]; ok {
			return n, true
		}
	}
	for _, slug := range []string{categorySlug, category} {
		if slug == "" {
			continue
		}
		if n, ok := bySlug[slug]; ok {
			return n, true
		}
	}
	return catNode{}, false
}

// sameAncestors reports whether two ancestor chains are identical.
func sameAncestors(a, b []bson.ObjectID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
