package payment

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/platform/tron"
)

// fakeReader is a controllable tron.Reader keyed by deposit address.
type fakeReader struct {
	mu       sync.Mutex
	payments map[string]*tron.Payment
	errs     map[string]error
	calls    int
}

func newFakeReader() *fakeReader {
	return &fakeReader{payments: map[string]*tron.Payment{}, errs: map[string]error{}}
}

func (f *fakeReader) FindPayment(_ context.Context, w tron.Watch) (*tron.Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if err := f.errs[w.Address]; err != nil {
		return nil, err
	}
	return f.payments[w.Address], nil
}

func (f *fakeReader) pay(address, txHash string, micros int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.payments[address] = &tron.Payment{TxHash: txHash, From: "TSender", AmountMicros: micros, BlockTime: time.Now().UTC()}
}

func newWatcherEnv(t *testing.T, cfg Config) (*testEnv, *fakeReader, *Watcher) {
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
	reader := newFakeReader()
	e.svc = NewService(e.store, reader, e.crediter, e.notifier, cfg)
	e.svc.SetOrderSettler(e.settler)
	return e, reader, NewWatcher(e.svc, time.Minute)
}

func TestWatcherTickScanClaimSettle(t *testing.T) {
	e, reader, w := newWatcherEnv(t, Config{})
	userID := bson.NewObjectID()
	in, err := e.svc.CreateTopUpIntent(context.Background(), userID, 25, "")
	if err != nil {
		t.Fatal(err)
	}

	// Nothing on-chain yet: tick is a no-op.
	w.tick(context.Background())
	if got := e.store.get(t, in.ID); got.Status != StatusPending {
		t.Fatalf("status = %s before payment, want pending", got.Status)
	}

	// Transfer appears → claimed + settled in one tick.
	reader.pay(in.Address, "tx-abc", 25_000_000)
	w.tick(context.Background())
	got := e.store.get(t, in.ID)
	if got.Status != StatusConfirmed || got.TxHash != "tx-abc" || got.Settlement != SettlementWalletTopUp {
		t.Errorf("intent after tick: %+v", got)
	}
	if len(e.crediter.calls) != 1 || e.crediter.calls[0].Amount != 25 {
		t.Errorf("credits: %+v", e.crediter.calls)
	}

	// Further ticks leave the confirmed intent alone.
	w.tick(context.Background())
	if len(e.crediter.calls) != 1 {
		t.Errorf("confirmed intent re-settled: %+v", e.crediter.calls)
	}
}

func TestWatcherExpiryFailsOrderIntent(t *testing.T) {
	e, _, w := newWatcherEnv(t, Config{IntentExpiry: time.Nanosecond})
	userID, orderID := bson.NewObjectID(), bson.NewObjectID()
	in, err := e.svc.CreateOrderIntent(context.Background(), userID, orderID, 50)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond) // pass the nanosecond expiry

	w.tick(context.Background())

	if got := e.store.get(t, in.ID); got.Status != StatusExpired {
		t.Fatalf("status = %s, want expired", got.Status)
	}
	if reason := e.settler.failed[orderID]; reason != "payment expired" {
		t.Errorf("order fail reason = %q, want 'payment expired'", reason)
	}
	if kinds := e.notifier.kinds(); len(kinds) != 1 || kinds[0] != notification.KindPaymentExpired {
		t.Errorf("notifications: %v", kinds)
	}
}

func TestWatcherLatePaymentCreditsWallet(t *testing.T) {
	e, reader, w := newWatcherEnv(t, Config{IntentExpiry: time.Nanosecond})
	userID, orderID := bson.NewObjectID(), bson.NewObjectID()
	in, err := e.svc.CreateOrderIntent(context.Background(), userID, orderID, 50)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)

	w.tick(context.Background()) // expires the intent + fails the order
	if got := e.store.get(t, in.ID); got.Status != StatusExpired {
		t.Fatalf("precondition: status = %s, want expired", got.Status)
	}

	// The money arrives late (within grace). The order module would refuse
	// fulfillment (already failed) — the payment lands in the wallet.
	e.settler.fulfillErr = ErrOrderNotPayable
	reader.pay(in.Address, "tx-late", 50_000_000)
	w.tick(context.Background())

	got := e.store.get(t, in.ID)
	if got.Status != StatusConfirmed || got.Settlement != SettlementLate {
		t.Errorf("intent after late payment: status=%s settlement=%q", got.Status, got.Settlement)
	}
	if len(e.crediter.calls) != 1 || e.crediter.calls[0].Amount != 50 {
		t.Errorf("credits: %+v", e.crediter.calls)
	}
}

func TestWatcherReaderErrorDoesNotAbortTick(t *testing.T) {
	e, reader, w := newWatcherEnv(t, Config{})
	u := bson.NewObjectID()
	broken, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, "k1")
	if err != nil {
		t.Fatal(err)
	}
	healthy, err := e.svc.CreateTopUpIntent(context.Background(), u, 20, "k2")
	if err != nil {
		t.Fatal(err)
	}
	reader.errs[broken.Address] = errors.New("trongrid 500")
	reader.pay(healthy.Address, "tx-ok", 20_000_000)

	w.tick(context.Background())

	if got := e.store.get(t, healthy.ID); got.Status != StatusConfirmed {
		t.Errorf("healthy intent status = %s, want confirmed (tick aborted on the broken one?)", got.Status)
	}
	if got := e.store.get(t, broken.ID); got.Status != StatusPending {
		t.Errorf("broken intent status = %s, want pending", got.Status)
	}
}

func TestWatcherRetriesStuckSettlement(t *testing.T) {
	e, reader, w := newWatcherEnv(t, Config{})
	userID := bson.NewObjectID()
	in, err := e.svc.CreateTopUpIntent(context.Background(), userID, 25, "")
	if err != nil {
		t.Fatal(err)
	}
	reader.pay(in.Address, "tx-1", 25_000_000)

	// First tick: claim succeeds, wallet is down → settlement fails.
	e.crediter.failN = 1
	w.tick(context.Background())
	if got := e.store.get(t, in.ID); got.Status != StatusConfirming {
		t.Fatalf("status = %s, want confirming", got.Status)
	}

	// Wallet recovers → the retry pass settles it (no new chain claim needed).
	w.tick(context.Background())
	got := e.store.get(t, in.ID)
	if got.Status != StatusConfirmed {
		t.Errorf("status after retry tick = %s, want confirmed", got.Status)
	}
	if len(e.crediter.calls) != 1 {
		t.Errorf("credited %d times, want 1", len(e.crediter.calls))
	}
}

func TestWatcherDuplicateTxHashClaimConflict(t *testing.T) {
	e, reader, w := newWatcherEnv(t, Config{})
	u := bson.NewObjectID()
	a, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, "ka")
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, "kb")
	if err != nil {
		t.Fatal(err)
	}
	// A provider quirk reports the SAME transfer for both intents — the
	// unique txHash guard must credit exactly once.
	reader.pay(a.Address, "tx-same", 10_000_000)
	reader.pay(b.Address, "tx-same", 10_000_000)

	w.tick(context.Background())

	confirmed := 0
	for _, id := range []bson.ObjectID{a.ID, b.ID} {
		if e.store.get(t, id).Status == StatusConfirmed {
			confirmed++
		}
	}
	if confirmed != 1 {
		t.Errorf("confirmed %d intents from one transfer, want 1", confirmed)
	}
	if len(e.crediter.calls) != 1 {
		t.Errorf("credited %d times from one transfer, want 1", len(e.crediter.calls))
	}
}
