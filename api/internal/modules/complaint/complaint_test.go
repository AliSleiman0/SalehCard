package complaint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/modules/role"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/blob"
	"github.com/AliSleiman0/salehcard/api/internal/platform/photoupload"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// ---- fakes -----------------------------------------------------------------

// fakeRepo is an in-memory Repository honoring the same guards as Mongo
// (expected status, owner, thread cap).
type fakeRepo struct {
	mu     sync.Mutex
	seq    int
	byID   map[bson.ObjectID]*Complaint
	users  map[bson.ObjectID]Contact
	orders map[bson.ObjectID]*OrderSummary
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		byID:   map[bson.ObjectID]*Complaint{},
		users:  map[bson.ObjectID]Contact{},
		orders: map[bson.ObjectID]*OrderSummary{},
	}
}

func clone(c *Complaint) *Complaint {
	cp := *c
	cp.Messages = append([]Message(nil), c.Messages...)
	return &cp
}

func (f *fakeRepo) NextNumber(context.Context) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	return numberBase + f.seq, nil
}
func (f *fakeRepo) Insert(_ context.Context, c *Complaint) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byID[c.ID] = clone(c)
	return nil
}
func (f *fakeRepo) Get(_ context.Context, id bson.ObjectID) (*Complaint, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.byID[id]
	if !ok {
		return nil, errNotFound
	}
	return clone(c), nil
}
func (f *fakeRepo) ListForUser(_ context.Context, userID bson.ObjectID, _ pagination.Params) ([]Complaint, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []Complaint{}
	for _, c := range f.byID {
		if c.UserID == userID {
			cp := clone(c)
			cp.Messages = nil
			out = append(out, *cp)
		}
	}
	return out, int64(len(out)), nil
}
func (f *fakeRepo) CountOpenForUser(_ context.Context, userID bson.ObjectID) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for _, c := range f.byID {
		if c.UserID == userID && (c.Status == StatusOpen || c.Status == StatusInProgress) {
			n++
		}
	}
	return n, nil
}
func applySet(c *Complaint, set bson.D) {
	for _, e := range set {
		switch e.Key {
		case "status":
			c.Status = e.Value.(string)
		case "updatedAt":
			c.UpdatedAt = e.Value.(time.Time)
		case "lastCustomerMessageAt":
			t := e.Value.(time.Time)
			c.LastCustomerMessageAt = &t
		case "lastAdminMessageAt":
			t := e.Value.(time.Time)
			c.LastAdminMessageAt = &t
		case "assigneeId":
			id := e.Value.(bson.ObjectID)
			c.AssigneeID = &id
		}
	}
}
func (f *fakeRepo) AppendMessage(_ context.Context, id bson.ObjectID, owner *bson.ObjectID, expect string, msg Message, set bson.D) (*Complaint, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.byID[id]
	if !ok || c.Status != expect || len(c.Messages) >= maxMessages || (owner != nil && c.UserID != *owner) {
		return nil, errConflict
	}
	c.Messages = append(c.Messages, msg)
	applySet(c, set)
	return clone(c), nil
}
func (f *fakeRepo) UpdateGuarded(_ context.Context, id bson.ObjectID, expect string, set, unset bson.D) (*Complaint, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.byID[id]
	if !ok || c.Status != expect {
		return nil, errConflict
	}
	applySet(c, set)
	for _, e := range unset {
		if e.Key == "assigneeId" {
			c.AssigneeID = nil
		}
	}
	return clone(c), nil
}
func (f *fakeRepo) AdminList(context.Context, AdminFilter, pagination.Params) ([]Complaint, int64, error) {
	return nil, 0, nil
}
func (f *fakeRepo) Counts(context.Context) (Counts, int64, error) { return Counts{}, 0, nil }
func (f *fakeRepo) Users(_ context.Context, ids []bson.ObjectID) (map[bson.ObjectID]Contact, error) {
	out := map[bson.ObjectID]Contact{}
	for _, id := range ids {
		if u, ok := f.users[id]; ok {
			out[id] = u
		}
	}
	return out, nil
}
func (f *fakeRepo) Order(_ context.Context, id bson.ObjectID) (*OrderSummary, error) {
	o, ok := f.orders[id]
	if !ok {
		return nil, errNotFound
	}
	return o, nil
}

// fakeNotifier counts Notify calls.
type fakeNotifier struct {
	mu    sync.Mutex
	notes []notification.Note
	users []bson.ObjectID
}

func (n *fakeNotifier) Notify(_ context.Context, userID bson.ObjectID, note notification.Note) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.notes = append(n.notes, note)
	n.users = append(n.users, userID)
}
func (n *fakeNotifier) NotifyMany(context.Context, []bson.ObjectID, notification.Note) int { return 0 }

type nopRecorder struct{}

func (nopRecorder) Record(context.Context, audit.Entry) {}

const testBase = "http://localhost:8090"

func newTestService(t *testing.T) (*Service, *fakeRepo, *fakeNotifier) {
	t.Helper()
	repo := newFakeRepo()
	ntf := &fakeNotifier{}
	store := blob.NewLocal(blob.LocalConfig{Dir: t.TempDir(), PublicBase: testBase})
	return NewService(repo, ntf, photoupload.NewChecker(store, PhotoPrefix)), repo, ntf
}

func mustCreate(t *testing.T, s *Service, user bson.ObjectID) *Complaint {
	t.Helper()
	c, err := s.Create(context.Background(), user, CreateInput{Category: CategoryOther, Subject: "Help", Message: "Something broke"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return c
}

// ---- service rules -----------------------------------------------------------

func TestCreate_Validation(t *testing.T) {
	s, repo, _ := newTestService(t)
	user := bson.NewObjectID()
	ctx := context.Background()

	bad := []CreateInput{
		{Category: "nope", Subject: "s", Message: "m"},
		{Category: CategoryOther, Subject: "", Message: "m"},
		{Category: CategoryOther, Subject: strings.Repeat("x", maxSubjectLen+1), Message: "m"},
		{Category: CategoryOther, Subject: "s", Message: strings.Repeat("x", maxBodyLen+1)},
	}
	for i, in := range bad {
		var inv *invalidError
		if _, err := s.Create(ctx, user, in); !errors.As(err, &inv) {
			t.Errorf("case %d: want invalid, got %v", i, err)
		}
	}

	c := mustCreate(t, s, user)
	if c.Number != numberBase+1 || c.Status != StatusOpen || len(c.Messages) != 1 || c.Messages[0].Author != AuthorCustomer {
		t.Errorf("unexpected complaint: %+v", c)
	}
	if len(repo.byID) != 1 {
		t.Errorf("want 1 stored, got %d", len(repo.byID))
	}
}

func TestCreate_RejectsForeignPhotoURL(t *testing.T) {
	s, _, _ := newTestService(t)
	user := bson.NewObjectID()
	for _, u := range []string{
		"https://attacker.example/uploads/complaints/x.jpg", // tracking pixel, same path shape
		testBase + "/uploads/kyc/x.jpg",                     // other keyspace
		testBase + "/uploads/complaints/../kyc/x.jpg",       // traversal
		"not a url",
	} {
		_, err := s.Create(context.Background(), user, CreateInput{
			Category: CategoryOther, Subject: "s", Message: "m", PhotoURL: u,
		})
		var inv *invalidError
		if !errors.As(err, &inv) {
			t.Errorf("photo %q: want invalid, got %v", u, err)
		}
	}
	ok := testBase + "/uploads/complaints/" + bson.NewObjectID().Hex() + ".jpg"
	if _, err := s.Create(context.Background(), user, CreateInput{
		Category: CategoryOther, Subject: "s", Message: "m", PhotoURL: ok,
	}); err != nil {
		t.Errorf("our own upload must pass: %v", err)
	}
}

func TestCreate_OrderMustBelongToUser(t *testing.T) {
	s, repo, _ := newTestService(t)
	me, other := bson.NewObjectID(), bson.NewObjectID()
	mine, theirs := bson.NewObjectID(), bson.NewObjectID()
	repo.orders[mine] = &OrderSummary{ID: mine, UserID: me}
	repo.orders[theirs] = &OrderSummary{ID: theirs, UserID: other}
	ctx := context.Background()

	for _, oid := range []string{theirs.Hex(), bson.NewObjectID().Hex(), "garbage"} {
		_, err := s.Create(ctx, me, CreateInput{Category: CategoryOrder, Subject: "s", Message: "m", OrderID: oid})
		var inv *invalidError
		if !errors.As(err, &inv) {
			t.Errorf("order %s: want invalid, got %v", oid, err)
		}
	}
	c, err := s.Create(ctx, me, CreateInput{Category: CategoryOrder, Subject: "s", Message: "m", OrderID: mine.Hex()})
	if err != nil || c.OrderID == nil || *c.OrderID != mine {
		t.Fatalf("own order must link: %v %+v", err, c)
	}
}

func TestCreate_OpenCap(t *testing.T) {
	s, _, _ := newTestService(t)
	user := bson.NewObjectID()
	var last *Complaint
	for range maxOpenPerUser {
		last = mustCreate(t, s, user)
	}
	if _, err := s.Create(context.Background(), user, CreateInput{Category: CategoryOther, Subject: "s", Message: "m"}); !errors.Is(err, errTooMany) {
		t.Fatalf("want errTooMany, got %v", err)
	}
	// Resolving one frees a slot.
	if _, _, err := s.SetStatus(context.Background(), last.ID, StatusResolved); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(context.Background(), user, CreateInput{Category: CategoryOther, Subject: "s", Message: "m"}); err != nil {
		t.Fatalf("slot should be free: %v", err)
	}
}

func TestGetMine_OtherUserIsNotFound(t *testing.T) {
	s, _, _ := newTestService(t)
	owner, stranger := bson.NewObjectID(), bson.NewObjectID()
	c := mustCreate(t, s, owner)
	if _, err := s.GetMine(context.Background(), stranger, c.ID); !errors.Is(err, errNotFound) {
		t.Fatalf("want errNotFound, got %v", err)
	}
	if _, err := s.CustomerReply(context.Background(), stranger, c.ID, ReplyInput{Body: "hi"}); !errors.Is(err, errNotFound) {
		t.Fatalf("reply by stranger: want errNotFound, got %v", err)
	}
}

func TestCustomerReply_ReopensResolvedAndRefusesClosed(t *testing.T) {
	s, _, ntf := newTestService(t)
	user := bson.NewObjectID()
	ctx := context.Background()
	c := mustCreate(t, s, user)

	if _, _, err := s.SetStatus(ctx, c.ID, StatusResolved); err != nil {
		t.Fatal(err)
	}
	notesBefore := len(ntf.notes)
	out, err := s.CustomerReply(ctx, user, c.ID, ReplyInput{Body: "still broken"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != StatusOpen {
		t.Errorf("customer reply must reopen a resolved complaint, got %s", out.Status)
	}
	if len(ntf.notes) != notesBefore {
		t.Error("a customer must never be notified about their own message")
	}

	if _, _, err := s.SetStatus(ctx, c.ID, StatusClosed); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CustomerReply(ctx, user, c.ID, ReplyInput{Body: "hello?"}); !errors.Is(err, errClosed) {
		t.Fatalf("want errClosed, got %v", err)
	}
}

func TestAdminReply_NotifiesOnceAndNotesNever(t *testing.T) {
	s, _, ntf := newTestService(t)
	user := bson.NewObjectID()
	ctx := context.Background()
	c := mustCreate(t, s, user)
	admin := Actor{ID: bson.NewObjectID(), Label: "staff@x"}

	out, err := s.AdminReply(ctx, admin, c.ID, AdminReplyInput{Body: "checking with the supplier", Internal: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(ntf.notes) != 0 {
		t.Fatalf("internal note must not notify, got %d", len(ntf.notes))
	}
	if out.Status != StatusOpen || out.LastAdminMessageAt != nil {
		t.Error("an internal note must not change status or count as a reply")
	}

	out, err = s.AdminReply(ctx, admin, c.ID, AdminReplyInput{Body: strings.Repeat("Thanks for waiting. ", 20)})
	if err != nil {
		t.Fatal(err)
	}
	if len(ntf.notes) != 1 {
		t.Fatalf("admin reply must notify exactly once, got %d", len(ntf.notes))
	}
	n := ntf.notes[0]
	if n.Kind != notification.KindComplaintReply || ntf.users[0] != user ||
		n.Data["complaintId"] != c.ID.Hex() || n.Data["number"] != "1001" ||
		n.Title != "Reply to your complaint C-1001" || len([]rune(n.Body)) > 101 {
		t.Errorf("unexpected note: %+v", n)
	}
	if out.Status != StatusInProgress {
		t.Errorf("a public reply on an open complaint moves it to in_progress, got %s", out.Status)
	}
	if out.AwaitingReply() {
		t.Error("after a staff reply the complaint is not awaiting reply")
	}
}

func TestSetStatus_NotifiesOnResolveAndCloseOnly(t *testing.T) {
	s, _, ntf := newTestService(t)
	ctx := context.Background()
	c := mustCreate(t, s, bson.NewObjectID())

	for _, st := range []string{StatusInProgress, StatusResolved, StatusResolved, StatusClosed} {
		if _, _, err := s.SetStatus(ctx, c.ID, st); err != nil {
			t.Fatal(err)
		}
	}
	// in_progress: none; resolved: 1; resolved again (no change): none; closed: 1.
	if len(ntf.notes) != 2 {
		t.Fatalf("want 2 notifications, got %d", len(ntf.notes))
	}
	for _, n := range ntf.notes {
		if n.Kind != notification.KindComplaintStatus {
			t.Errorf("kind %s", n.Kind)
		}
	}
	if _, _, err := s.SetStatus(ctx, c.ID, "bogus"); err == nil {
		t.Error("invalid status must be refused")
	}
}

func TestSetAssignee_MustBeActiveAdmin(t *testing.T) {
	s, repo, _ := newTestService(t)
	ctx := context.Background()
	c := mustCreate(t, s, bson.NewObjectID())
	admin, customer := bson.NewObjectID(), bson.NewObjectID()
	repo.users[admin] = Contact{ID: admin, Role: "admin", Status: "active"}
	repo.users[customer] = Contact{ID: customer, Role: "customer"}

	if _, _, err := s.SetAssignee(ctx, c.ID, &customer); err == nil {
		t.Error("a customer cannot be an assignee")
	}
	_, after, err := s.SetAssignee(ctx, c.ID, &admin)
	if err != nil || after.AssigneeID == nil || *after.AssigneeID != admin {
		t.Fatalf("assign: %v %+v", err, after)
	}
	_, after, err = s.SetAssignee(ctx, c.ID, nil)
	if err != nil || after.AssigneeID != nil {
		t.Fatalf("unassign: %v %+v", err, after)
	}
}

func TestThreadCap(t *testing.T) {
	s, repo, _ := newTestService(t)
	user := bson.NewObjectID()
	c := mustCreate(t, s, user)
	for i := len(repo.byID[c.ID].Messages); i < maxMessages; i++ {
		repo.byID[c.ID].Messages = append(repo.byID[c.ID].Messages, Message{ID: bson.NewObjectID()})
	}
	if _, err := s.CustomerReply(context.Background(), user, c.ID, ReplyInput{Body: "x"}); !errors.Is(err, errThreadFull) {
		t.Fatalf("want errThreadFull, got %v", err)
	}
}

// ---- HTTP: customer responses never carry internal notes ----------------------

func withClaims(c *auth.Claims) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.ContextWithClaims(r.Context(), c)))
		})
	}
}

func customerRouter(s *Service, user bson.ObjectID) http.Handler {
	r := chi.NewRouter()
	r.Use(withClaims(&auth.Claims{UserID: user.Hex(), Role: "customer"}))
	pass := func(next http.Handler) http.Handler { return next }
	mountCustomer(r, &customerHandler{svc: s}, pass, pass, func(w http.ResponseWriter, _ *http.Request) {})
	return r
}

func do(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHTTP_CustomerNeverSeesInternalNotes(t *testing.T) {
	s, _, _ := newTestService(t)
	user := bson.NewObjectID()
	c := mustCreate(t, s, user)
	const secret = "SECRET-INTERNAL-NOTE"
	admin := Actor{ID: bson.NewObjectID(), Label: "staff@salehcard.com"}
	if _, err := s.AdminReply(context.Background(), admin, c.ID, AdminReplyInput{Body: secret, Internal: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AdminReply(context.Background(), admin, c.ID, AdminReplyInput{Body: "public answer"}); err != nil {
		t.Fatal(err)
	}

	h := customerRouter(s, user)
	for _, call := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/" + c.ID.Hex(), nil},
		{http.MethodGet, "/", nil},
		{http.MethodPost, "/" + c.ID.Hex() + "/messages", ReplyInput{Body: "thanks"}},
	} {
		rec := do(t, h, call.method, call.path, call.body)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s: %d %s", call.method, call.path, rec.Code, rec.Body.String())
		}
		got := rec.Body.String()
		if strings.Contains(got, secret) || strings.Contains(got, "staff@salehcard.com") ||
			strings.Contains(got, `"internal"`) || strings.Contains(got, admin.ID.Hex()) {
			t.Errorf("%s %s leaked internal data: %s", call.method, call.path, got)
		}
		if call.method == http.MethodGet && call.path != "/" && !strings.Contains(got, "public answer") {
			t.Errorf("public reply missing: %s", got)
		}
	}
}

func TestHTTP_StatusCodes(t *testing.T) {
	s, _, _ := newTestService(t)
	owner, stranger := bson.NewObjectID(), bson.NewObjectID()
	c := mustCreate(t, s, owner)

	// Another user's complaint: 404, never 403.
	if rec := do(t, customerRouter(s, stranger), http.MethodGet, "/"+c.ID.Hex(), nil); rec.Code != http.StatusNotFound {
		t.Errorf("stranger GET: want 404, got %d", rec.Code)
	}
	// Closed → 409 CLOSED.
	if _, _, err := s.SetStatus(context.Background(), c.ID, StatusClosed); err != nil {
		t.Fatal(err)
	}
	rec := do(t, customerRouter(s, owner), http.MethodPost, "/"+c.ID.Hex()+"/messages", ReplyInput{Body: "hi"})
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "CLOSED") {
		t.Errorf("closed reply: want 409 CLOSED, got %d %s", rec.Code, rec.Body.String())
	}
	// Open cap → 409 TOO_MANY_OPEN.
	for range maxOpenPerUser {
		mustCreate(t, s, owner)
	}
	rec = do(t, customerRouter(s, owner), http.MethodPost, "/", CreateInput{Category: CategoryOther, Subject: "s", Message: "m"})
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "TOO_MANY_OPEN") {
		t.Errorf("cap: want 409 TOO_MANY_OPEN, got %d %s", rec.Code, rec.Body.String())
	}
}

// ---- RBAC ---------------------------------------------------------------------

func TestRBAC_ComplaintsDomain(t *testing.T) {
	if !role.ValidPermission("complaints.view") || !role.ValidPermission("complaints.manage") {
		t.Fatal("complaints must be in the RBAC permission catalog")
	}
	s, repo, _ := newTestService(t)
	c := mustCreate(t, s, bson.NewObjectID())
	h := &adminHandler{svc: s, repo: repo, rec: nopRecorder{}}

	router := func(perms ...string) http.Handler {
		r := chi.NewRouter()
		r.Use(withClaims(&auth.Claims{UserID: bson.NewObjectID().Hex(), Role: "admin", Perms: perms}))
		r.Group(func(g chi.Router) {
			g.Use(auth.RequireDomain("complaints"))
			mountAdmin(g, h)
		})
		return r
	}

	// A custom role without complaints.* is refused on reads and writes.
	noPerm := router("orders.view", "orders.manage")
	if rec := do(t, noPerm, http.MethodGet, "/complaints/"+c.ID.Hex(), nil); rec.Code != http.StatusForbidden {
		t.Errorf("GET without complaints.view: want 403, got %d", rec.Code)
	}
	if rec := do(t, noPerm, http.MethodPost, "/complaints/"+c.ID.Hex()+"/messages", AdminReplyInput{Body: "x"}); rec.Code != http.StatusForbidden {
		t.Errorf("POST without complaints.manage: want 403, got %d", rec.Code)
	}
	// view-only reads but cannot reply.
	viewOnly := router("complaints.view")
	if rec := do(t, viewOnly, http.MethodGet, "/complaints/"+c.ID.Hex(), nil); rec.Code != http.StatusOK {
		t.Errorf("GET with complaints.view: want 200, got %d", rec.Code)
	}
	if rec := do(t, viewOnly, http.MethodPut, "/complaints/"+c.ID.Hex()+"/status", map[string]string{"status": "resolved"}); rec.Code != http.StatusForbidden {
		t.Errorf("PUT with view only: want 403, got %d", rec.Code)
	}
	// manage can reply; the admin detail includes internal notes.
	manage := router("complaints.manage", "complaints.view")
	if rec := do(t, manage, http.MethodPost, "/complaints/"+c.ID.Hex()+"/messages", AdminReplyInput{Body: "note!", Internal: true}); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "note!") {
		t.Errorf("manage reply: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPreview(t *testing.T) {
	if got := preview("  hello \n world  ", 100); got != "hello world" {
		t.Errorf("got %q", got)
	}
	if got := preview(strings.Repeat("ab ", 60), 10); len([]rune(got)) > 11 || !strings.HasSuffix(got, "…") {
		t.Errorf("got %q", got)
	}
}

func TestRef(t *testing.T) {
	if Ref(1042) != "C-1042" {
		t.Error(Ref(1042))
	}
}
