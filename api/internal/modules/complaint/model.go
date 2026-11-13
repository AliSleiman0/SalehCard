// Package complaint is the customer complaint desk: a customer opens a
// complaint (optionally about one of their orders, with a photo), support staff
// follow up from the admin console (public replies + internal notes, status,
// assignee), and every admin reply / resolution reaches the customer as an
// in-app notification (inbox row + push). No SMS: SMS is billed.
//
// A complaint is ONE document with an embedded, capped message thread, so every
// write is a single guarded FindOneAndUpdate (Cosmos has no multi-document
// transactions). Internal notes are admin-only and are stripped server-side
// from every customer response.
package complaint

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Categories a customer can file under.
const (
	CategoryOrder   = "order"
	CategoryTopup   = "topup"
	CategoryAccount = "account"
	CategoryOther   = "other"
)

// Lifecycle statuses. open / in_progress count toward the per-user open cap; a
// customer reply reopens a resolved complaint; closed is final for customers.
const (
	StatusOpen       = "open"
	StatusInProgress = "in_progress"
	StatusResolved   = "resolved"
	StatusClosed     = "closed"
)

// Message authors.
const (
	AuthorCustomer = "customer"
	AuthorAdmin    = "admin"
)

const (
	maxSubjectLen = 120
	maxBodyLen    = 2000
	// maxMessages caps the embedded thread so the document stays small.
	maxMessages = 200
	// maxOpenPerUser caps how many open / in-progress complaints one customer
	// may hold at once (409 TOO_MANY_OPEN beyond it).
	maxOpenPerUser = 5
	// numberBase offsets the human-readable number so the first complaint is
	// C-1001 rather than C-1.
	numberBase = 1000
	// PhotoPrefix is the blob keyspace for complaint photos.
	PhotoPrefix = "complaints/"
)

// ValidCategory reports whether c is a known category.
func ValidCategory(c string) bool {
	switch c {
	case CategoryOrder, CategoryTopup, CategoryAccount, CategoryOther:
		return true
	}
	return false
}

// ValidStatus reports whether s is a known status.
func ValidStatus(s string) bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusResolved, StatusClosed:
		return true
	}
	return false
}

// Ref formats the human-readable complaint number ("C-1042").
func Ref(number int) string { return fmt.Sprintf("C-%d", number) }

// Message is one entry in a complaint thread. AdminID / AdminLabel identify the
// staff author (admin console only); Internal marks an admin-only note.
type Message struct {
	ID         bson.ObjectID  `bson:"id" json:"id"`
	Author     string         `bson:"author" json:"author"`
	AdminID    *bson.ObjectID `bson:"adminId,omitempty" json:"adminId,omitempty"`
	AdminLabel string         `bson:"adminLabel,omitempty" json:"adminLabel,omitempty"`
	Body       string         `bson:"body" json:"body"`
	PhotoURL   string         `bson:"photoUrl,omitempty" json:"photoUrl,omitempty"`
	Internal   bool           `bson:"internal" json:"internal"`
	CreatedAt  time.Time      `bson:"createdAt" json:"createdAt"`
}

// Complaint is one stored complaint (collection complaints).
type Complaint struct {
	ID                    bson.ObjectID  `bson:"_id" json:"id"`
	Number                int            `bson:"number" json:"number"`
	UserID                bson.ObjectID  `bson:"userId" json:"userId"`
	Category              string         `bson:"category" json:"category"`
	OrderID               *bson.ObjectID `bson:"orderId,omitempty" json:"orderId,omitempty"`
	Subject               string         `bson:"subject" json:"subject"`
	Status                string         `bson:"status" json:"status"`
	AssigneeID            *bson.ObjectID `bson:"assigneeId,omitempty" json:"assigneeId,omitempty"`
	Messages              []Message      `bson:"messages" json:"messages"`
	LastCustomerMessageAt *time.Time     `bson:"lastCustomerMessageAt,omitempty" json:"lastCustomerMessageAt,omitempty"`
	LastAdminMessageAt    *time.Time     `bson:"lastAdminMessageAt,omitempty" json:"lastAdminMessageAt,omitempty"`
	CreatedAt             time.Time      `bson:"createdAt" json:"createdAt"`
	UpdatedAt             time.Time      `bson:"updatedAt" json:"updatedAt"`
}

// AwaitingReply reports whether the customer spoke last (internal notes do not
// count as a reply), i.e. support owes them an answer.
func (c *Complaint) AwaitingReply() bool {
	if c.LastCustomerMessageAt == nil {
		return false
	}
	return c.LastAdminMessageAt == nil || c.LastCustomerMessageAt.After(*c.LastAdminMessageAt)
}

// CustomerMessage is a thread entry as the customer sees it: never internal,
// never carrying the staff member's identity.
type CustomerMessage struct {
	ID        bson.ObjectID `json:"id"`
	Author    string        `json:"author"`
	Body      string        `json:"body"`
	PhotoURL  string        `json:"photoUrl,omitempty"`
	CreatedAt time.Time     `json:"createdAt"`
}

// CustomerView is the customer-facing shape of a complaint. Messages is nil on
// list rows (the list endpoint does not load threads).
type CustomerView struct {
	ID        bson.ObjectID     `json:"id"`
	Number    int               `json:"number"`
	Ref       string            `json:"ref"`
	Category  string            `json:"category"`
	OrderID   *bson.ObjectID    `json:"orderId,omitempty"`
	Subject   string            `json:"subject"`
	Status    string            `json:"status"`
	Messages  []CustomerMessage `json:"messages,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// ForCustomer projects c for its owner: internal notes and staff identities are
// dropped here, server-side, for every customer response.
func (c *Complaint) ForCustomer() CustomerView {
	v := CustomerView{
		ID: c.ID, Number: c.Number, Ref: Ref(c.Number), Category: c.Category,
		OrderID: c.OrderID, Subject: c.Subject, Status: c.Status,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	if c.Messages != nil {
		v.Messages = make([]CustomerMessage, 0, len(c.Messages))
		for _, m := range c.Messages {
			if m.Internal {
				continue
			}
			v.Messages = append(v.Messages, CustomerMessage{
				ID: m.ID, Author: m.Author, Body: m.Body, PhotoURL: m.PhotoURL, CreatedAt: m.CreatedAt,
			})
		}
	}
	return v
}

// CreateInput is the customer's POST /api/v1/complaints body.
type CreateInput struct {
	Category string `json:"category"`
	Subject  string `json:"subject"`
	Message  string `json:"message"`
	OrderID  string `json:"orderId"`
	PhotoURL string `json:"photoUrl"`
}

// ReplyInput is a customer reply body.
type ReplyInput struct {
	Body     string `json:"body"`
	PhotoURL string `json:"photoUrl"`
}

// AdminReplyInput is an admin reply / internal-note body.
type AdminReplyInput struct {
	Body     string `json:"body"`
	Internal bool   `json:"internal"`
}

// cleanText trims s and enforces a non-empty, max-rune length.
func cleanText(s string, max int, field string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", invalid(field + " is required")
	}
	if utf8.RuneCountInString(s) > max {
		return "", invalid(fmt.Sprintf("%s must be at most %d characters", field, max))
	}
	return s, nil
}
