package notification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/push"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// fakeRepo is an in-memory Repository for notifier and handler tests.
type fakeRepo struct {
	inserted []*Notification
	tokens   map[string]bson.ObjectID // token -> user
	unread   int64
	marked   int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{tokens: map[string]bson.ObjectID{}}
}

func (f *fakeRepo) Insert(_ context.Context, n *Notification) error {
	f.inserted = append(f.inserted, n)
	return nil
}

func (f *fakeRepo) ListByUser(_ context.Context, userID bson.ObjectID, _ pagination.Params) ([]*Notification, int64, error) {
	out := []*Notification{}
	for _, n := range f.inserted {
		if n.UserID == userID {
			out = append(out, n)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakeRepo) CountUnread(context.Context, bson.ObjectID) (int64, error) {
	return f.unread, nil
}

func (f *fakeRepo) MarkAllRead(context.Context, bson.ObjectID) (int64, error) {
	f.marked = f.unread
	f.unread = 0
	return f.marked, nil
}

func (f *fakeRepo) UpsertToken(_ context.Context, userID bson.ObjectID, token, _ string) error {
	f.tokens[token] = userID
	return nil
}

func (f *fakeRepo) DeleteToken(_ context.Context, userID bson.ObjectID, token string) error {
	if f.tokens[token] == userID {
		delete(f.tokens, token)
	}
	return nil
}

func (f *fakeRepo) TokensForUser(_ context.Context, userID bson.ObjectID) ([]string, error) {
	out := []string{}
	for token, uid := range f.tokens {
		if uid == userID {
			out = append(out, token)
		}
	}
	return out, nil
}

func (f *fakeRepo) DeleteTokenValue(_ context.Context, token string) error {
	delete(f.tokens, token)
	return nil
}

// fakeSender records pushes; tokens listed in stale return ErrUnregistered.
type fakeSender struct {
	sent  []push.Message
	stale map[string]bool
}

func (f *fakeSender) Send(_ context.Context, msg push.Message) error {
	f.sent = append(f.sent, msg)
	if f.stale[msg.Token] {
		return push.ErrUnregistered
	}
	return nil
}

func TestNotifier_Notify_InsertsRowAndFansOut(t *testing.T) {
	repo := newFakeRepo()
	sender := &fakeSender{}
	userID := bson.NewObjectID()
	repo.tokens["tok-a"] = userID
	repo.tokens["tok-b"] = userID
	repo.tokens["tok-other"] = bson.NewObjectID() // someone else's device

	nt := &notifier{repo: repo, sender: sender}
	nt.Notify(context.Background(), userID, Note{
		Kind:  KindOrderCompleted,
		Title: "Order completed",
		Body:  "Your order was delivered — $25.00.",
		Data:  map[string]string{"orderId": "abc"},
	})

	// The inbox row is written synchronously.
	require.Len(t, repo.inserted, 1)
	row := repo.inserted[0]
	assert.Equal(t, userID, row.UserID)
	assert.Equal(t, KindOrderCompleted, row.Kind)
	assert.Nil(t, row.ReadAt) // new rows are unread
	assert.False(t, row.CreatedAt.IsZero())

	// The push fan-out runs in the background — poll briefly for both sends.
	require.Eventually(t, func() bool { return len(sender.sent) == 2 }, 2*time.Second, 10*time.Millisecond)
	gotTokens := []string{sender.sent[0].Token, sender.sent[1].Token}
	assert.ElementsMatch(t, []string{"tok-a", "tok-b"}, gotTokens)
	assert.Equal(t, "Order completed", sender.sent[0].Title)
	assert.Equal(t, "abc", sender.sent[0].Data["orderId"])
}

func TestNotifier_FanOut_PrunesUnregisteredTokens(t *testing.T) {
	repo := newFakeRepo()
	userID := bson.NewObjectID()
	repo.tokens["tok-live"] = userID
	repo.tokens["tok-stale"] = userID
	sender := &fakeSender{stale: map[string]bool{"tok-stale": true}}

	nt := &notifier{repo: repo, sender: sender}
	nt.fanOut(context.Background(), userID, Note{Kind: KindTopUpApproved, Title: "t", Body: "b"})

	assert.Len(t, sender.sent, 2)
	assert.NotContains(t, repo.tokens, "tok-stale") // pruned
	assert.Contains(t, repo.tokens, "tok-live")
}

const testSecret = "test-secret"

// authedRequest builds a request bearing a real JWT for userID; serve routes it
// through the actual AuthRequired middleware so the claims land in the context.
func authedRequest(t *testing.T, method, target, body string, userID bson.ObjectID) *http.Request {
	t.Helper()
	if body == "" {
		body = "{}"
	}
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	token, err := auth.IssueAccessToken(testSecret, auth.Claims{UserID: userID.Hex()}, time.Minute)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// serve runs req through AuthRequired + h and returns the recorder.
func serve(h http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	auth.AuthRequired(testSecret)(h).ServeHTTP(rec, req)
	return rec
}

func TestHandler_UnreadCountAndReadAll(t *testing.T) {
	repo := newFakeRepo()
	repo.unread = 3
	h := NewHandler(repo)
	userID := bson.NewObjectID()

	rec := serve(h.UnreadCount, authedRequest(t, http.MethodGet, "/api/v1/notifications/unread-count", "", userID))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"count":3`)

	rec = serve(h.ReadAll, authedRequest(t, http.MethodPost, "/api/v1/notifications/read-all", "", userID))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"updated":3`)
	assert.EqualValues(t, 0, repo.unread)
}

func TestHandler_RegisterDevice(t *testing.T) {
	repo := newFakeRepo()
	h := NewHandler(repo)
	userID := bson.NewObjectID()

	t.Run("registers with default platform", func(t *testing.T) {
		rec := serve(h.RegisterDevice, authedRequest(t, http.MethodPost, "/api/v1/notifications/devices", `{"token":"tok-1"}`, userID))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, userID, repo.tokens["tok-1"])
	})

	t.Run("blank token rejected", func(t *testing.T) {
		rec := serve(h.RegisterDevice, authedRequest(t, http.MethodPost, "/api/v1/notifications/devices", `{"token":"  "}`, userID))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unregister removes own token", func(t *testing.T) {
		rec := serve(h.UnregisterDevice, authedRequest(t, http.MethodDelete, "/api/v1/notifications/devices", `{"token":"tok-1"}`, userID))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.NotContains(t, repo.tokens, "tok-1")
	})

	t.Run("unauthenticated rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/devices", strings.NewReader(`{"token":"x"}`))
		rec := serve(h.RegisterDevice, req) // no bearer → middleware 401s
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
