package wallet

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// fakeBalanceRepo is an in-memory Repository whose ledger insert can fail.
type fakeBalanceRepo struct {
	balances   map[bson.ObjectID]float64
	failCreate bool
	ledger     []*WalletTransaction
}

func (f *fakeBalanceRepo) Create(_ context.Context, tx *WalletTransaction) error {
	if f.failCreate {
		return errors.New("ledger down")
	}
	if tx.ID.IsZero() {
		tx.ID = bson.NewObjectID()
	}
	f.ledger = append(f.ledger, tx)
	return nil
}

func (f *fakeBalanceRepo) FindByUserID(_ context.Context, _ bson.ObjectID) ([]*WalletTransaction, error) {
	return f.ledger, nil
}

func (f *fakeBalanceRepo) GetBalance(_ context.Context, id bson.ObjectID) (float64, error) {
	return f.balances[id], nil
}

func (f *fakeBalanceRepo) Credit(_ context.Context, id bson.ObjectID, amount float64) (float64, error) {
	f.balances[id] += amount
	return f.balances[id], nil
}

func (f *fakeBalanceRepo) Debit(_ context.Context, id bson.ObjectID, amount float64) (float64, error) {
	if f.balances[id] < amount {
		return 0, ErrInsufficientFunds
	}
	f.balances[id] -= amount
	return f.balances[id], nil
}

func (f *fakeBalanceRepo) TopUpsForDay(_ context.Context, _ time.Time) (float64, error) {
	return 0, nil
}

// fakeTopUpStore is an in-memory TopUpStore.
type fakeTopUpStore struct {
	byID map[bson.ObjectID]*TopUpRequest
}

func newFakeTopUpStore() *fakeTopUpStore {
	return &fakeTopUpStore{byID: map[bson.ObjectID]*TopUpRequest{}}
}

func (f *fakeTopUpStore) Create(_ context.Context, req *TopUpRequest) error {
	if req.ID.IsZero() {
		req.ID = bson.NewObjectID()
	}
	req.Status = TopUpPending
	f.byID[req.ID] = req
	return nil
}

func (f *fakeTopUpStore) FindByUser(_ context.Context, userID bson.ObjectID) ([]*TopUpRequest, error) {
	out := []*TopUpRequest{}
	for _, r := range f.byID {
		if r.UserID == userID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeTopUpStore) CountPendingForUser(_ context.Context, userID bson.ObjectID) (int64, error) {
	var n int64
	for _, r := range f.byID {
		if r.UserID == userID && r.Status == TopUpPending {
			n++
		}
	}
	return n, nil
}

func (f *fakeTopUpStore) CountPending(_ context.Context) (int64, error) {
	var n int64
	for _, r := range f.byID {
		if r.Status == TopUpPending {
			n++
		}
	}
	return n, nil
}

func (f *fakeTopUpStore) List(_ context.Context, _, _ string, _ pagination.Params) ([]*TopUpRequest, int64, error) {
	out := []*TopUpRequest{}
	for _, r := range f.byID {
		out = append(out, r)
	}
	return out, int64(len(out)), nil
}

func (f *fakeTopUpStore) Claim(_ context.Context, id bson.ObjectID, to TopUpStatus, decidedBy, reason string) (*TopUpRequest, error) {
	r, ok := f.byID[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	if r.Status != TopUpPending {
		return nil, apperrors.ErrConflict
	}
	r.Status = to
	r.DecidedBy = decidedBy
	r.DecisionReason = reason
	return r, nil
}

func (f *fakeTopUpStore) Revert(_ context.Context, id bson.ObjectID) error {
	if r, ok := f.byID[id]; ok {
		r.Status = TopUpPending
		r.DecidedBy = ""
		r.DecisionReason = ""
	}
	return nil
}

func (f *fakeTopUpStore) SetTxID(_ context.Context, id bson.ObjectID, txID string) error {
	if r, ok := f.byID[id]; ok {
		r.TxID = txID
	}
	return nil
}

func newTopUpSUT() (*WalletService, *fakeBalanceRepo, *fakeTopUpStore) {
	repo := &fakeBalanceRepo{balances: map[bson.ObjectID]float64{}}
	store := newFakeTopUpStore()
	return NewWalletService(repo, store), repo, store
}

func seedPending(store *fakeTopUpStore, userID bson.ObjectID, amount float64) *TopUpRequest {
	req := &TopUpRequest{UserID: userID, Amount: amount, Currency: "USD", Channel: "whish"}
	_ = store.Create(context.Background(), req)
	return req
}

func TestCreateTopUpRequestValidation(t *testing.T) {
	svc, _, store := newTopUpSUT()
	uid := bson.NewObjectID()

	if _, err := svc.CreateTopUpRequest(context.Background(), uid, TopUpInput{Amount: 0, Channel: "whish"}); err == nil {
		t.Fatal("zero amount accepted")
	}
	if _, err := svc.CreateTopUpRequest(context.Background(), uid, TopUpInput{Amount: 25, Channel: "paypal"}); err == nil {
		t.Fatal("unknown channel accepted")
	}
	req, err := svc.CreateTopUpRequest(context.Background(), uid, TopUpInput{Amount: 25, Channel: "Whish", Note: "ref 123"})
	if err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	if req.Status != TopUpPending || req.Channel != "whish" {
		t.Fatalf("unexpected request: %+v", req)
	}
	// Pending-queue guard: three pending requests block a fourth.
	seedPending(store, uid, 10)
	seedPending(store, uid, 10)
	if _, err := svc.CreateTopUpRequest(context.Background(), uid, TopUpInput{Amount: 5, Channel: "cash"}); err == nil {
		t.Fatal("fourth pending request accepted")
	}
}

func TestApproveTopUpCreditsAndWritesLedger(t *testing.T) {
	svc, repo, store := newTopUpSUT()
	uid := bson.NewObjectID()
	req := seedPending(store, uid, 40)

	out, err := svc.ApproveTopUpRequest(context.Background(), req.ID, "admin@x")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if out.Status != TopUpApproved || out.TxID == "" {
		t.Fatalf("unexpected request after approve: %+v", out)
	}
	if repo.balances[uid] != 40 {
		t.Fatalf("balance = %v, want 40", repo.balances[uid])
	}
	if len(repo.ledger) != 1 || repo.ledger[0].Type != TxTypeTopUp || repo.ledger[0].Ref != req.ID.Hex() {
		t.Fatalf("unexpected ledger: %+v", repo.ledger)
	}
}

func TestApproveTopUpTwiceConflicts(t *testing.T) {
	svc, repo, store := newTopUpSUT()
	uid := bson.NewObjectID()
	req := seedPending(store, uid, 40)

	if _, err := svc.ApproveTopUpRequest(context.Background(), req.ID, "a"); err != nil {
		t.Fatalf("first approve: %v", err)
	}
	if _, err := svc.ApproveTopUpRequest(context.Background(), req.ID, "b"); !errors.Is(err, apperrors.ErrConflict) {
		t.Fatalf("second approve error = %v, want ErrConflict", err)
	}
	if repo.balances[uid] != 40 {
		t.Fatalf("balance = %v, want exactly one credit (40)", repo.balances[uid])
	}
}

func TestApproveTopUpLedgerFailureCompensates(t *testing.T) {
	svc, repo, store := newTopUpSUT()
	uid := bson.NewObjectID()
	req := seedPending(store, uid, 40)
	repo.failCreate = true

	_, err := svc.ApproveTopUpRequest(context.Background(), req.ID, "a")
	if !errors.Is(err, ErrLedgerWriteFailed) {
		t.Fatalf("error = %v, want ErrLedgerWriteFailed", err)
	}
	if repo.balances[uid] != 0 {
		t.Fatalf("balance = %v, want 0 (credit reverted)", repo.balances[uid])
	}
	if store.byID[req.ID].Status != TopUpPending {
		t.Fatalf("request status = %q, want pending (reverted, retryable)", store.byID[req.ID].Status)
	}
}

func TestRejectTopUpRequiresReason(t *testing.T) {
	svc, _, store := newTopUpSUT()
	uid := bson.NewObjectID()
	req := seedPending(store, uid, 40)

	if _, err := svc.RejectTopUpRequest(context.Background(), req.ID, "a", "  "); err == nil {
		t.Fatal("empty reason accepted")
	}
	out, err := svc.RejectTopUpRequest(context.Background(), req.ID, "a", "no payment received")
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if out.Status != TopUpRejected || out.DecisionReason != "no payment received" {
		t.Fatalf("unexpected request after reject: %+v", out)
	}
}
