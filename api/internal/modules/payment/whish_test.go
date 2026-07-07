package payment

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/platform/whish"
)

// fakeWhish is a scriptable whish.Provider for callback/settlement tests.
type fakeWhish struct {
	initiateErr error
	status      whish.CollectStatus
	statusErr   error
	payerPhone  string
	initiated   int
	statusCalls int
}

func (f *fakeWhish) Initiate(_ context.Context, in whish.InitiateInput) (whish.InitiateResult, error) {
	f.initiated++
	if f.initiateErr != nil {
		return whish.InitiateResult{}, f.initiateErr
	}
	return whish.InitiateResult{
		RedirectURL: "https://pay.whish/" + strconv.FormatInt(in.ExternalID, 10),
		ProviderRef: "ref-" + strconv.FormatInt(in.ExternalID, 10),
	}, nil
}

func (f *fakeWhish) GetStatus(_ context.Context, _ whish.StatusQuery) (whish.StatusResult, error) {
	f.statusCalls++
	if f.statusErr != nil {
		return whish.StatusResult{}, f.statusErr
	}
	return whish.StatusResult{Status: f.status, PayerPhone: f.payerPhone}, nil
}

func (e *testEnv) enableWhish(w whish.Provider) { e.svc.SetWhish(w, NewTokens("test-hmac-secret")) }

func TestTokens_SignVerify(t *testing.T) {
	tk := NewTokens("s3cr3t")
	const id int64 = 1234567890123
	sig := tk.SignExternalID(id)
	if !tk.VerifyExternalID(id, sig) {
		t.Fatal("valid token failed verification")
	}
	if tk.VerifyExternalID(id, sig+"00") {
		t.Fatal("tampered token verified")
	}
	if tk.VerifyExternalID(id+1, sig) {
		t.Fatal("token verified for a different external id")
	}
	if tk.VerifyExternalID(id, "not-hex-zz") {
		t.Fatal("non-hex token verified")
	}
	if NewTokens("other").VerifyExternalID(id, sig) {
		t.Fatal("token verified under a different secret")
	}
}

func TestCreateWhishTopUpIntent_Initiates(t *testing.T) {
	e := newTestEnv(t, Config{})
	fw := &fakeWhish{status: whish.CollectStatusSuccess}
	e.enableWhish(fw)

	in, err := e.svc.CreateWhishTopUpIntent(context.Background(), bson.NewObjectID(), 25, "idem-1")
	if err != nil {
		t.Fatalf("CreateWhishTopUpIntent: %v", err)
	}
	if fw.initiated != 1 {
		t.Fatalf("initiated = %d, want 1", fw.initiated)
	}
	if in.Provider != ProviderWhish || in.Status != StatusPending {
		t.Fatalf("provider/status = %s/%s", in.Provider, in.Status)
	}
	if in.RedirectURL == "" || in.ExternalID == 0 {
		t.Fatalf("redirectURL/externalID not set: %q / %d", in.RedirectURL, in.ExternalID)
	}
}

func TestCreateWhishTopUpIntent_InitiateFailureMarksFailed(t *testing.T) {
	e := newTestEnv(t, Config{})
	fw := &fakeWhish{initiateErr: errors.New("gateway down")}
	e.enableWhish(fw)

	_, err := e.svc.CreateWhishTopUpIntent(context.Background(), bson.NewObjectID(), 25, "idem-x")
	if err == nil {
		t.Fatal("expected error when Initiate fails")
	}
	// The intent was inserted then marked failed (so its idempotency key isn't
	// permanently blocked as pending).
	var found *Intent
	for _, v := range e.store.intents {
		found = v
	}
	if found == nil || found.Status != StatusFailed {
		t.Fatalf("intent not marked failed: %+v", found)
	}
}

func TestHandleCallback_SuccessCreditsOnce(t *testing.T) {
	e := newTestEnv(t, Config{})
	fw := &fakeWhish{status: whish.CollectStatusSuccess, payerPhone: "96170902894"}
	e.enableWhish(fw)

	userID := bson.NewObjectID()
	in, err := e.svc.CreateWhishTopUpIntent(context.Background(), userID, 25, "idem-2")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	token := e.svc.tokens.SignExternalID(in.ExternalID)
	eid := strconv.FormatInt(in.ExternalID, 10)

	if err := e.svc.HandleCallback(context.Background(), eid, token); err != nil {
		t.Fatalf("HandleCallback: %v", err)
	}
	got := e.store.get(t, in.ID)
	if got.Status != StatusConfirmed {
		t.Fatalf("status = %s, want confirmed", got.Status)
	}
	if len(e.crediter.calls) != 1 {
		t.Fatalf("credits = %d, want 1", len(e.crediter.calls))
	}
	if c := e.crediter.calls[0]; c.Method != "whish" || c.Ref != in.ID.Hex() || c.Amount != 25 {
		t.Fatalf("bad credit: %+v", c)
	}

	// Re-delivery of the callback is a safe no-op (still exactly one credit).
	if err := e.svc.HandleCallback(context.Background(), eid, token); err != nil {
		t.Fatalf("HandleCallback replay: %v", err)
	}
	if len(e.crediter.calls) != 1 {
		t.Fatalf("credits after replay = %d, want 1", len(e.crediter.calls))
	}
}

func TestHandleCallback_BadToken(t *testing.T) {
	e := newTestEnv(t, Config{})
	e.enableWhish(&fakeWhish{status: whish.CollectStatusSuccess})
	err := e.svc.HandleCallback(context.Background(), "12345", "deadbeef")
	if !errors.Is(err, ErrInvalidCallback) {
		t.Fatalf("err = %v, want ErrInvalidCallback", err)
	}
}

func TestHandleCallback_FailureFailsOrder(t *testing.T) {
	e := newTestEnv(t, Config{})
	fw := &fakeWhish{status: whish.CollectStatusFailed}
	e.enableWhish(fw)

	userID := bson.NewObjectID()
	orderID := bson.NewObjectID()
	in, err := e.svc.CreateWhishOrderIntent(context.Background(), userID, orderID, 30)
	if err != nil {
		t.Fatalf("create order intent: %v", err)
	}
	token := e.svc.tokens.SignExternalID(in.ExternalID)
	if err := e.svc.HandleCallback(context.Background(), strconv.FormatInt(in.ExternalID, 10), token); err != nil {
		t.Fatalf("HandleCallback: %v", err)
	}
	got := e.store.get(t, in.ID)
	if got.Status != StatusFailed {
		t.Fatalf("status = %s, want failed", got.Status)
	}
	if _, ok := e.settler.failed[orderID]; !ok {
		t.Fatalf("order %s was not failed", orderID.Hex())
	}
	if len(e.crediter.calls) != 0 {
		t.Fatalf("credits = %d, want 0 on failure", len(e.crediter.calls))
	}
}

func TestHandleCallback_PendingIsNoop(t *testing.T) {
	e := newTestEnv(t, Config{})
	fw := &fakeWhish{status: whish.CollectStatusPending}
	e.enableWhish(fw)

	in, err := e.svc.CreateWhishTopUpIntent(context.Background(), bson.NewObjectID(), 25, "idem-3")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	token := e.svc.tokens.SignExternalID(in.ExternalID)
	if err := e.svc.HandleCallback(context.Background(), strconv.FormatInt(in.ExternalID, 10), token); err != nil {
		t.Fatalf("HandleCallback: %v", err)
	}
	if got := e.store.get(t, in.ID); got.Status != StatusPending {
		t.Fatalf("status = %s, want pending (unresolved)", got.Status)
	}
	if len(e.crediter.calls) != 0 {
		t.Fatalf("credits = %d, want 0 while pending", len(e.crediter.calls))
	}
}
