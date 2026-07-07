// Command findphone is a throwaway, read-only lookup: given an email it prints
// that user's name/email/phone/role from the users collection. Connects with
// MONGO_URI (never logged). Usage: MONGO_URI=... go run ./cmd/findphone <email>
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		fmt.Fprintln(os.Stderr, "MONGO_URI is required")
		os.Exit(1)
	}
	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "salehcard"
	}
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: findphone <email>")
		os.Exit(1)
	}
	email := strings.ToLower(strings.TrimSpace(os.Args[1]))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	col := client.Database(dbName).Collection("users")

	type row struct {
		Name  string  `bson:"name"`
		Email string  `bson:"email"`
		Phone *string `bson:"phone"`
		Role  string  `bson:"role"`
	}

	// Exact (case-insensitive) email match first.
	var u row
	err = col.FindOne(ctx, bson.D{{Key: "email", Value: bson.D{{Key: "$regex", Value: "^" + regexpEscape(email) + "$"}, {Key: "$options", Value: "i"}}}}).Decode(&u)
	if err == nil {
		printRow("Exact match", u.Name, u.Email, u.Phone, u.Role)
		return
	}
	if err != mongo.ErrNoDocuments {
		fmt.Fprintln(os.Stderr, "query:", err)
		os.Exit(1)
	}

	// Not found — list admins so the caller can see what exists.
	fmt.Printf("No user with email %q. Admin accounts:\n", email)
	cur, err := col.Find(ctx, bson.D{{Key: "role", Value: "admin"}})
	if err != nil {
		fmt.Fprintln(os.Stderr, "list admins:", err)
		os.Exit(1)
	}
	defer func() { _ = cur.Close(ctx) }()
	n := 0
	for cur.Next(ctx) {
		var a row
		if err := cur.Decode(&a); err != nil {
			continue
		}
		printRow("admin", a.Name, a.Email, a.Phone, a.Role)
		n++
	}
	if n == 0 {
		fmt.Println("  (none)")
	}
}

func printRow(tag, name, email string, phone *string, role string) {
	p := "(none)"
	if phone != nil {
		p = *phone
	}
	fmt.Printf("  [%s] name=%q email=%q role=%s phone=%s\n", tag, name, email, role, p)
}

func regexpEscape(s string) string {
	specials := `\.+*?()|[]{}^$`
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(specials, r) {
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
