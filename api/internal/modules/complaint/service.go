package complaint

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/platform/photoupload"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// guardRetries bounds the optimistic re-read loop when a guarded update loses a
// race to a concurrent status change.
const guardRetries = 3

// Service holds the complaint rules shared by the customer and admin handlers.
type Service struct {
	repo    Repository
	ntf     notification.Notifier
	photoOK photoupload.Checker
	now     func() time.Time
}

// NewService wires a Service. photoOK validates client-submitted photo URLs
// (photoupload.NewChecker over the complaints/ keyspace).
func NewService(repo Repository, ntf notification.Notifier, photoOK photoupload.Checker) *Service {
	if ntf == nil {
		ntf = notification.Nop{}
	}
	return &Service{repo: repo, ntf: ntf, photoOK: photoOK, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) checkPhoto(u string) (string, error) {
	u = strings.TrimSpace(u)
	if u == "" {
		return "", nil
	}
	if s.photoOK == nil || !s.photoOK(u) {
		return "", invalid("photoUrl must be a photo uploaded through the complaint form")
	}
	return u, nil
}

// Create opens a complaint for userID. The first message is the description.
func (s *Service) Create(ctx context.Context, userID bson.ObjectID, in CreateInput) (*Complaint, error) {
	category := strings.TrimSpace(in.Category)
	if !ValidCategory(category) {
		return nil, invalid("category must be one of order, topup, account, other")
	}
	subject, err := cleanText(in.Subject, maxSubjectLen, "subject")
	if err != nil {
		return nil, err
	}
	body, err := cleanText(in.Message, maxBodyLen, "message")
	if err != nil {
		return nil, err
	}
	photo, err := s.checkPhoto(in.PhotoURL)
	if err != nil {
		return nil, err
	}
	var orderID *bson.ObjectID
	if raw := strings.TrimSpace(in.OrderID); raw != "" {
		id, err := bson.ObjectIDFromHex(raw)
		if err != nil {
			return nil, invalid("orderId is not a valid order")
		}
		o, err := s.repo.Order(ctx, id)
		if errors.Is(err, errNotFound) || (err == nil && o.UserID != userID) {
			// Someone else's order reads exactly like a missing one.
			return nil, invalid("orderId is not one of your orders")
		}
		if err != nil {
			return nil, err
		}
		orderID = &id
	}

	open, err := s.repo.CountOpenForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if open >= maxOpenPerUser {
		return nil, errTooMany
	}

	number, err := s.repo.NextNumber(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now()
	c := &Complaint{
		ID: bson.NewObjectID(), Number: number, UserID: userID,
		Category: category, OrderID: orderID, Subject: subject, Status: StatusOpen,
		Messages: []Message{{
			ID: bson.NewObjectID(), Author: AuthorCustomer, Body: body, PhotoURL: photo, CreatedAt: now,
		}},
		LastCustomerMessageAt: &now, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.Insert(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListMine is the customer's own list, newest first.
func (s *Service) ListMine(ctx context.Context, userID bson.ObjectID, p pagination.Params) ([]Complaint, int64, error) {
	return s.repo.ListForUser(ctx, userID, p)
}

// GetMine loads a complaint only if userID owns it (errNotFound otherwise, so
// a foreign id is indistinguishable from a missing one).
func (s *Service) GetMine(ctx context.Context, userID, id bson.ObjectID) (*Complaint, error) {
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.UserID != userID {
		return nil, errNotFound
	}
	return c, nil
}

// CustomerReply appends the owner's reply. A resolved complaint reopens; a
// closed one refuses (errClosed). Never notifies: customers are not told about
// their own messages.
func (s *Service) CustomerReply(ctx context.Context, userID, id bson.ObjectID, in ReplyInput) (*Complaint, error) {
	body, err := cleanText(in.Body, maxBodyLen, "message")
	if err != nil {
		return nil, err
	}
	photo, err := s.checkPhoto(in.PhotoURL)
	if err != nil {
		return nil, err
	}
	for range guardRetries {
		cur, err := s.GetMine(ctx, userID, id)
		if err != nil {
			return nil, err
		}
		if cur.Status == StatusClosed {
			return nil, errClosed
		}
		if len(cur.Messages) >= maxMessages {
			return nil, errThreadFull
		}
		now := s.now()
		msg := Message{ID: bson.NewObjectID(), Author: AuthorCustomer, Body: body, PhotoURL: photo, CreatedAt: now}
		set := bson.D{{Key: "lastCustomerMessageAt", Value: now}, {Key: "updatedAt", Value: now}}
		if cur.Status == StatusResolved {
			set = append(set, bson.E{Key: "status", Value: StatusOpen})
		}
		out, err := s.repo.AppendMessage(ctx, id, &userID, cur.Status, msg, set)
		if errors.Is(err, errConflict) {
			continue // status/thread moved underneath us: re-read and re-decide
		}
		return out, err
	}
	return nil, errConflict
}

// Actor is the admin performing a console action.
type Actor struct {
	ID    bson.ObjectID
	Label string
}

// AdminReply appends a staff reply (internal=false → the customer sees it and is
// notified once) or an internal note (never shown, never notified). A public
// reply on an open complaint moves it to in_progress; public replies on a
// closed complaint are refused (the customer could not answer), notes are not.
func (s *Service) AdminReply(ctx context.Context, actor Actor, id bson.ObjectID, in AdminReplyInput) (*Complaint, error) {
	body, err := cleanText(in.Body, maxBodyLen, "message")
	if err != nil {
		return nil, err
	}
	for range guardRetries {
		cur, err := s.repo.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if !in.Internal && cur.Status == StatusClosed {
			return nil, errClosed
		}
		if len(cur.Messages) >= maxMessages {
			return nil, errThreadFull
		}
		now := s.now()
		msg := Message{
			ID: bson.NewObjectID(), Author: AuthorAdmin, Body: body, Internal: in.Internal, CreatedAt: now,
		}
		if !actor.ID.IsZero() {
			aid := actor.ID
			msg.AdminID = &aid
		}
		msg.AdminLabel = actor.Label
		set := bson.D{{Key: "updatedAt", Value: now}}
		if !in.Internal {
			set = append(set, bson.E{Key: "lastAdminMessageAt", Value: now})
			if cur.Status == StatusOpen {
				set = append(set, bson.E{Key: "status", Value: StatusInProgress})
			}
		}
		out, err := s.repo.AppendMessage(ctx, id, nil, cur.Status, msg, set)
		if errors.Is(err, errConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !in.Internal {
			s.ntf.Notify(ctx, out.UserID, notification.Note{
				Kind:  notification.KindComplaintReply,
				Title: "Reply to your complaint " + Ref(out.Number),
				Body:  preview(body, 100),
				Data:  noteData(out),
			})
		}
		return out, nil
	}
	return nil, errConflict
}

// SetStatus moves a complaint to status. Moving to resolved / closed notifies
// the customer (once, only when the status actually changes). Returns the
// complaint before and after.
func (s *Service) SetStatus(ctx context.Context, id bson.ObjectID, status string) (before, after *Complaint, err error) {
	if !ValidStatus(status) {
		return nil, nil, invalid("status must be one of open, in_progress, resolved, closed")
	}
	for range guardRetries {
		cur, err := s.repo.Get(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if cur.Status == status {
			return cur, cur, nil
		}
		out, err := s.repo.UpdateGuarded(ctx, id, cur.Status,
			bson.D{{Key: "status", Value: status}, {Key: "updatedAt", Value: s.now()}}, nil)
		if errors.Is(err, errConflict) {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		if status == StatusResolved || status == StatusClosed {
			title := "Your complaint " + Ref(out.Number) + " was resolved"
			body := "If the problem is still there, reply to reopen it."
			if status == StatusClosed {
				title = "Your complaint " + Ref(out.Number) + " was closed"
				body = "Thank you for contacting support."
			}
			data := noteData(out)
			data["status"] = status
			s.ntf.Notify(ctx, out.UserID, notification.Note{
				Kind: notification.KindComplaintStatus, Title: title, Body: body, Data: data,
			})
		}
		return cur, out, nil
	}
	return nil, nil, errConflict
}

// SetAssignee assigns the complaint to an admin, or clears it (adminID nil).
// The assignee must be an active admin account.
func (s *Service) SetAssignee(ctx context.Context, id bson.ObjectID, adminID *bson.ObjectID) (before, after *Complaint, err error) {
	if adminID != nil {
		users, err := s.repo.Users(ctx, []bson.ObjectID{*adminID})
		if err != nil {
			return nil, nil, err
		}
		u, ok := users[*adminID]
		if !ok || u.Role != "admin" || u.Status == "suspended" || u.Status == "deleted" {
			return nil, nil, invalid("assignee must be an active admin")
		}
	}
	for range guardRetries {
		cur, err := s.repo.Get(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		set := bson.D{{Key: "updatedAt", Value: s.now()}}
		var unset bson.D
		if adminID != nil {
			set = append(set, bson.E{Key: "assigneeId", Value: *adminID})
		} else {
			unset = bson.D{{Key: "assigneeId", Value: ""}}
		}
		out, err := s.repo.UpdateGuarded(ctx, id, cur.Status, set, unset)
		if errors.Is(err, errConflict) {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		return cur, out, nil
	}
	return nil, nil, errConflict
}

func noteData(c *Complaint) map[string]string {
	return map[string]string{"complaintId": c.ID.Hex(), "number": strconv.Itoa(c.Number)}
}

// preview shortens s to at most n runes, adding an ellipsis when cut.
func preview(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:n])) + "…"
}
