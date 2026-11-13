package complaint

import (
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// adminHandler serves /api/admin/complaints (wrapped in the "complaints" RBAC
// domain: GET = complaints.view, mutations = complaints.manage).
type adminHandler struct {
	svc  *Service
	repo Repository
	rec  audit.Recorder
}

// AdminRow is one line of the console queue.
type AdminRow struct {
	ID             bson.ObjectID  `json:"id"`
	Number         int            `json:"number"`
	Ref            string         `json:"ref"`
	Category       string         `json:"category"`
	Subject        string         `json:"subject"`
	Status         string         `json:"status"`
	AssigneeID     *bson.ObjectID `json:"assigneeId,omitempty"`
	OrderID        *bson.ObjectID `json:"orderId,omitempty"`
	Customer       Contact        `json:"customer"`
	AwaitingReply  bool           `json:"awaitingReply"`
	LastActivityAt string         `json:"lastActivityAt"`
	CreatedAt      string         `json:"createdAt"`
}

// AdminDetail is the full console view: the whole thread (internal notes
// included), the customer's contact details, and the linked order.
type AdminDetail struct {
	*Complaint
	Ref           string        `json:"ref"`
	AwaitingReply bool          `json:"awaitingReply"`
	Customer      Contact       `json:"customer"`
	Order         *OrderSummary `json:"order,omitempty"`
}

const timeLayout = "2006-01-02T15:04:05.000Z07:00"

func (h *adminHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := AdminFilter{
		Status:   strings.TrimSpace(q.Get("status")),
		Category: strings.TrimSpace(q.Get("category")),
		Assignee: strings.TrimSpace(q.Get("assignee")),
		Search:   strings.TrimSpace(q.Get("search")),
	}
	if f.Status != "" && !ValidStatus(f.Status) {
		response.BadRequest(w, "invalid status filter")
		return
	}
	if f.Category != "" && !ValidCategory(f.Category) {
		response.BadRequest(w, "invalid category filter")
		return
	}
	if f.Assignee == "me" {
		if id, ok := auth.UserIDFromContext(r.Context()); ok {
			f.Assignee = id.Hex()
		}
	}
	p := pagination.ParseParams(r)
	rows, total, err := h.repo.AdminList(r.Context(), f, p)
	if err != nil {
		response.InternalError(w)
		return
	}
	ids := make([]bson.ObjectID, 0, len(rows))
	for _, c := range rows {
		ids = append(ids, c.UserID)
	}
	users, err := h.repo.Users(r.Context(), ids)
	if err != nil {
		response.InternalError(w)
		return
	}
	out := make([]AdminRow, 0, len(rows))
	for i := range rows {
		c := &rows[i]
		cust, ok := users[c.UserID]
		if !ok {
			cust = Contact{ID: c.UserID}
		}
		out = append(out, AdminRow{
			ID: c.ID, Number: c.Number, Ref: Ref(c.Number), Category: c.Category,
			Subject: c.Subject, Status: c.Status, AssigneeID: c.AssigneeID, OrderID: c.OrderID,
			Customer: cust, AwaitingReply: c.AwaitingReply(),
			LastActivityAt: c.UpdatedAt.Format(timeLayout), CreatedAt: c.CreatedAt.Format(timeLayout),
		})
	}
	response.OKWithMeta(w, out, pagination.CalcMeta(p, total))
}

// counts is the cheap tally behind the status tabs and the nav badge.
func (h *adminHandler) counts(w http.ResponseWriter, r *http.Request) {
	c, awaiting, err := h.repo.Counts(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]any{"byStatus": c, "awaitingReply": awaiting})
}

// assignees lists the admins a complaint can be assigned to. It lives inside
// the complaints domain so a support role needs no users.view to use it.
func (h *adminHandler) assignees(w http.ResponseWriter, r *http.Request) {
	out, err := h.repo.Admins(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, out)
}

func (h *adminHandler) detail(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	c, err := h.repo.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	h.writeDetail(w, r, c)
}

func (h *adminHandler) writeDetail(w http.ResponseWriter, r *http.Request, c *Complaint) {
	d := AdminDetail{Complaint: c, Ref: Ref(c.Number), AwaitingReply: c.AwaitingReply(), Customer: Contact{ID: c.UserID}}
	if users, err := h.repo.Users(r.Context(), []bson.ObjectID{c.UserID}); err == nil {
		if u, ok := users[c.UserID]; ok {
			d.Customer = u
		}
	}
	if c.OrderID != nil {
		if o, err := h.repo.Order(r.Context(), *c.OrderID); err == nil {
			d.Order = o
		}
	}
	response.OK(w, d)
}

func actorOf(r *http.Request) Actor {
	c, _ := auth.ClaimsFromContext(r.Context())
	a := Actor{Label: auth.ActorLabel(c)}
	if id, ok := auth.UserIDFromContext(r.Context()); ok {
		a.ID = id
	}
	return a
}

func (h *adminHandler) reply(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in AdminReplyInput
	if !decodeJSON(w, r, &in) {
		return
	}
	c, err := h.svc.AdminReply(r.Context(), actorOf(r), id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	action := audit.ActionComplaintReply
	if in.Internal {
		action = audit.ActionComplaintNote
	}
	h.audit(r, action, c, map[string]any{"length": len([]rune(strings.TrimSpace(in.Body)))})
	h.writeDetail(w, r, c)
}

func (h *adminHandler) setStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	before, after, err := h.svc.SetStatus(r.Context(), id, strings.TrimSpace(body.Status))
	if err != nil {
		writeError(w, err)
		return
	}
	if before.Status != after.Status {
		h.audit(r, audit.ActionComplaintStatus, after, map[string]any{"from": before.Status, "to": after.Status})
	}
	h.writeDetail(w, r, after)
}

func (h *adminHandler) setAssignee(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		AdminID *string `json:"adminId"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	var adminID *bson.ObjectID
	if body.AdminID != nil && strings.TrimSpace(*body.AdminID) != "" {
		aid, err := bson.ObjectIDFromHex(strings.TrimSpace(*body.AdminID))
		if err != nil {
			response.BadRequest(w, "adminId is not a valid id")
			return
		}
		adminID = &aid
	}
	before, after, err := h.svc.SetAssignee(r.Context(), id, adminID)
	if err != nil {
		writeError(w, err)
		return
	}
	hexOrEmpty := func(p *bson.ObjectID) string {
		if p == nil {
			return ""
		}
		return p.Hex()
	}
	h.audit(r, audit.ActionComplaintAssign, after, map[string]any{
		"from": hexOrEmpty(before.AssigneeID), "to": hexOrEmpty(after.AssigneeID),
	})
	h.writeDetail(w, r, after)
}

func (h *adminHandler) audit(r *http.Request, action string, c *Complaint, extra map[string]any) {
	summary := map[string]any{"ref": Ref(c.Number), "userId": c.UserID.Hex()}
	for k, v := range extra {
		summary[k] = v
	}
	h.rec.Record(r.Context(), audit.Entry{Action: action, TargetType: "complaint", TargetID: c.ID.Hex(), Summary: summary})
}
