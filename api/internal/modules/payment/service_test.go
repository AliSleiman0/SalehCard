package payment

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/internal/platform/tron"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// --- fakes -----------------------------------------------------------------

// fakeStore is an in-memory Store honoring the same atomicity contracts as
// the Mongo implementation (status preconditions, unique txHash, idempotency).
type fakeStore struct {
	mu      sync.Mutex
	intents map[bson.ObjectID]*Intent
	seq     int64
	txSeen  map[string]bool // network|txHash uniqueness

	failMarkConfirmed int // fail the next N MarkConfirmed calls
}

func newFakeStore() *fakeStore {
	return &fakeStore{intents: map[bson.ObjectID]*Intent{}, txSeen: map[string]bool{}}
}

func (f *fakeStore) Insert(_ context.Context, in *Intent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if in.IdempotencyKey != "" {
		for _, e := range f.intents {
			if e.UserID == in.UserID && e.IdempotencyKey == in.IdempotencyKey {
				return apperrors.ErrConflict
			}
		}
	}
	if in.ID.IsZero() {
		in.ID = bson.NewObjectID()
	}
	cp := *in
	f.intents[in.ID] = &cp
	return nil
}

func (f *fakeStore) GetByID(_ context.Context, id bson.ObjectID) (*Intent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	in, ok := f.intents[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	cp := *in
	return &cp, nil
}

func (f *fakeStore) GetByIdempotencyKey(_ context.Context, userID bson.ObjectID, key string) (*Intent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, in := range f.intents {
		if in.UserID == userID && in.IdempotencyKey == key {
			cp := *in
			return &cp, nil
		}
	}
	return nil, apperrors.ErrNotFound
}

func (f *fakeStore) GetActiveByOrder(_ context.Context, orderID bson.ObjectID) (*Intent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, in := range f.intents {
		if in.OrderID != nil && *in.OrderID == orderID && (in.Status == StatusPending || in.Status == StatusConfirming) {
			cp := *in
			return &cp, nil
		}
	}
	return nil, apperrors.ErrNotFound
}

func (f *fakeStore) GetByExternalID(_ context.Context, externalID int64) (*Intent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, in := range f.intents {
		if in.Provider == ProviderWhish && in.ExternalID == externalID {
			cp := *in
			return &cp, nil
		}
	}
	return nil, apperrors.ErrNotFound
}

func (f *fakeStore) UpdateAfterInitiate(_ context.Context, id bson.ObjectID, redirectURL, providerRef string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	in, ok := f.intents[id]
	if !ok {
		return apperrors.ErrNotFound
	}
	in.RedirectURL = redirectURL
	in.ProviderRef = providerRef
	return nil
}

func (f *fakeStore) ClaimWhishConfirmed(_ context.Context, id bson.ObjectID, receivedMicros int64, payerPhone string) (*Intent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	in, ok := f.intents[id]
	if !ok || in.Status != StatusPending {
		return nil, apperrors.ErrConflict
	}
	in.Status = StatusConfirming
	in.AmountReceivedMicros = receivedMicros
	in.PayerPhone = payerPhone
	cp := *in
	return &cp, nil
}

func (f *fakeStore) MarkFailed(_ context.Context, id bson.ObjectID, reason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	in, ok := f.intents[id]
	if !ok || in.Status != StatusPending {
		return apperrors.ErrConflict
	}
	in.Status = StatusFailed
	in.Settlement = reason
	return nil
}

func (f *fakeStore) ListWhishExpiryCandidates(_ context.Context, now time.Time, _ int) ([]*Intent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Intent
	for _, in := range f.intents {
		if in.Provider == ProviderWhish && in.Status == StatusPending && in.ExpiresAt.Before(now) {
			cp := *in
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeStore) CountOpenForUser(_ context.Context, userID bson.ObjectID) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for _, in := range f.intents {
		if in.UserID == userID && in.Status == StatusPending {
			n++
		}
	}
	return n, nil
}

func (f *fakeStore) ListWatchable(_ context.Context, now time.Time, grace time.Duration, _ int) ([]*Intent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Intent
	for _, in := range f.intents {
		if in.Status == StatusPending ||
			(in.Status == StatusExpired && in.TxHash == "" && in.ExpiresAt.After(now.Add(-grace))) {
			cp := *in
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeStore) ListSettleRetries(_ context.Context, _ int) ([]*Intent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Intent
	for _, in := range f.intents {
		if in.Status == StatusConfirming {
			cp := *in
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeStore) ListExpiryCandidates(_ context.Context, now time.Time, _ int) ([]*Intent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Intent
	for _, in := range f.intents {
		if in.Status == StatusPending && in.ExpiresAt.Before(now) {
			cp := *in
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeStore) ClaimPaymentSeen(_ context.Context, id bson.ObjectID, txHash, from string, receivedMicros int64) (*Intent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	in, ok := f.intents[id]
	if !ok || (in.Status != StatusPending && in.Status != StatusExpired) {
		return nil, apperrors.ErrConflict
	}
	key := in.Network + "|" + txHash
	if f.txSeen[key] {
		return nil, apperrors.ErrConflict
	}
	f.txSeen[key] = true
	in.Status = StatusConfirming
	in.TxHash = txHash
	in.FromAddress = from
	in.AmountReceivedMicros = receivedMicros
	cp := *in
	return &cp, nil
}

func (f *fakeStore) MarkConfirmed(_ context.Context, id bson.ObjectID, settlement string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failMarkConfirmed > 0 {
		f.failMarkConfirmed--
		return errors.New("simulated mark-confirmed failure")
	}
	in, ok := f.intents[id]
	if !ok || in.Status != StatusConfirming {
		return apperrors.ErrConflict
	}
	now := time.Now().UTC()
	in.Status = StatusConfirmed
	in.Settlement = settlement
	in.ConfirmedAt = &now
	return nil
}

func (f *fakeStore) IncSettleAttempts(_ context.Context, id bson.ObjectID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if in, ok := f.intents[id]; ok {
		in.SettleAttempts++
	}
	return nil
}

func (f *fakeStore) ExpireOne(_ context.Context, id bson.ObjectID, now time.Time) (*Intent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	in, ok := f.intents[id]
	if !ok || in.Status != StatusPending || !in.ExpiresAt.Before(now) {
		return nil, apperrors.ErrConflict
	}
	in.Status = StatusExpired
	cp := *in
	return &cp, nil
}

func (f *fakeStore) NextDerivationIndex(_ context.Context) (uint32, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	return uint32(f.seq - 1), nil
}

func (f *fakeStore) ListAll(_ context.Context, _ IntentFilter, _ pagination.Params) ([]*Intent, int64, error) {
	return nil, 0, nil
}

func (f *fakeStore) get(t *testing.T, id bson.ObjectID) *Intent {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	in, ok := f.intents[id]
	if !ok {
		t.Fatalf("intent %s not in store", id.Hex())
	}
	cp := *in
	return &cp
}

// creditCall records one wallet TopUp invocation.
type creditCall struct {
	UserID bson.ObjectID
	Amount float64
	Method string
	Ref    string
}

// fakeCrediter is the wallet port fake. Refs already credited return a mongo
// duplicate-key error, mirroring the unique (method, ref) ledger index.
type fakeCrediter struct {
	mu      sync.Mutex
	calls   []creditCall
	byRef   map[string]bool
	failN   int // fail the next N calls with a generic error
	lastErr error
}

func newFakeCrediter() *fakeCrediter { return &fakeCrediter{byRef: map[string]bool{}} }

func duplicateKeyErr() error {
	return mongo.WriteException{WriteErrors: []mongo.WriteError{{Code: 11000}}}
}

func (f *fakeCrediter) TopUp(_ context.Context, userID bson.ObjectID, amount float64, method, ref string) (*wallet.WalletTransaction, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failN > 0 {
		f.failN--
		f.lastErr = errors.New("simulated wallet failure")
		return nil, f.lastErr
	}
	if f.byRef[ref] {
		return nil, duplicateKeyErr()
	}
	f.byRef[ref] = true
	f.calls = append(f.calls, creditCall{UserID: userID, Amount: amount, Method: method, Ref: ref})
	return &wallet.WalletTransaction{Amount: amount, Method: method, Ref: ref}, nil
}

// fakeSettler is the order port fake.
type fakeSettler struct {
	mu           sync.Mutex
	fulfilled    []bson.ObjectID
	failed       map[bson.ObjectID]string
	fulfillErr   error
	failOrderErr error
}

func newFakeSettler() *fakeSettler { return &fakeSettler{failed: map[bson.ObjectID]string{}} }

func (f *fakeSettler) FulfillPaidOrder(_ context.Context, orderID bson.ObjectID, _ float64, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fulfillErr != nil {
		return f.fulfillErr
	}
	f.fulfilled = append(f.fulfilled, orderID)
	return nil
}

func (f *fakeSettler) FailUnpaidOrder(_ context.Context, orderID bson.ObjectID, reason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failOrderErr != nil {
		return f.failOrderErr
	}
	f.failed[orderID] = reason
	return nil
}

// fakeNotifier records notifications.
type fakeNotifier struct {
	mu    sync.Mutex
	notes []notification.Note
}

func (f *fakeNotifier) Notify(_ context.Context, _ bson.ObjectID, n notification.Note) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notes = append(f.notes, n)
}

func (f *fakeNotifier) kinds() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.notes))
	for i, n := range f.notes {
		out[i] = n.Kind
	}
	return out
}

// --- helpers ----------------------------------------------------------------

// testXPub derives a valid watch-only account key for tests (same seed as
// platform/tron's vector tests).
func testXPub(t *testing.T) string {
	t.Helper()
	seed, _ := hex.DecodeString("5eb00bbddcf069084889a8ab9155568165f5c453ccb85e70811aaed6f6da5fc19a5ac40b389cd370d086206dec8aa6c43daea6690f20ad3d8d48b2d2ce9e38e4")
	key, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []uint32{44, 195, 0} {
		if key, err = key.Derive(hdkeychain.HardenedKeyStart + step); err != nil {
			t.Fatal(err)
		}
	}
	xpub, err := key.Neuter()
	if err != nil {
		t.Fatal(err)
	}
	return xpub.String()
}

type testEnv struct {
	store    *fakeStore
	crediter *fakeCrediter
	settler  *fakeSettler
	notifier *fakeNotifier
	svc      *Service
}

func newTestEnv(t *testing.T, cfg Config) *testEnv {
	t.Helper()
	if cfg.XPub == "" {
		cfg.XPub = testXPub(t)
	}
	e := &testEnv{
		store:    newFakeStore(),
		crediter: newFakeCrediter(),
		settler:  newFakeSettler(),
		notifier: &fakeNotifier{},
	}
	e.svc = NewService(e.store, tron.NewStub(tron.StubConfig{Delay: time.Hour}), e.crediter, e.notifier, cfg)
	e.svc.SetOrderSettler(e.settler)
	return e
}

// claimedOrderIntent seeds a confirming order intent as the watcher would
// leave it after ClaimPaymentSeen.
func (e *testEnv) claimedOrderIntent(t *testing.T, expectedMicros, receivedMicros int64) *Intent {
	t.Helper()
	userID := bson.NewObjectID()
	orderID := bson.NewObjectID()
	in, err := e.svc.CreateOrderIntent(context.Background(), userID, orderID, MicrosToUSD(expectedMicros))
	if err != nil {
		t.Fatalf("CreateOrderIntent: %v", err)
	}
	claimed, err := e.store.ClaimPaymentSeen(context.Background(), in.ID, "tx-"+in.ID.Hex(), "TSender", receivedMicros)
	if err != nil {
		t.Fatalf("ClaimPaymentSeen: %v", err)
	}
	return claimed
}

// --- tests -------------------------------------------------------------------

func TestCanTransition(t *testing.T) {
	cases := []struct {
		cur, next IntentStatus
		want      bool
	}{
		{StatusPending, StatusConfirming, true},
		{StatusPending, StatusExpired, true},
		{StatusPending, StatusFailed, true},
		{StatusPending, StatusConfirmed, false},
		{StatusConfirming, StatusFailed, false},
		{StatusFailed, StatusConfirming, false},
		{StatusConfirming, StatusConfirmed, true},
		{StatusConfirming, StatusExpired, false},
		{StatusConfirming, StatusPending, false},
		{StatusExpired, StatusConfirming, true},
		{StatusExpired, StatusConfirmed, false},
		{StatusConfirmed, StatusConfirming, false},
		{StatusConfirmed, StatusExpired, false},
	}
	for _, c := range cases {
		if got := CanTransition(c.cur, c.next); got != c.want {
			t.Errorf("CanTransition(%s → %s) = %v, want %v", c.cur, c.next, got, c.want)
		}
	}
}

func TestMicrosConversion(t *testing.T) {
	cases := []struct {
		usd    float64
		micros int64
	}{
		{1, 1_000_000},
		{0.01, 10_000},
		{49.99, 49_990_000},
		{10_000, 10_000_000_000},
	}
	for _, c := range cases {
		if got := USDToMicros(c.usd); got != c.micros {
			t.Errorf("USDToMicros(%v) = %d, want %d", c.usd, got, c.micros)
		}
		if got := MicrosToUSD(c.micros); got != c.usd {
			t.Errorf("MicrosToUSD(%d) = %v, want %v", c.micros, got, c.usd)
		}
	}
}

func TestCreateTopUpIntent(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path derives a unique address per intent", func(t *testing.T) {
		e := newTestEnv(t, Config{})
		userA := bson.NewObjectID()
		a, err := e.svc.CreateTopUpIntent(ctx, userA, 25, "")
		if err != nil {
			t.Fatalf("CreateTopUpIntent: %v", err)
		}
		if a.Status != StatusPending || a.Network != NetworkTRC20 {
			t.Errorf("unexpected intent: %+v", a)
		}
		if a.AmountExpectedMicros != 25_000_000 {
			t.Errorf("expected 25000000 micros, got %d", a.AmountExpectedMicros)
		}
		if a.Address == "" || a.Address[0] != 'T' {
			t.Errorf("address not derived: %q", a.Address)
		}
		b, err := e.svc.CreateTopUpIntent(ctx, userA, 25, "")
		if err != nil {
			t.Fatalf("second intent: %v", err)
		}
		if b.Address == a.Address {
			t.Error("two intents share a deposit address")
		}
	})

	t.Run("disabled service refuses", func(t *testing.T) {
		e := &testEnv{store: newFakeStore(), crediter: newFakeCrediter(), notifier: &fakeNotifier{}}
		e.svc = NewService(e.store, tron.NewStub(tron.StubConfig{}), e.crediter, e.notifier, Config{XPub: ""})
		_, err := e.svc.CreateTopUpIntent(ctx, bson.NewObjectID(), 25, "")
		var appErr *apperrors.AppError
		if !errors.As(err, &appErr) || appErr.Code != "PAYMENT_METHOD_UNAVAILABLE" {
			t.Fatalf("want PAYMENT_METHOD_UNAVAILABLE, got %v", err)
		}
	})

	t.Run("amount bounds", func(t *testing.T) {
		e := newTestEnv(t, Config{})
		for _, amount := range []float64{0, -5, maxIntentUSD + 1} {
			if _, err := e.svc.CreateTopUpIntent(ctx, bson.NewObjectID(), amount, ""); err == nil {
				t.Errorf("amount %v accepted", amount)
			}
		}
	})

	t.Run("idempotency key replays the same intent", func(t *testing.T) {
		e := newTestEnv(t, Config{})
		userID := bson.NewObjectID()
		a, err := e.svc.CreateTopUpIntent(ctx, userID, 25, "key-1")
		if err != nil {
			t.Fatal(err)
		}
		b, err := e.svc.CreateTopUpIntent(ctx, userID, 25, "key-1")
		if err != nil {
			t.Fatal(err)
		}
		if a.ID != b.ID {
			t.Error("idempotent replay created a second intent")
		}
	})

	t.Run("pending cap", func(t *testing.T) {
		e := newTestEnv(t, Config{MaxPending: 2})
		userID := bson.NewObjectID()
		for i := range 2 {
			if _, err := e.svc.CreateTopUpIntent(ctx, userID, 10, fmt.Sprintf("k%d", i)); err != nil {
				t.Fatal(err)
			}
		}
		_, err := e.svc.CreateTopUpIntent(ctx, userID, 10, "k-final")
		var appErr *apperrors.AppError
		if !errors.As(err, &appErr) || appErr.Code != "TOO_MANY_PENDING" {
			t.Fatalf("want TOO_MANY_PENDING, got %v", err)
		}
	})
}

func TestGetIntentOwnership(t *testing.T) {
	e := newTestEnv(t, Config{})
	owner := bson.NewObjectID()
	in, err := e.svc.CreateTopUpIntent(context.Background(), owner, 25, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.GetIntent(context.Background(), owner, in.ID); err != nil {
		t.Errorf("owner read failed: %v", err)
	}
	if _, err := e.svc.GetIntent(context.Background(), bson.NewObjectID(), in.ID); !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("foreign read: want ErrNotFound, got %v", err)
	}
}

func TestSettleTopUp(t *testing.T) {
	e := newTestEnv(t, Config{})
	userID := bson.NewObjectID()
	in, err := e.svc.CreateTopUpIntent(context.Background(), userID, 25, "")
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := e.store.ClaimPaymentSeen(context.Background(), in.ID, "tx-1", "TSender", 25_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.svc.settle(context.Background(), claimed); err != nil {
		t.Fatalf("settle: %v", err)
	}
	if got := e.store.get(t, in.ID); got.Status != StatusConfirmed || got.Settlement != SettlementWalletTopUp {
		t.Errorf("intent not confirmed as topup: %+v", got)
	}
	if len(e.crediter.calls) != 1 || e.crediter.calls[0].Amount != 25 ||
		e.crediter.calls[0].Method != "usdt_trc20" || e.crediter.calls[0].Ref != in.ID.Hex() {
		t.Errorf("unexpected credit: %+v", e.crediter.calls)
	}
	if kinds := e.notifier.kinds(); len(kinds) != 1 || kinds[0] != notification.KindPaymentConfirmed {
		t.Errorf("unexpected notifications: %v", kinds)
	}
}

func TestSettleOrder(t *testing.T) {
	t.Run("exact payment fulfills the order", func(t *testing.T) {
		e := newTestEnv(t, Config{})
		in := e.claimedOrderIntent(t, 50_000_000, 50_000_000)
		if err := e.svc.settle(context.Background(), in); err != nil {
			t.Fatalf("settle: %v", err)
		}
		if len(e.settler.fulfilled) != 1 || e.settler.fulfilled[0] != *in.OrderID {
			t.Errorf("order not fulfilled: %+v", e.settler.fulfilled)
		}
		if len(e.crediter.calls) != 0 {
			t.Errorf("exact payment should not credit the wallet: %+v", e.crediter.calls)
		}
		if got := e.store.get(t, in.ID); got.Settlement != SettlementOrderFulfilled {
			t.Errorf("settlement = %q", got.Settlement)
		}
	})

	t.Run("over-payment credits the excess", func(t *testing.T) {
		e := newTestEnv(t, Config{})
		in := e.claimedOrderIntent(t, 50_000_000, 52_500_000)
		if err := e.svc.settle(context.Background(), in); err != nil {
			t.Fatalf("settle: %v", err)
		}
		if len(e.settler.fulfilled) != 1 {
			t.Fatalf("order not fulfilled")
		}
		if len(e.crediter.calls) != 1 || e.crediter.calls[0].Amount != 2.5 || e.crediter.calls[0].Ref != in.ID.Hex()+":excess" {
			t.Errorf("unexpected excess credit: %+v", e.crediter.calls)
		}
	})

	t.Run("dust over-payment is not credited", func(t *testing.T) {
		e := newTestEnv(t, Config{})
		in := e.claimedOrderIntent(t, 50_000_000, 50_005_000) // +0.005 USD < 1 cent
		if err := e.svc.settle(context.Background(), in); err != nil {
			t.Fatalf("settle: %v", err)
		}
		if len(e.crediter.calls) != 0 {
			t.Errorf("dust was credited: %+v", e.crediter.calls)
		}
	})

	t.Run("under-payment fails the order and credits the received amount", func(t *testing.T) {
		e := newTestEnv(t, Config{})
		in := e.claimedOrderIntent(t, 50_000_000, 30_000_000)
		if err := e.svc.settle(context.Background(), in); err != nil {
			t.Fatalf("settle: %v", err)
		}
		if reason := e.settler.failed[*in.OrderID]; reason != "underpaid" {
			t.Errorf("order fail reason = %q", reason)
		}
		if len(e.crediter.calls) != 1 || e.crediter.calls[0].Amount != 30 {
			t.Errorf("unexpected credit: %+v", e.crediter.calls)
		}
		if got := e.store.get(t, in.ID); got.Settlement != SettlementUnderpaid {
			t.Errorf("settlement = %q", got.Settlement)
		}
		if kinds := e.notifier.kinds(); len(kinds) != 1 || kinds[0] != notification.KindPaymentUnderpaid {
			t.Errorf("unexpected notifications: %v", kinds)
		}
	})

	t.Run("unpayable order falls back to a wallet credit", func(t *testing.T) {
		e := newTestEnv(t, Config{})
		in := e.claimedOrderIntent(t, 50_000_000, 50_000_000)
		e.settler.fulfillErr = ErrOrderNotPayable
		if err := e.svc.settle(context.Background(), in); err != nil {
			t.Fatalf("settle: %v", err)
		}
		if len(e.crediter.calls) != 1 || e.crediter.calls[0].Amount != 50 {
			t.Errorf("unexpected credit: %+v", e.crediter.calls)
		}
		if got := e.store.get(t, in.ID); got.Settlement != SettlementLate {
			t.Errorf("settlement = %q", got.Settlement)
		}
	})

	t.Run("fulfillment error keeps the intent confirming for retry", func(t *testing.T) {
		e := newTestEnv(t, Config{})
		in := e.claimedOrderIntent(t, 50_000_000, 50_000_000)
		e.settler.fulfillErr = errors.New("db down")
		if err := e.svc.settle(context.Background(), in); err == nil {
			t.Fatal("expected an error")
		}
		got := e.store.get(t, in.ID)
		if got.Status != StatusConfirming {
			t.Errorf("status = %s, want confirming", got.Status)
		}
		if got.SettleAttempts != 1 {
			t.Errorf("settleAttempts = %d, want 1", got.SettleAttempts)
		}
		// dependency recovers → retry succeeds
		e.settler.fulfillErr = nil
		if err := e.svc.settle(context.Background(), got); err != nil {
			t.Fatalf("retry settle: %v", err)
		}
		if final := e.store.get(t, in.ID); final.Status != StatusConfirmed {
			t.Errorf("status after retry = %s", final.Status)
		}
	})
}

// TestSettleRetryDoesNotDoubleCredit is the money-safety core: a MarkConfirmed
// failure after a successful credit must not credit again on retry (the
// duplicate ledger ref is treated as already-done).
func TestSettleRetryDoesNotDoubleCredit(t *testing.T) {
	e := newTestEnv(t, Config{})
	userID := bson.NewObjectID()
	in, err := e.svc.CreateTopUpIntent(context.Background(), userID, 25, "")
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := e.store.ClaimPaymentSeen(context.Background(), in.ID, "tx-1", "TSender", 25_000_000)
	if err != nil {
		t.Fatal(err)
	}
	e.store.failMarkConfirmed = 1
	if err := e.svc.settle(context.Background(), claimed); err == nil {
		t.Fatal("expected mark-confirmed failure")
	}
	if got := e.store.get(t, in.ID); got.Status != StatusConfirming {
		t.Fatalf("status = %s, want confirming", got.Status)
	}
	// Watcher retry: credit dedupes on ref, mark succeeds.
	if err := e.svc.settle(context.Background(), e.store.get(t, in.ID)); err != nil {
		t.Fatalf("retry settle: %v", err)
	}
	if len(e.crediter.calls) != 1 {
		t.Fatalf("wallet credited %d times, want exactly 1", len(e.crediter.calls))
	}
	if got := e.store.get(t, in.ID); got.Status != StatusConfirmed {
		t.Errorf("status = %s, want confirmed", got.Status)
	}
}
