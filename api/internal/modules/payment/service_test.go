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
	mu       sync.Mutex
	intents  map[bson.ObjectID]*Intent
	seq      int64
	saltSeq  int64
	txSeen   map[string]bool // network|txHash uniqueness
	deposits map[bson.ObjectID]*Deposit

	failMarkConfirmed int // fail the next N MarkConfirmed calls
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		intents:  map[bson.ObjectID]*Intent{},
		txSeen:   map[string]bool{},
		deposits: map[bson.ObjectID]*Deposit{},
	}
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
	// Mirror the unique partial (network, sharedOpen) index: no two OPEN
	// shared intents ON THE SAME NETWORK may expect the same amount.
	if in.SharedOpen {
		for _, e := range f.intents {
			if e.SharedOpen && e.Network == in.Network && e.AmountExpectedMicros == in.AmountExpectedMicros {
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

func (f *fakeStore) NextAmountSalt(_ context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saltSeq++
	return 1 + ((f.saltSeq - 1) % saltRange), nil
}

func (f *fakeStore) ReleaseSharedSlots(_ context.Context, now time.Time, grace time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, in := range f.intents {
		if in.SharedOpen && in.Status == StatusExpired && in.TxHash == "" && in.ExpiresAt.Before(now.Add(-grace)) {
			in.SharedOpen = false
		}
	}
	return nil
}

func (f *fakeStore) FilterKnownTxHashes(_ context.Context, network string, hashes []string) (map[string]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	known := map[string]bool{}
	for _, h := range hashes {
		if f.txSeen[network+"|"+h] {
			known[h] = true
		}
	}
	return known, nil
}

func (f *fakeStore) RecordUnmatchedDeposit(_ context.Context, d *Deposit) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, e := range f.deposits {
		if e.Network == d.Network && e.TxHash == d.TxHash {
			return nil // idempotent upsert: already recorded
		}
	}
	cp := *d
	if cp.ID.IsZero() {
		cp.ID = bson.NewObjectID()
	}
	if cp.Status == "" {
		cp.Status = DepositUnmatched
	}
	if cp.SeenAt.IsZero() {
		cp.SeenAt = time.Now().UTC()
	}
	f.deposits[cp.ID] = &cp
	return nil
}

func (f *fakeStore) ListDeposits(_ context.Context, status string, _ pagination.Params) ([]*Deposit, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Deposit
	for _, d := range f.deposits {
		if status == "" || d.Status == status {
			cp := *d
			out = append(out, &cp)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakeStore) GetDeposit(_ context.Context, id bson.ObjectID) (*Deposit, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.deposits[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	cp := *d
	return &cp, nil
}

func (f *fakeStore) ClaimDepositAttribution(_ context.Context, id, userID bson.ObjectID, adminRef, note string) (*Deposit, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.deposits[id]
	if !ok || d.Status != DepositUnmatched {
		return nil, apperrors.ErrConflict
	}
	now := time.Now().UTC()
	d.Status = DepositCredited
	d.AttributedUserID = &userID
	d.AttributedBy = adminRef
	d.AttributedAt = &now
	d.Note = note
	cp := *d
	return &cp, nil
}

func (f *fakeStore) RevertDepositAttribution(_ context.Context, id bson.ObjectID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if d, ok := f.deposits[id]; ok && d.Status == DepositCredited {
		d.Status = DepositUnmatched
		d.AttributedUserID = nil
		d.AttributedBy = ""
		d.AttributedAt = nil
	}
	return nil
}

func (f *fakeStore) MarkDepositIgnored(_ context.Context, id bson.ObjectID, adminRef, note string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.deposits[id]
	if !ok || d.Status != DepositUnmatched {
		return apperrors.ErrConflict
	}
	now := time.Now().UTC()
	d.Status = DepositIgnored
	d.AttributedBy = adminRef
	d.AttributedAt = &now
	d.Note = note
	return nil
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

func (f *fakeNotifier) NotifyMany(_ context.Context, ids []bson.ObjectID, n notification.Note) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notes = append(f.notes, n)
	return len(ids)
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
	e.svc = NewService(e.store, tron.NewStub(tron.StubConfig{Delay: time.Hour}), nil, e.crediter, e.notifier, cfg)
	e.svc.SetOrderSettler(e.settler)
	return e
}

// claimedOrderIntent seeds a confirming order intent as the watcher would
// leave it after ClaimPaymentSeen.
func (e *testEnv) claimedOrderIntent(t *testing.T, expectedMicros, receivedMicros int64) *Intent {
	t.Helper()
	userID := bson.NewObjectID()
	orderID := bson.NewObjectID()
	in, err := e.svc.CreateOrderIntent(context.Background(), userID, orderID, MicrosToUSD(expectedMicros), "")
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
		{StatusPending, StatusConfirmed, false},
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
		a, err := e.svc.CreateTopUpIntent(ctx, userA, 25, "", "")
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
		if a.AddressMode != AddressModeDerived || a.SharedOpen {
			t.Errorf("derived intent not stamped: mode=%q sharedOpen=%v", a.AddressMode, a.SharedOpen)
		}
		b, err := e.svc.CreateTopUpIntent(ctx, userA, 25, "", "")
		if err != nil {
			t.Fatalf("second intent: %v", err)
		}
		if b.Address == a.Address {
			t.Error("two intents share a deposit address")
		}
	})

	t.Run("disabled service refuses", func(t *testing.T) {
		e := &testEnv{store: newFakeStore(), crediter: newFakeCrediter(), notifier: &fakeNotifier{}}
		e.svc = NewService(e.store, tron.NewStub(tron.StubConfig{}), nil, e.crediter, e.notifier, Config{XPub: ""})
		_, err := e.svc.CreateTopUpIntent(ctx, bson.NewObjectID(), 25, "", "")
		var appErr *apperrors.AppError
		if !errors.As(err, &appErr) || appErr.Code != "PAYMENT_METHOD_UNAVAILABLE" {
			t.Fatalf("want PAYMENT_METHOD_UNAVAILABLE, got %v", err)
		}
	})

	t.Run("amount bounds", func(t *testing.T) {
		e := newTestEnv(t, Config{})
		for _, amount := range []float64{0, -5, maxIntentUSD + 1} {
			if _, err := e.svc.CreateTopUpIntent(ctx, bson.NewObjectID(), amount, "", ""); err == nil {
				t.Errorf("amount %v accepted", amount)
			}
		}
	})

	t.Run("idempotency key replays the same intent", func(t *testing.T) {
		e := newTestEnv(t, Config{})
		userID := bson.NewObjectID()
		a, err := e.svc.CreateTopUpIntent(ctx, userID, 25, "", "key-1")
		if err != nil {
			t.Fatal(err)
		}
		b, err := e.svc.CreateTopUpIntent(ctx, userID, 25, "", "key-1")
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
			if _, err := e.svc.CreateTopUpIntent(ctx, userID, 10, "", fmt.Sprintf("k%d", i)); err != nil {
				t.Fatal(err)
			}
		}
		_, err := e.svc.CreateTopUpIntent(ctx, userID, 10, "", "k-final")
		var appErr *apperrors.AppError
		if !errors.As(err, &appErr) || appErr.Code != "TOO_MANY_PENDING" {
			t.Fatalf("want TOO_MANY_PENDING, got %v", err)
		}
	})
}

func TestGetIntentOwnership(t *testing.T) {
	e := newTestEnv(t, Config{})
	owner := bson.NewObjectID()
	in, err := e.svc.CreateTopUpIntent(context.Background(), owner, 25, "", "")
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
	in, err := e.svc.CreateTopUpIntent(context.Background(), userID, 25, "", "")
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
	in, err := e.svc.CreateTopUpIntent(context.Background(), userID, 25, "", "")
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

// --- shared-address mode -----------------------------------------------------

// testSharedAddr is a valid TRON mainnet address for shared-mode tests.
const testSharedAddr = "TLRaHegyg2grMQqX85nJyCzbdRtvM5nCDn"

// testBEP20Addr is a valid BSC address for second-network tests.
const testBEP20Addr = "0x5e0a66cedc7688aab52c87dc02bff97f6575d7dc"

// newSharedTestEnv wires a service in shared-address mode (no xpub).
func newSharedTestEnv(t *testing.T, cfg Config) *testEnv {
	t.Helper()
	cfg.SharedAddress = testSharedAddr
	e := &testEnv{
		store:    newFakeStore(),
		crediter: newFakeCrediter(),
		settler:  newFakeSettler(),
		notifier: &fakeNotifier{},
	}
	e.svc = NewService(e.store, tron.NewStub(tron.StubConfig{Delay: time.Hour}), nil, e.crediter, e.notifier, cfg)
	e.svc.SetOrderSettler(e.settler)
	return e
}

func TestCreateTopUpIntentShared(t *testing.T) {
	ctx := context.Background()
	e := newSharedTestEnv(t, Config{})
	userID := bson.NewObjectID()

	a, err := e.svc.CreateTopUpIntent(ctx, userID, 10, "", "")
	if err != nil {
		t.Fatalf("CreateTopUpIntent: %v", err)
	}
	if a.Address != testSharedAddr || a.AddressMode != AddressModeShared || !a.SharedOpen {
		t.Errorf("shared intent not stamped: %+v", a)
	}
	if a.AmountSaltMicros < 1 || a.AmountSaltMicros > saltRange {
		t.Errorf("salt %d outside 1..%d", a.AmountSaltMicros, saltRange)
	}
	if a.AmountExpectedMicros != 10_000_000+a.AmountSaltMicros {
		t.Errorf("amount %d != base 10000000 + salt %d", a.AmountExpectedMicros, a.AmountSaltMicros)
	}

	// Same base amount → same address, DIFFERENT salted total.
	b, err := e.svc.CreateTopUpIntent(ctx, userID, 10, "", "")
	if err != nil {
		t.Fatalf("second intent: %v", err)
	}
	if b.Address != a.Address {
		t.Error("shared intents should share the deposit address")
	}
	if b.AmountExpectedMicros == a.AmountExpectedMicros {
		t.Error("two open shared intents expect the same amount")
	}
}

// TestSharedResaltOnCollision forces the cross-base collision the counter
// alone cannot prevent: base 10.000000+salt(1) was taken by intent A, and
// intent B's base 9.999999+salt(2) lands on the same 10_000_001 total — the
// unique open-amount guard must trip and B must retry with a fresh salt.
func TestSharedResaltOnCollision(t *testing.T) {
	ctx := context.Background()
	e := newSharedTestEnv(t, Config{})
	userID := bson.NewObjectID()

	a, err := e.svc.CreateTopUpIntent(ctx, userID, 10, "", "") // salt 1 → 10_000_001
	if err != nil {
		t.Fatal(err)
	}
	if a.AmountExpectedMicros != 10_000_001 {
		t.Fatalf("precondition: intent A amount = %d, want 10000001", a.AmountExpectedMicros)
	}

	b, err := e.svc.CreateTopUpIntent(ctx, userID, 9.999999, "", "") // salt 2 collides → resalt to 3
	if err != nil {
		t.Fatalf("resalt should have recovered the collision: %v", err)
	}
	if b.AmountExpectedMicros == a.AmountExpectedMicros {
		t.Errorf("collision not resolved: both intents expect %d", a.AmountExpectedMicros)
	}
	if b.AmountExpectedMicros != 9_999_999+b.AmountSaltMicros {
		t.Errorf("amount %d != base 9999999 + salt %d", b.AmountExpectedMicros, b.AmountSaltMicros)
	}
}

func TestEnabledMatrix(t *testing.T) {
	store, crediter, ntf := newFakeStore(), newFakeCrediter(), &fakeNotifier{}
	stub := tron.NewStub(tron.StubConfig{Delay: time.Hour})

	cases := []struct {
		name     string
		svc      *Service
		enabled  bool
		networks []string
	}{
		{"xpub only", NewService(store, stub, nil, crediter, ntf, Config{XPub: testXPub(t)}), true, []string{NetworkTRC20}},
		{"shared address only", NewService(store, stub, nil, crediter, ntf, Config{SharedAddress: testSharedAddr}), true, []string{NetworkTRC20}},
		{"neither", NewService(store, stub, nil, crediter, ntf, Config{}), false, nil},
		{"nil reader", NewService(store, nil, nil, crediter, ntf, Config{SharedAddress: testSharedAddr}), false, nil},
		{"bep20 only", NewService(store, nil, stub, crediter, ntf, Config{BEP20SharedAddress: testBEP20Addr}), true, []string{NetworkBEP20}},
		{"bep20 address without lister", NewService(store, stub, nil, crediter, ntf, Config{BEP20SharedAddress: testBEP20Addr}), false, nil},
		{"both networks", NewService(store, stub, stub, crediter, ntf, Config{SharedAddress: testSharedAddr, BEP20SharedAddress: testBEP20Addr}), true, []string{NetworkTRC20, NetworkBEP20}},
	}
	for _, c := range cases {
		if got := c.svc.Enabled(); got != c.enabled {
			t.Errorf("%s: Enabled() = %v, want %v", c.name, got, c.enabled)
		}
		got := c.svc.Networks()
		if len(got) != len(c.networks) {
			t.Errorf("%s: Networks() = %v, want %v", c.name, got, c.networks)
			continue
		}
		for i := range got {
			if got[i] != c.networks[i] {
				t.Errorf("%s: Networks() = %v, want %v", c.name, got, c.networks)
				break
			}
		}
	}
}

func seedDeposit(t *testing.T, e *testEnv, txHash string, micros int64) *Deposit {
	t.Helper()
	d := &Deposit{
		Network:      NetworkTRC20,
		TxHash:       txHash,
		FromAddress:  "TSender",
		ToAddress:    testSharedAddr,
		AmountMicros: micros,
		BlockTime:    time.Now().UTC(),
	}
	if err := e.store.RecordUnmatchedDeposit(context.Background(), d); err != nil {
		t.Fatalf("seed deposit: %v", err)
	}
	list, _, err := e.store.ListDeposits(context.Background(), DepositUnmatched, pagination.Params{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	for _, got := range list {
		if got.TxHash == txHash {
			return got
		}
	}
	t.Fatalf("seeded deposit %s not found", txHash)
	return nil
}

func TestAttributeDeposit(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path credits the wallet once", func(t *testing.T) {
		e := newSharedTestEnv(t, Config{})
		userID := bson.NewObjectID()
		d := seedDeposit(t, e, "tx-dep-1", 12_500_000)

		out, err := e.svc.AttributeDeposit(ctx, d.ID, userID, "admin@test", "customer forgot the cents")
		if err != nil {
			t.Fatalf("AttributeDeposit: %v", err)
		}
		if out.Status != DepositCredited || out.AttributedUserID == nil || *out.AttributedUserID != userID {
			t.Errorf("deposit after attribution: %+v", out)
		}
		if len(e.crediter.calls) != 1 {
			t.Fatalf("wallet credited %d times, want 1", len(e.crediter.calls))
		}
		call := e.crediter.calls[0]
		if call.UserID != userID || call.Amount != 12.5 || call.Ref != "deposit:tx-dep-1" {
			t.Errorf("unexpected credit: %+v", call)
		}
		if kinds := e.notifier.kinds(); len(kinds) != 1 || kinds[0] != notification.KindPaymentConfirmed {
			t.Errorf("notifications: %v", kinds)
		}

		// A second attribution attempt must conflict, not double-credit.
		if _, err := e.svc.AttributeDeposit(ctx, d.ID, bson.NewObjectID(), "admin2@test", ""); !errors.Is(err, apperrors.ErrConflict) {
			t.Errorf("second attribution: want ErrConflict, got %v", err)
		}
		if len(e.crediter.calls) != 1 {
			t.Errorf("wallet credited %d times after conflict, want 1", len(e.crediter.calls))
		}
	})

	t.Run("wallet failure reverts the claim", func(t *testing.T) {
		e := newSharedTestEnv(t, Config{})
		d := seedDeposit(t, e, "tx-dep-2", 5_000_000)
		e.crediter.failN = 1

		if _, err := e.svc.AttributeDeposit(ctx, d.ID, bson.NewObjectID(), "admin@test", ""); err == nil {
			t.Fatal("expected the wallet failure to surface")
		}
		got, err := e.store.GetDeposit(ctx, d.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != DepositUnmatched {
			t.Errorf("deposit status = %s after revert, want unmatched", got.Status)
		}

		// Retry succeeds normally.
		if _, err := e.svc.AttributeDeposit(ctx, d.ID, bson.NewObjectID(), "admin@test", ""); err != nil {
			t.Fatalf("retry after revert: %v", err)
		}
	})

	t.Run("retry after a credited-but-unreverted crash is idempotent", func(t *testing.T) {
		e := newSharedTestEnv(t, Config{})
		userID := bson.NewObjectID()
		d := seedDeposit(t, e, "tx-dep-3", 7_000_000)
		// Simulate: a prior attempt credited the wallet, crashed before the
		// deposit doc update stuck, and an operator reset it to unmatched.
		e.crediter.byRef["deposit:tx-dep-3"] = true

		out, err := e.svc.AttributeDeposit(ctx, d.ID, userID, "admin@test", "")
		if err != nil {
			t.Fatalf("idempotent retry: %v", err)
		}
		if out.Status != DepositCredited {
			t.Errorf("status = %s, want credited", out.Status)
		}
		if len(e.crediter.calls) != 0 {
			t.Errorf("wallet re-credited on a duplicate ref: %+v", e.crediter.calls)
		}
	})

	t.Run("ignore blocks later attribution", func(t *testing.T) {
		e := newSharedTestEnv(t, Config{})
		d := seedDeposit(t, e, "tx-dep-4", 100)

		if err := e.svc.IgnoreDeposit(ctx, d.ID, "admin@test", "dust"); err != nil {
			t.Fatalf("IgnoreDeposit: %v", err)
		}
		got, err := e.store.GetDeposit(ctx, d.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != DepositIgnored || got.Note != "dust" {
			t.Errorf("deposit after ignore: %+v", got)
		}
		if _, err := e.svc.AttributeDeposit(ctx, d.ID, bson.NewObjectID(), "admin@test", ""); !errors.Is(err, apperrors.ErrConflict) {
			t.Errorf("attribute after ignore: want ErrConflict, got %v", err)
		}
	})
}

// --- BEP20 second network ------------------------------------------------

// newDualNetworkTestEnv wires a service with BOTH shared networks enabled
// (the same stub serves as TRC20 reader and BEP20 lister).
func newDualNetworkTestEnv(t *testing.T) *testEnv {
	t.Helper()
	e := &testEnv{
		store:    newFakeStore(),
		crediter: newFakeCrediter(),
		settler:  newFakeSettler(),
		notifier: &fakeNotifier{},
	}
	stub := tron.NewStub(tron.StubConfig{Delay: time.Hour})
	e.svc = NewService(e.store, stub, stub, e.crediter, e.notifier,
		Config{SharedAddress: testSharedAddr, BEP20SharedAddress: testBEP20Addr})
	e.svc.SetOrderSettler(e.settler)
	return e
}

func TestCreateTopUpIntentBEP20(t *testing.T) {
	ctx := context.Background()

	t.Run("bep20 intent gets the BEP20 address, shared mode, salt", func(t *testing.T) {
		e := newDualNetworkTestEnv(t)
		in, err := e.svc.CreateTopUpIntent(ctx, bson.NewObjectID(), 10, NetworkBEP20, "")
		if err != nil {
			t.Fatalf("CreateTopUpIntent: %v", err)
		}
		if in.Network != NetworkBEP20 || in.Address != testBEP20Addr || in.AddressMode != AddressModeShared || !in.SharedOpen {
			t.Errorf("bep20 intent not stamped: %+v", in)
		}
		if in.AmountSaltMicros < 1 || in.AmountSaltMicros > saltRange {
			t.Errorf("salt %d outside 1..%d", in.AmountSaltMicros, saltRange)
		}
		if in.AmountExpectedMicros != 10_000_000+in.AmountSaltMicros {
			t.Errorf("amount %d != base 10000000 + salt %d", in.AmountExpectedMicros, in.AmountSaltMicros)
		}
	})

	t.Run("empty network defaults to trc20 with the TRC20 address", func(t *testing.T) {
		e := newDualNetworkTestEnv(t)
		in, err := e.svc.CreateTopUpIntent(ctx, bson.NewObjectID(), 10, "", "")
		if err != nil {
			t.Fatal(err)
		}
		if in.Network != NetworkTRC20 || in.Address != testSharedAddr {
			t.Errorf("default network intent: %+v", in)
		}
	})

	t.Run("unsupported network refused", func(t *testing.T) {
		e := newDualNetworkTestEnv(t)
		_, err := e.svc.CreateTopUpIntent(ctx, bson.NewObjectID(), 10, "erc20", "")
		var appErr *apperrors.AppError
		if !errors.As(err, &appErr) || appErr.Code != "BAD_REQUEST" {
			t.Fatalf("want BAD_REQUEST, got %v", err)
		}
	})

	t.Run("bep20 refused when only trc20 is configured", func(t *testing.T) {
		e := newSharedTestEnv(t, Config{})
		_, err := e.svc.CreateTopUpIntent(ctx, bson.NewObjectID(), 10, NetworkBEP20, "")
		var appErr *apperrors.AppError
		if !errors.As(err, &appErr) || appErr.Code != "BAD_REQUEST" {
			t.Fatalf("want BAD_REQUEST, got %v", err)
		}
	})
}

// TestSharedAmountUniquenessIsPerNetwork mirrors the compound
// (network, amountExpectedMicros) index semantics: the same open salted amount
// may coexist across networks but not within one.
func TestSharedAmountUniquenessIsPerNetwork(t *testing.T) {
	f := newFakeStore()
	mk := func(network string, amount int64) *Intent {
		return &Intent{
			UserID: bson.NewObjectID(), Purpose: PurposeTopUp, Network: network,
			AddressMode: AddressModeShared, SharedOpen: true,
			AmountExpectedMicros: amount, Status: StatusPending,
		}
	}
	if err := f.Insert(context.Background(), mk(NetworkTRC20, 10_000_001)); err != nil {
		t.Fatal(err)
	}
	if err := f.Insert(context.Background(), mk(NetworkBEP20, 10_000_001)); err != nil {
		t.Errorf("same amount on the OTHER network must be allowed: %v", err)
	}
	if err := f.Insert(context.Background(), mk(NetworkBEP20, 10_000_001)); !errors.Is(err, apperrors.ErrConflict) {
		t.Errorf("same amount on the SAME network must conflict, got %v", err)
	}
}

// TestBEP20SettleUsesBEP20Method: a settled BEP20 top-up must land in the
// ledger under usdt_bep20 (its own dedup index), not usdt_trc20.
func TestBEP20SettleUsesBEP20Method(t *testing.T) {
	ctx := context.Background()
	e := newDualNetworkTestEnv(t)
	userID := bson.NewObjectID()
	in, err := e.svc.CreateTopUpIntent(ctx, userID, 25, NetworkBEP20, "")
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := e.store.ClaimPaymentSeen(ctx, in.ID, "0xdeadbeef", "0xSender", in.AmountExpectedMicros)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.svc.settle(ctx, claimed); err != nil {
		t.Fatalf("settle: %v", err)
	}
	if len(e.crediter.calls) != 1 || e.crediter.calls[0].Method != "usdt_bep20" {
		t.Errorf("credits: %+v, want one usdt_bep20 credit", e.crediter.calls)
	}
	if got := e.store.get(t, in.ID); got.Status != StatusConfirmed || got.Settlement != SettlementWalletTopUp {
		t.Errorf("intent after settle: %+v", got)
	}
}

// TestAttributeDepositBEP20: attributing a bep20 unmatched deposit credits
// under the bep20 ledger method.
func TestAttributeDepositBEP20(t *testing.T) {
	ctx := context.Background()
	e := newDualNetworkTestEnv(t)
	userID := bson.NewObjectID()
	d := &Deposit{
		Network:      NetworkBEP20,
		TxHash:       "0xstray",
		FromAddress:  "0xSender",
		ToAddress:    testBEP20Addr,
		AmountMicros: 3_000_000,
		BlockTime:    time.Now().UTC(),
	}
	if err := e.store.RecordUnmatchedDeposit(ctx, d); err != nil {
		t.Fatal(err)
	}
	list, _, err := e.store.ListDeposits(ctx, DepositUnmatched, pagination.Params{Limit: 10})
	if err != nil || len(list) != 1 {
		t.Fatalf("seed: %v, %d deposits", err, len(list))
	}
	out, err := e.svc.AttributeDeposit(ctx, list[0].ID, userID, "admin@test", "")
	if err != nil {
		t.Fatalf("AttributeDeposit: %v", err)
	}
	if out.Status != DepositCredited {
		t.Errorf("status = %s, want credited", out.Status)
	}
	if len(e.crediter.calls) != 1 || e.crediter.calls[0].Method != "usdt_bep20" ||
		e.crediter.calls[0].Ref != "deposit:0xstray" {
		t.Errorf("credits: %+v", e.crediter.calls)
	}
}
